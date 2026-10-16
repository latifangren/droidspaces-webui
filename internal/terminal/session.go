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

type closeableBroadcaster interface {
	Close()
}

// winsize holds client-reported terminal window dimensions.
type winsize struct {
	Cols uint16
	Rows uint16
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
	clientSizes map[string]winsize
	mu          sync.RWMutex
	outputMu    sync.Mutex
	detachTimer *time.Timer
	idleTimeout time.Duration
	closed      bool
	onClose     func(id string)
}

var (
	ptyStart   = pty.Start
	ptySetsize = pty.Setsize
)

// MockPTY allows test suites to mock PTY creation.
func MockPTY(fn func(cmd *exec.Cmd) (*os.File, error), setsize func(f *os.File, sz *pty.Winsize) error) func() {
	origStart := ptyStart
	origSetsize := ptySetsize
	ptyStart = fn
	ptySetsize = setsize
	return func() {
		ptyStart = origStart
		ptySetsize = origSetsize
	}
}

// NewSession starts a process in a PTY and sets up the streaming session.
func NewSession(id, title, target, container, user string, cmd *exec.Cmd, onClose func(id string)) (*Session, error) {
	ptmx, err := ptyStart(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}

	// Default window size
	_ = ptySetsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

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
		clientSizes: make(map[string]winsize),
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
		s.mu.RLock()
		ptmx := s.ptmx
		s.mu.RUnlock()
		if ptmx == nil {
			return
		}
		n, err := ptmx.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			s.outputMu.Lock()
			_, _ = s.Buffer.Write(chunk)

			s.mu.RLock()
			activeClients := make(map[string]Broadcaster, len(s.clients))
			for id, client := range s.clients {
				activeClients[id] = client
			}
			s.mu.RUnlock()

			for id, client := range activeClients {
				if err := client.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
					s.detachFailedClient(id, client)
				}
			}
			s.outputMu.Unlock()
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

// Attach registers a WebSocket client and queues scrollback before live output.
func (s *Session) Attach(clientID string, client Broadcaster) {
	s.outputMu.Lock()
	s.mu.Lock()
	closed := s.closed
	if !closed {
		if s.detachTimer != nil {
			s.detachTimer.Stop()
			s.detachTimer = nil
		}
		s.clients[clientID] = client
	}
	s.mu.Unlock()

	replayData := s.Buffer.Bytes()
	var err error
	if len(replayData) > 0 {
		err = client.WriteMessage(websocket.BinaryMessage, replayData)
	}
	if closed {
		err = client.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[33m[Process exited]\x1b[0m\r\n"))
	}
	if err != nil {
		s.detachFailedClient(clientID, client)
	}
	s.outputMu.Unlock()
}

func (s *Session) detachFailedClient(clientID string, client Broadcaster) {
	s.mu.Lock()
	if current, ok := s.clients[clientID]; ok && current == client {
		delete(s.clients, clientID)
		delete(s.clientSizes, clientID)
		if len(s.clients) == 0 && !s.closed {
			s.LastDetachedAt = time.Now()
			if s.detachTimer != nil {
				s.detachTimer.Stop()
			}
			s.detachTimer = time.AfterFunc(s.idleTimeout, func() { _ = s.Close() })
		}
	}
	s.mu.Unlock()
	if c, ok := client.(closeableBroadcaster); ok {
		c.Close()
	}
}

// Detach unregisters a client. If zero clients remain, begins idle cleanup timer.
func (s *Session) Detach(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, clientID)
	if s.clientSizes != nil {
		delete(s.clientSizes, clientID)
	}
	if len(s.clients) > 0 {
		_ = s.applySmallestSizeLocked()
	}
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

// Resize updates the terminal window dimensions for a client and adopts
// the smallest dimensions across all currently connected clients.
func (s *Session) Resize(clientID string, cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.ptmx == nil {
		return fmt.Errorf("session is closed")
	}

	if cols == 0 || rows == 0 {
		return fmt.Errorf("invalid terminal dimensions: cols and rows must be > 0")
	}

	if s.clientSizes == nil {
		s.clientSizes = make(map[string]winsize)
	}

	s.clientSizes[clientID] = winsize{Cols: cols, Rows: rows}
	return s.applySmallestSizeLocked()
}

// applySmallestSizeLocked computes the smallest dimensions across active clients
// and applies them to the PTY. Caller must hold s.mu.
func (s *Session) applySmallestSizeLocked() error {
	if s.closed || s.ptmx == nil {
		return fmt.Errorf("session is closed")
	}

	if len(s.clientSizes) == 0 {
		return nil
	}

	var minCols, minRows uint16
	found := false

	for id, sz := range s.clientSizes {
		if len(s.clients) > 0 && id != "" {
			if _, ok := s.clients[id]; !ok {
				continue
			}
		}
		if sz.Cols == 0 || sz.Rows == 0 {
			continue
		}
		if !found {
			minCols = sz.Cols
			minRows = sz.Rows
			found = true
		} else {
			if sz.Cols < minCols {
				minCols = sz.Cols
			}
			if sz.Rows < minRows {
				minRows = sz.Rows
			}
		}
	}

	if !found {
		return nil
	}

	s.Cols = minCols
	s.Rows = minRows
	return ptySetsize(s.ptmx, &pty.Winsize{
		Rows: minRows,
		Cols: minCols,
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
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
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

	// Queue the exit notice before closing socket writers so the terminal UI can
	// mark the session as dead instead of silently reconnecting.
	for _, client := range s.clients {
		if err := client.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[33m[Session closed]\x1b[0m\r\n")); err != nil {
			if c, ok := client.(closeableBroadcaster); ok {
				c.Close()
			}
			continue
		}
		if c, ok := client.(interface{ CloseAfterQueuedMessages() }); ok {
			c.CloseAfterQueuedMessages()
		} else if c, ok := client.(closeableBroadcaster); ok {
			c.Close()
		}
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
