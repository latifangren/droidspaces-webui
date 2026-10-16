package terminal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/config"
	"github.com/latifangren/droidspaces-webui/internal/model"
	"github.com/latifangren/droidspaces-webui/internal/runner"
)

// Manager coordinates all active detached terminal sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	runner   *runner.Client
}

// NewManager creates a global terminal session manager.
func NewManager(client *runner.Client) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		runner:   client,
	}
}

// generateSessionID generates a cryptographically random session identifier.
func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate terminal session ID: %w", err)
	}
	return "ses_" + hex.EncodeToString(b), nil
}

func (m *Manager) findHostShell() string {
	shells := []string{"/system/bin/sh", "/bin/sh", "/bin/bash", "sh"}
	for _, s := range shells {
		if strings.HasPrefix(s, "/") {
			if _, err := os.Stat(s); err == nil {
				return s
			}
		} else {
			if p, err := exec.LookPath(s); err == nil {
				return p
			}
		}
	}
	return "/bin/sh"
}

// CreateSession allocates a persistent PTY session for host or container target.
func (m *Manager) CreateSession(target, container, user, title string) (*Session, error) {
	if target == "" {
		target = "container"
	}
	if target != "host" && target != "container" {
		return nil, fmt.Errorf("invalid terminal target: %s", target)
	}

	var cmd *exec.Cmd
	binPath := "droidspaces"
	if m.runner != nil {
		binPath = m.runner.BinaryPath()
	}

	if target == "host" {
		sh := m.findHostShell()
		cmd = exec.Command(sh, "-i")
		if title == "" {
			title = "Host Shell (root)"
		}
	} else {
		if !config.SafeContainerName(container) {
			return nil, fmt.Errorf("invalid container name: %s", container)
		}

		if m.runner != nil {
			show, err := m.runner.Show()
			if err != nil {
				return nil, fmt.Errorf("check container status: %w", err)
			}
			isRunning := false
			for _, r := range show.Running {
				if r.Name == container {
					isRunning = true
					break
				}
			}
			if !isRunning {
				log.Printf("[terminal] Auto-starting container %s for terminal session...", container)
				if err := m.runner.Start(model.StartRequest{Name: container}); err != nil {
					return nil, fmt.Errorf("start container %s: %w", container, err)
				}
				const readinessTimeout = 30 * time.Second
				deadline := time.Now().Add(readinessTimeout)
				for time.Now().Before(deadline) {
					ctx, cancel := context.WithDeadline(context.Background(), deadline)
					show, err = m.runner.ShowContext(ctx)
					cancel()
					if err == nil && show != nil {
						for _, running := range show.Running {
							if running.Name == container {
								isRunning = true
								break
							}
						}
					}
					if isRunning {
						break
					}
					if !time.Now().Before(deadline) {
						break
					}
					time.Sleep(250 * time.Millisecond)
				}
				if !isRunning {
					return nil, fmt.Errorf("container %s did not become ready within 30 seconds", container)
				}
			}
		}

		// Droidspaces interactive terminal command is 'enter [user]'
		args := []string{"--name=" + container, "enter"}
		if user != "" {
			args = append(args, user)
		}
		cmd = exec.Command(binPath, args...)

		if title == "" {
			if user != "" && user != "root" {
				title = fmt.Sprintf("%s@%s", user, container)
			} else {
				title = container
			}
		}
	}

	// Environment configuration for standard terminal emulation
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"LANG=en_US.UTF-8",
	)

	var sessionID string
	for attempt := 0; attempt < 3; attempt++ {
		id, err := generateSessionID()
		if err != nil {
			return nil, err
		}
		m.mu.RLock()
		_, exists := m.sessions[id]
		m.mu.RUnlock()
		if !exists {
			sessionID = id
			break
		}
	}
	if sessionID == "" {
		return nil, fmt.Errorf("could not allocate unique terminal session ID")
	}

	s, err := NewSession(sessionID, title, target, container, user, cmd, func(closedID string) {
		time.AfterFunc(5*time.Minute, func() {
			m.mu.Lock()
			delete(m.sessions, closedID)
			m.mu.Unlock()
		})
	})
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	if _, exists := m.sessions[sessionID]; exists {
		m.mu.Unlock()
		_ = s.Close()
		return nil, fmt.Errorf("terminal session ID collision")
	}
	m.sessions[sessionID] = s
	m.mu.Unlock()

	return s, nil
}

// GetSession retrieves an active session by its identifier.
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// ListSessions returns a summary of all active terminal sessions.
func (m *Manager) ListSessions() []model.TerminalSessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]model.TerminalSessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		list = append(list, model.TerminalSessionInfo{
			ID:            s.ID,
			Title:         s.Title,
			Target:        s.Target,
			Container:     s.Container,
			User:          s.User,
			CreatedAt:     s.CreatedAt.Unix(),
			ActiveClients: s.ClientCount(),
			Cols:          s.Cols,
			Rows:          s.Rows,
			IsRunning:     s.IsRunning(),
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt < list[j].CreatedAt
	})

	return list
}

// KillSession explicitly terminates and purges a session.
func (m *Manager) KillSession(id string) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	return s.Close()
}

// CloseAll terminates all active sessions during daemon shutdown.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for _, s := range sessions {
		_ = s.Close()
	}
}
