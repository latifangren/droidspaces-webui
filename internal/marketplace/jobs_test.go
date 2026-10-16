package marketplace

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type executorFunc func(context.Context, string, string, string) (string, error)

func (fn executorFunc) ExecContext(ctx context.Context, container, command, user string) (string, error) {
	return fn(ctx, container, command, user)
}

func TestJobManagerStartSnapshotAndActiveJobExclusion(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	manager := NewJobManager(executorFunc(func(ctx context.Context, container, command, user string) (string, error) {
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
			return "done", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}))

	initial, err := manager.Start("nginx-static", "marketplace-test")
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if initial.Status != JobQueued {
		t.Fatalf("Start snapshot status = %q, want %q", initial.Status, JobQueued)
	}
	if initial.TotalSteps == 0 || initial.CompletedSteps != 0 {
		t.Fatalf("unexpected initial step counts: completed=%d total=%d", initial.CompletedSteps, initial.TotalSteps)
	}

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("executor did not start")
	}
	if _, err := manager.Start("nginx-static", "marketplace-test"); err == nil {
		t.Fatal("expected duplicate active job to be rejected")
	}
	close(release)

	waitForJobStatus(t, manager, initial.ID, JobSucceeded)
	job, ok := manager.Get(initial.ID)
	if !ok {
		t.Fatal("completed job missing from manager")
	}
	if job.CompletedSteps != job.TotalSteps || job.Output == "" {
		t.Fatalf("unexpected completed snapshot: %+v", job)
	}
}

func TestJobManagerConcurrentStartOnlyRunsOneJobPerContainer(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	manager := NewJobManager(executorFunc(func(ctx context.Context, container, command, user string) (string, error) {
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
			return "ok", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}))

	const contenders = 12
	var wg sync.WaitGroup
	wg.Add(contenders)
	results := make(chan error, contenders)
	for range contenders {
		go func() {
			defer wg.Done()
			_, err := manager.Start("nginx-static", "same-container")
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful concurrent starts = %d, want 1", successes)
	}

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("executor did not start")
	}
	close(release)
}

func TestJobManagerFailureSnapshot(t *testing.T) {
	manager := NewJobManager(executorFunc(func(context.Context, string, string, string) (string, error) {
		return "partial output", errors.New("step failed")
	}))
	initial, err := manager.Start("nginx-static", "failure-container")
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	job := waitForJobStatus(t, manager, initial.ID, JobPartialFailure)
	if job.Error != "step failed" || job.Output != "partial output" || job.CurrentStep == "" {
		t.Fatalf("unexpected failed job snapshot: %+v", job)
	}
	if _, err := manager.Start("nginx-static", "failure-container"); err != nil {
		t.Fatalf("container should be eligible for a retry after failure: %v", err)
	}
}

func TestAppendBoundedRetainsOnlyOutputTail(t *testing.T) {
	got := appendBounded("old output", strings.Repeat("x", maxJobOutput*32))
	if len(got) != maxJobOutput {
		t.Fatalf("bounded output length = %d, want %d", len(got), maxJobOutput)
	}
	if got != strings.Repeat("x", maxJobOutput) {
		t.Fatal("large output tail was not retained")
	}

	got = appendBounded(strings.Repeat("a", maxJobOutput), "tail")
	if len(got) != maxJobOutput || !strings.HasSuffix(got, "\ntail") {
		t.Fatalf("combined output does not retain bounded tail: length=%d suffix=%q", len(got), got[len(got)-min(len(got), 8):])
	}
}

func waitForJobStatus(t *testing.T, manager *JobManager, id string, status JobStatus) Job {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		job, ok := manager.Get(id)
		if ok && job.Status == status {
			return job
		}
		time.Sleep(time.Millisecond)
	}
	job, _ := manager.Get(id)
	t.Fatalf("job status = %q, want %q (job: %+v)", job.Status, status, job)
	return Job{}
}
