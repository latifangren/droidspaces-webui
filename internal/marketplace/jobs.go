package marketplace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	jobTimeout    = 20 * time.Minute
	maxJobOutput  = 16 * 1024
	maxStoredJobs = 64
	jobRetention  = 2 * time.Hour
)

type JobStatus string

const (
	JobQueued         JobStatus = "queued"
	JobRunning        JobStatus = "running"
	JobSucceeded      JobStatus = "succeeded"
	JobPartialFailure JobStatus = "partial_failure"
)

type Job struct {
	ID             string    `json:"id"`
	PresetID       string    `json:"preset_id"`
	ContainerName  string    `json:"container_name"`
	Status         JobStatus `json:"status"`
	CurrentStep    string    `json:"current_step,omitempty"`
	CompletedSteps int       `json:"completed_steps"`
	TotalSteps     int       `json:"total_steps"`
	Output         string    `json:"output,omitempty"`
	Error          string    `json:"error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Executor interface {
	ExecContext(ctx context.Context, container, command, user string) (string, error)
}

type boundedExecutor interface {
	ExecContextBounded(ctx context.Context, container, command, user string, maxOutput int) (string, error)
}

type JobManager struct {
	mu           sync.RWMutex
	jobs         map[string]*Job
	activeByName map[string]string
	executor     Executor
}

func NewJobManager(executor Executor) *JobManager {
	return &JobManager{
		jobs:         make(map[string]*Job),
		activeByName: make(map[string]string),
		executor:     executor,
	}
}

func (m *JobManager) Start(presetID, container string) (Job, error) {
	preset, ok := ByID(presetID)
	if !ok {
		return Job{}, fmt.Errorf("unknown app preset")
	}
	if err := ValidateName(container); err != nil {
		return Job{}, err
	}
	if m.executor == nil {
		return Job{}, fmt.Errorf("marketplace setup executor is unavailable")
	}
	if len(preset.SetupSteps) == 0 {
		return Job{}, fmt.Errorf("app preset has no setup steps")
	}

	id, err := generateJobID()
	if err != nil {
		return Job{}, err
	}
	now := time.Now().UTC()
	job := &Job{
		ID:            id,
		PresetID:      preset.ID,
		ContainerName: container,
		Status:        JobQueued,
		TotalSteps:    len(preset.SetupSteps),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	m.mu.Lock()
	m.cleanupLocked(now)
	if _, exists := m.activeByName[container]; exists {
		m.mu.Unlock()
		return Job{}, fmt.Errorf("setup already running for container %q", container)
	}
	if len(m.jobs) >= maxStoredJobs {
		m.mu.Unlock()
		return Job{}, fmt.Errorf("too many marketplace jobs; wait for existing jobs to expire")
	}
	m.jobs[id] = job
	m.activeByName[container] = id
	snapshot := cloneJob(job)
	m.mu.Unlock()

	go m.run(id, preset)
	return snapshot, nil
}

func (m *JobManager) Get(id string) (Job, bool) {
	m.mu.RLock()
	stored, ok := m.jobs[id]
	if !ok {
		m.mu.RUnlock()
		return Job{}, false
	}
	job := cloneJob(stored)
	m.mu.RUnlock()
	return job, true
}

func (m *JobManager) GetForContainer(container, id string) (Job, bool) {
	job, ok := m.Get(id)
	if !ok || job.ContainerName != container {
		return Job{}, false
	}
	return job, true
}

func (m *JobManager) run(id string, preset AppPreset) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	for i, step := range preset.SetupSteps {
		m.update(id, func(job *Job) {
			job.Status = JobRunning
			job.CurrentStep = step.Name
		})

		var output string
		var err error
		if executor, ok := m.executor.(boundedExecutor); ok {
			output, err = executor.ExecContextBounded(ctx, m.containerForJob(id), step.Command, "root", maxJobOutput)
		} else {
			output, err = m.executor.ExecContext(ctx, m.containerForJob(id), step.Command, "root")
		}
		m.update(id, func(job *Job) {
			job.Output = appendBounded(job.Output, output)
		})
		if err != nil {
			m.finish(id, JobPartialFailure, step.Name, sanitizeError(err.Error()))
			return
		}
		m.update(id, func(job *Job) {
			job.CompletedSteps = i + 1
		})
	}
	m.finish(id, JobSucceeded, "", "")
}

func (m *JobManager) containerForJob(id string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if job := m.jobs[id]; job != nil {
		return job.ContainerName
	}
	return ""
}

func (m *JobManager) update(id string, fn func(*Job)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job := m.jobs[id]; job != nil {
		fn(job)
		job.UpdatedAt = time.Now().UTC()
	}
}

func (m *JobManager) finish(id string, status JobStatus, failedStep, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[id]
	if job == nil {
		return
	}
	job.Status = status
	job.CurrentStep = failedStep
	job.Error = message
	job.UpdatedAt = time.Now().UTC()
	delete(m.activeByName, job.ContainerName)
}

func (m *JobManager) cleanupLocked(now time.Time) {
	for id, job := range m.jobs {
		if !isTerminal(job.Status) || now.Sub(job.UpdatedAt) < jobRetention {
			continue
		}
		delete(m.jobs, id)
		if m.activeByName[job.ContainerName] == id {
			delete(m.activeByName, job.ContainerName)
		}
	}
}

func isTerminal(status JobStatus) bool {
	return status == JobSucceeded || status == JobPartialFailure
}

func cloneJob(job *Job) Job {
	return *job
}

func appendBounded(current, next string) string {
	if next == "" {
		return current
	}

	// Preserve only the tail needed for the job snapshot. Avoid concatenating
	// unbounded executor output before truncating it.
	if len(next) >= maxJobOutput {
		return next[len(next)-maxJobOutput:]
	}

	separator := ""
	if current != "" {
		separator = "\n"
	}
	remaining := maxJobOutput - len(next) - len(separator)
	if remaining <= 0 {
		return next
	}
	if len(current) > remaining {
		current = current[len(current)-remaining:]
	}
	return current + separator + next
}

func sanitizeError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 2048 {
		message = message[len(message)-2048:]
	}
	return message
}

func generateJobID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate marketplace job ID: %w", err)
	}
	return "job_" + hex.EncodeToString(b), nil
}
