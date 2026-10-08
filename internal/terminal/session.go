package terminal

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// DefaultIdleTimeout is how long an unattached background session lives before cleanup.
const DefaultIdleTimeout = 60 * time.Minute

// Broadcaster abstracts the client connection writing terminal bytes.
type Broadcaster interface {
	WriteMessage(messageType int, data []byte) error
}

// Session represents a persistent Pseudo-Terminal process and its attached clients.
type Session struct {
	ID             string
	Title          string
	Target         string // "container" | "host"
	Container      string
	User           string
	CreatedAt      time.Time
	LastDetachedAt time.Time
	Cols           uint16
	Rows           uint16

	Buffer      *RingBuffer
	ptmx        *os.File
	cmd         *exec.Cmd
	clients     map[string]Broadcaster
	mu          sync.RWMutex
	detachTimer *time.Timer
	idleTimeout time.Duration
	closed      bool
	onClose     func(id string)
}

// NewSession starts a process in a PTY and sets up the streaming session.
func NewSession(id, title, target, container, user string, cmd *exec.Cmd, onClose func(id string)) (*Session, error) {
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}

	// Default window size
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	s := &Session{
		ID:          id,
		Title:       title,
		Target:      target,
		Container:   container,
		User:        user,
		CreatedAt:   time.Now(),
		Cols:        80,
		Rows:        24,
		Buffer:      NewRingBuffer(DefaultRingBufferCap),
		ptmx:        ptmx,
		cmd:         cmd,
		clients:     make(map[string]Broadcaster),
		idleTimeout: DefaultIdleTimeout,
		onClose:     onClose,
	}

	// Start reading PTY master stdout/stderr
	go s.readPTYLoop()

	return s, nil
}

// readPTYLoop continuously reads from PTY, buffers data, and broadcasts to attached clients.
func (s *Session) readPTYLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			// 1. Append to circular ring buffer for replay
			_, _ = s.Buffer.Write(chunk)

			// 2. Broadcast to all active attached WebSocket participants
			s.mu.RLock()
			for cid, client := range s.clients {
				if err := client.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
					log.Printf("[terminal:%s] client %s write error: %v", s.ID, cid, err)
				}
			}
			s.mu.RUnlock()
		}

		if err != nil {
			if err != io.EOF {
				log.Printf("[terminal:%s] pty exited: %v", s.ID, err)
			}
			s.Close()
			return
		}
	}
}

// Attach registers a WebSocket client to this session and replays the current output buffer.
func (s *Session) Attach(clientID string, client Broadcaster) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}

	// Cancel idle detach timer when a client reconnects
	if s.detachTimer != nil {
		s.detachTimer.Stop()
		s.detachTimer = nil
	}

	s.clients[clientID] = client
	s.mu.Unlock()

	// Replay buffered scrollback immediately to the joining client
	replayData := s.Buffer.Bytes()
	if len(replayData) > 0 {
		_ = client.WriteMessage(websocket.BinaryMessage, replayData)
	}
}

// Detach unregisters a client. If zero clients remain, begins idle cleanup timer.
func (s *Session) Detach(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, clientID)
	if len(s.clients) == 0 && !s.closed {
		s.LastDetachedAt = time.Now()
		// Start idle detach timer to keep process alive in background
		if s.detachTimer != nil {
			s.detachTimer.Stop()
		}
		s.detachTimer = time.AfterFunc(s.idleTimeout, func() {
			log.Printf("[terminal:%s] idle timeout reached (%v with 0 clients), cleaning up", s.ID, s.idleTimeout)
			s.Close()
		})
	}
}

// WriteInput sends keystrokes/data from client into the PTY stdin.
func (s *Session) WriteInput(data []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed || s.ptmx == nil {
		return fmt.Errorf("session is closed")
	}
	_, err := s.ptmx.Write(data)
	return err
}

// Resize updates the terminal window dimensions.
func (s *Session) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.ptmx == nil {
		return fmt.Errorf("session is closed")
	}

	s.Cols = cols
	s.Rows = rows
	return pty.Setsize(s.ptmx, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// ClientCount returns how many active clients are watching this session.
func (s *Session) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// IsRunning reports whether the session is still active.
func (s *Session) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.closed
}

// Close terminates the session, kills the process, and closes the PTY.
func (s *Session) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true

	if s.detachTimer != nil {
		s.detachTimer.Stop()
		s.detachTimer = nil
	}

	// Close attached clients
	for _, client := range s.clients {
		_ = client.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[33m[Session closed]\x1b[0m\r\n"))
	}
	s.clients = make(map[string]Broadcaster)

	if s.ptmx != nil {
		_ = s.ptmx.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
	s.mu.Unlock()

	if s.onClose != nil {
		s.onClose(s.ID)
	}
	return nil
}
