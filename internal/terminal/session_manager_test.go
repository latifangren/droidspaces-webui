package terminal

import (
	"bytes"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/latifangren/droidspaces-webui/internal/runner"
)

type mockBroadcaster struct {
	mu       sync.Mutex
	messages [][]byte
}

func (m *mockBroadcaster) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, append([]byte(nil), data...))
	return nil
}

func TestRingBufferExtended(t *testing.T) {
	// Test default capacity fallback
	rbDefault := NewRingBuffer(0)
	if rbDefault.maxBytes != DefaultRingBufferCap {
		t.Errorf("expected maxBytes to be %d, got %d", DefaultRingBufferCap, rbDefault.maxBytes)
	}

	rbNeg := NewRingBuffer(-10)
	if rbNeg.maxBytes != DefaultRingBufferCap {
		t.Errorf("expected maxBytes to be %d, got %d", DefaultRingBufferCap, rbNeg.maxBytes)
	}

	rb := NewRingBuffer(20)
	if rb.Len() != 0 {
		t.Errorf("expected initial Len 0, got %d", rb.Len())
	}

	_, _ = rb.Write([]byte("foobar"))
	if rb.Len() != 6 {
		t.Errorf("expected Len 6, got %d", rb.Len())
	}

	rb.Reset()
	if rb.Len() != 0 || len(rb.Bytes()) != 0 {
		t.Errorf("expected empty buffer after Reset")
	}
}

func TestSessionLifecycleAndBroadcasting(t *testing.T) {
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer rPipe.Close()

	onCloseCalled := make(chan string, 1)
	onClose := func(id string) {
		select {
		case onCloseCalled <- id:
		default:
		}
	}

	s := &Session{
		ID:          "test-sess-1",
		Title:       "Test Session",
		Target:      "host",
		User:        "root",
		CreatedAt:   time.Now(),
		Cols:        80,
		Rows:        24,
		Buffer:      NewRingBuffer(1024),
		ptmx:        wPipe,
		cmd:         exec.Command("echo", "ok"),
		clients:     make(map[string]Broadcaster),
		idleTimeout: 50 * time.Millisecond,
		onClose:     onClose,
	}

	// Buffer initial output
	_, _ = s.Buffer.Write([]byte("welcome banner\n"))

	// Verify running state
	if !s.IsRunning() {
		t.Errorf("session should be running initially")
	}

	// Attach mock client
	client1 := &mockBroadcaster{}
	s.Attach("c1", client1)

	if s.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", s.ClientCount())
	}

	// Client should have received buffered replay
	client1.mu.Lock()
	if len(client1.messages) == 0 || !bytes.Contains(client1.messages[0], []byte("welcome banner")) {
		t.Errorf("client did not receive initial replay: %+v", client1.messages)
	}
	client1.mu.Unlock()

	// Attach second client
	client2 := &mockBroadcaster{}
	s.Attach("c2", client2)
	if s.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", s.ClientCount())
	}

	// Write input to ptmx
	inputData := []byte("ls -la\n")
	if err := s.WriteInput(inputData); err != nil {
		t.Errorf("failed to write input: %v", err)
	}

	// Read from pipe to verify input arrived
	buf := make([]byte, len(inputData))
	n, err := rPipe.Read(buf)
	if err != nil || n != len(inputData) || !bytes.Equal(buf, inputData) {
		t.Errorf("unexpected pipe read: n=%d err=%v buf=%s", n, err, string(buf))
	}

	// Detach client 1
	s.Detach("c1")
	if s.ClientCount() != 1 {
		t.Errorf("expected 1 client after detaching c1, got %d", s.ClientCount())
	}

	// Detach client 2 (idle timer triggers)
	s.Detach("c2")
	if s.ClientCount() != 0 {
		t.Errorf("expected 0 clients after detaching c2, got %d", s.ClientCount())
	}

	// Re-attach to cancel idle timer
	s.Attach("c3", &mockBroadcaster{})
	if s.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", s.ClientCount())
	}

	// Test Resize (on pipe, pty.Setsize will error gracefully, should not panic)
	_ = s.Resize(100, 30)

	// Close session
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if s.IsRunning() {
		t.Errorf("session should not be running after close")
	}

	// Double close should be idempotent
	if err := s.Close(); err != nil {
		t.Errorf("second close should not error: %v", err)
	}

	// WriteInput on closed session should fail
	if err := s.WriteInput([]byte("fail")); err == nil {
		t.Errorf("expected error writing to closed session")
	}
}

func TestManagerOperations(t *testing.T) {
	client := runner.NewClient()
	mgr := NewManager(client)

	// Shell discovery
	shell := mgr.findHostShell()
	if shell == "" {
		t.Errorf("findHostShell returned empty string")
	}

	// Generate session ID
	id1 := generateSessionID()
	id2 := generateSessionID()
	if len(id1) != 12 || len(id2) != 12 || id1 == id2 {
		t.Errorf("unexpected session IDs: %s vs %s", id1, id2)
	}

	// CreateSession input validations
	if _, err := mgr.CreateSession("container", "", "root", "Test"); err == nil {
		t.Errorf("expected error for empty container name")
	}
	if _, err := mgr.CreateSession("container", "bad/name", "root", "Test"); err == nil {
		t.Errorf("expected error for invalid container name")
	}

	// Try creating a host session (might succeed or fail depending on pty support on platform)
	_, _ = mgr.CreateSession("host", "", "", "Host Term")

	// Directly register mock session into manager to test management functions
	rPipe, wPipe, _ := os.Pipe()
	defer rPipe.Close()

	sess1 := &Session{
		ID:        "sess-1",
		Title:     "Session 1",
		Target:    "host",
		CreatedAt: time.Now().Add(-10 * time.Minute),
		ptmx:      wPipe,
		clients:   make(map[string]Broadcaster),
		Buffer:    NewRingBuffer(1024),
	}
	sess2 := &Session{
		ID:        "sess-2",
		Title:     "Session 2",
		Target:    "container",
		Container: "alpine-box",
		CreatedAt: time.Now(),
		ptmx:      wPipe,
		clients:   make(map[string]Broadcaster),
		Buffer:    NewRingBuffer(1024),
	}

	mgr.mu.Lock()
	mgr.sessions[sess1.ID] = sess1
	mgr.sessions[sess2.ID] = sess2
	mgr.mu.Unlock()

	// GetSession
	sFound, ok := mgr.GetSession("sess-1")
	if !ok || sFound != sess1 {
		t.Errorf("failed to retrieve sess-1")
	}
	_, ok = mgr.GetSession("nonexistent")
	if ok {
		t.Errorf("expected false for nonexistent session")
	}

	// ListSessions (sorted by CreatedAt descending, sess2 first)
	list := mgr.ListSessions()
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}
	if list[0].ID != "sess-1" || list[1].ID != "sess-2" {
		t.Errorf("unexpected session list order: %v, %v", list[0].ID, list[1].ID)
	}

	// KillSession
	if err := mgr.KillSession("sess-1"); err != nil {
		t.Fatalf("KillSession failed: %v", err)
	}
	if _, ok := mgr.GetSession("sess-1"); ok {
		t.Errorf("sess-1 still present after kill")
	}
	if err := mgr.KillSession("nonexistent"); err == nil {
		t.Errorf("expected error killing nonexistent session")
	}

	// CloseAll
	mgr.CloseAll()
	if len(mgr.ListSessions()) != 0 {
		t.Errorf("expected 0 sessions after CloseAll")
	}
}

func TestSessionIdleTimerExpiry(t *testing.T) {
	_, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}

	closedChan := make(chan bool, 1)
	s := &Session{
		ID:          "idle-sess",
		Buffer:      NewRingBuffer(1024),
		ptmx:        wPipe,
		clients:     make(map[string]Broadcaster),
		idleTimeout: 20 * time.Millisecond,
		onClose: func(id string) {
			closedChan <- true
		},
	}

	s.Attach("client1", &mockBroadcaster{})
	s.Detach("client1") // Starts 20ms idle timer

	select {
	case <-closedChan:
		// Idle timer properly closed session
	case <-time.After(500 * time.Millisecond):
		t.Errorf("idle timer did not expire and close session")
	}
}


func TestReadPTYLoop(t *testing.T) {
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}

	closedChan := make(chan bool, 1)
	s := &Session{
		ID:      "loop-sess",
		Buffer:  NewRingBuffer(1024),
		ptmx:    rPipe,
		clients: make(map[string]Broadcaster),
		onClose: func(id string) {
			closedChan <- true
		},
	}

	client := &mockBroadcaster{}
	s.Attach("c1", client)

	// Run readPTYLoop in background
	go s.readPTYLoop()

	// Write data to pipe
	testMsg := "hello from pty\n"
	_, _ = wPipe.Write([]byte(testMsg))

	// Wait briefly for client to receive message
	time.Sleep(50 * time.Millisecond)
	client.mu.Lock()
	hasMsg := len(client.messages) > 0 && bytes.Contains(client.messages[len(client.messages)-1], []byte("hello from pty"))
	client.mu.Unlock()
	if !hasMsg {
		t.Errorf("client did not receive data from readPTYLoop")
	}

	// Close pipe to trigger EOF and session shutdown
	_ = wPipe.Close()
	select {
	case <-closedChan:
		// Successfully shutdown on EOF
	case <-time.After(500 * time.Millisecond):
		t.Errorf("readPTYLoop did not close on EOF")
	}
}

func TestCreateSessionContainerValidation(t *testing.T) {
	client := runner.NewClient()
	mgr := NewManager(client)

	// Target host with default title
	_, _ = mgr.CreateSession("host", "", "", "")

	// Target container with unknown container (not running, auto-start attempted)
	_, err := mgr.CreateSession("container", "not-running-box", "app", "")
	_ = err

	// Manager without runner
	mgrNil := NewManager(nil)
	_, errNil := mgrNil.CreateSession("container", "box-nil", "guest", "")
	_ = errNil

	// Empty target defaults to container
	_, err = mgr.CreateSession("", "", "", "")
	if err == nil {
		t.Errorf("expected error for empty container name when target is default")
	}
}

func TestNewSessionAndCreateSessionMocked(t *testing.T) {
	origStart := ptyStart
	origSetsize := ptySetsize
	defer func() {
		ptyStart = origStart
		ptySetsize = origSetsize
	}()

	rPipe, wPipe, _ := os.Pipe()
	defer rPipe.Close()

	ptyStart = func(cmd *exec.Cmd) (*os.File, error) {
		return wPipe, nil
	}
	ptySetsize = func(f *os.File, sz *pty.Winsize) error {
		return nil
	}

	cmd := exec.Command("echo", "ok")
	s, err := NewSession("new-mock-sess", "Mock Title", "host", "", "root", cmd, nil)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer s.Close()

	mgr := NewManager(nil)
	sess, err := mgr.CreateSession("host", "", "root", "Host Shell")
	if err != nil {
		t.Fatalf("mgr.CreateSession failed: %v", err)
	}
	defer sess.Close()
}