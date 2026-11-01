package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/marketplace"
	"github.com/latifangren/droidspaces-webui/internal/model"
)

type marketplaceExecutorFunc func(context.Context, string, string, string) (string, error)

func (fn marketplaceExecutorFunc) ExecContext(ctx context.Context, container, command, user string) (string, error) {
	return fn(ctx, container, command, user)
}

func TestMarketplaceSetupEndpointValidationAndInspectionFailures(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()
	var inspections int
	s.marketplaceShow = func() (*model.ShowResult, error) {
		inspections++
		return &model.ShowResult{Running: []model.ContainerSummary{{Name: "web", RootFS: "alpine-3.20"}}}, nil
	}

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "malformed json", body: "{"},
		{name: "unknown preset", body: `{"app_id":"unknown","container_name":"web"}`},
		{name: "invalid name", body: `{"app_id":"nginx-static","container_name":"../web"}`},
		{name: "trailing value", body: `{"app_id":"nginx-static","container_name":"web"} {}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/marketplace/setup-jobs", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
			}
		})
	}
	if inspections != 0 {
		t.Fatalf("container inspection called %d times for invalid requests", inspections)
	}

	large := `{"app_id":"nginx-static","container_name":"web","padding":"` + strings.Repeat("x", 17*1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/marketplace/setup-jobs", strings.NewReader(large))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("oversized request status = %d, want 400", w.Code)
	}
}

func TestMarketplaceSetupEndpointRootFSAndJobFailures(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()
	s.marketplaceJobs = marketplace.NewJobManager(marketplaceExecutorFunc(func(context.Context, string, string, string) (string, error) {
		return "", errors.New("setup failed")
	}))

	s.marketplaceShow = func() (*model.ShowResult, error) {
		return &model.ShowResult{Running: []model.ContainerSummary{{Name: "web", RootFS: "/data/local/Droidspaces/rootfs/alpine-3.20"}}}, nil
	}
	w := doRequest(h, token, http.MethodPost, "/api/marketplace/setup-jobs", map[string]string{"app_id": "nginx-static", "container_name": "web"})
	if w.Code != http.StatusAccepted {
		t.Fatalf("valid rootfs path status = %d, want 202: %s", w.Code, w.Body.String())
	}

	var accepted struct {
		Data marketplace.Job `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.Data.ID == "" {
		t.Fatal("accepted job response has no job ID")
	}

	poll := doRequest(h, token, http.MethodGet, "/api/marketplace/setup-jobs/"+accepted.Data.ID, nil)
	if poll.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200", poll.Code)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		poll = doRequest(h, token, http.MethodGet, "/api/marketplace/setup-jobs/"+accepted.Data.ID, nil)
		var result struct {
			Data marketplace.Job `json:"data"`
		}
		if err := json.Unmarshal(poll.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Data.Status == marketplace.JobPartialFailure {
			if result.Data.Error != "setup failed" {
				t.Fatalf("job error = %q", result.Data.Error)
			}
			break
		}
		time.Sleep(time.Millisecond)
	}

	for _, tc := range []struct {
		name string
		show *model.ShowResult
		want int
	}{
		{name: "wrong distro", show: &model.ShowResult{Running: []model.ContainerSummary{{Name: "web", RootFS: "debian-12"}}}, want: http.StatusConflict},
		{name: "stopped", show: &model.ShowResult{}, want: http.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.marketplaceShow = func() (*model.ShowResult, error) { return tc.show, nil }
			w := doRequest(h, token, http.MethodPost, "/api/marketplace/setup-jobs", map[string]string{"app_id": "nginx-static", "container_name": "web"})
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}

	s.marketplaceShow = func() (*model.ShowResult, error) { return nil, errors.New("runner unavailable") }
	w = doRequest(h, token, http.MethodPost, "/api/marketplace/setup-jobs", map[string]string{"app_id": "nginx-static", "container_name": "web"})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("inspection failure status = %d, want 500", w.Code)
	}
}

func TestMarketplaceSetupEndpointConcurrentDuplicate(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	s.marketplaceJobs = marketplace.NewJobManager(marketplaceExecutorFunc(func(ctx context.Context, container, command, user string) (string, error) {
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
	s.marketplaceShow = func() (*model.ShowResult, error) {
		return &model.ShowResult{Running: []model.ContainerSummary{{Name: "web", RootFS: "alpine-3.20"}}}, nil
	}

	const contenders = 8
	var wg sync.WaitGroup
	wg.Add(contenders)
	statuses := make(chan int, contenders)
	for range contenders {
		go func() {
			defer wg.Done()
			w := doRequest(h, token, http.MethodPost, "/api/marketplace/setup-jobs", map[string]string{"app_id": "nginx-static", "container_name": "web"})
			statuses <- w.Code
		}()
	}
	wg.Wait()
	close(statuses)
	accepted, conflicts := 0, 0
	for status := range statuses {
		switch status {
		case http.StatusAccepted:
			accepted++
		case http.StatusConflict:
			conflicts++
		default:
			t.Errorf("unexpected concurrent request status %d", status)
		}
	}
	if accepted != 1 || conflicts != contenders-1 {
		t.Fatalf("accepted=%d conflicts=%d, want 1 and %d", accepted, conflicts, contenders-1)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("setup executor did not start")
	}
	close(release)
}
