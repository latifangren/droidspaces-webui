package terminal

import (
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

// generateSessionID generates a short, unique session identifier.
func generateSessionID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "ses_" + hex.EncodeToString(b)
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
		if container == "" {
			return nil, fmt.Errorf("container name is required for container target")
		}

		// Ensure container is running before entering
		if m.runner != nil {
			show, err := m.runner.Show()
			if err == nil {
				isRunning := false
				for _, r := range show.Running {
					if r.Name == container {
						isRunning = true
						break
					}
				}
				if !isRunning {
					log.Printf("[terminal] Auto-starting container %s for terminal session...", container)
					_ = m.runner.Start(model.StartRequest{Name: container})
					time.Sleep(500 * time.Millisecond)
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

	sessionID := generateSessionID()

	s, err := NewSession(sessionID, title, target, container, user, cmd, func(closedID string) {
		// Retain session in registry for 5 minutes after process exit so scrollback can still be viewed
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
