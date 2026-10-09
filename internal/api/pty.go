package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/latifangren/droidspaces-webui/internal/terminal"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Non-browser clients (curl, native apps)
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		// Verify origin host matches request host to prevent CSWSH attacks
		return strings.EqualFold(u.Host, r.Host)
	},
}
type resizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type wsClientBroadcaster struct {
	conn      *websocket.Conn
	sendChan  chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func newWSClientBroadcaster(conn *websocket.Conn) *wsClientBroadcaster {
	b := &wsClientBroadcaster{
		conn:     conn,
		sendChan: make(chan []byte, 256),
		done:     make(chan struct{}),
	}
	go b.writerLoop()
	return b
}

func (w *wsClientBroadcaster) writerLoop() {
	for {
		select {
		case msg, ok := <-w.sendChan:
			if !ok {
				return
			}
			_ = w.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := w.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				w.Close()
				return
			}
		case <-w.done:
			return
		}
	}
}

func (w *wsClientBroadcaster) WriteMessage(messageType int, data []byte) error {
	select {
	case <-w.done:
		return fmt.Errorf("client broadcaster closed")
	default:
	}

	copied := make([]byte, len(data))
	copy(copied, data)

	select {
	case w.sendChan <- copied:
		return nil
	default:
		// Buffer full: drop chunk to prevent slow client from halting container PTY
		return fmt.Errorf("slow client buffer overflow")
	}
}

func (w *wsClientBroadcaster) Close() {
	w.closeOnce.Do(func() {
		close(w.done)
		if w.conn != nil {
			_ = w.conn.Close()
		}
	})
}

func generateClientID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("cli_%x", hex.EncodeToString(b))
}

func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[terminal-ws] upgrade error: %v", err)
		return
	}
	defer conn.Close()

	sessionID := r.URL.Query().Get("session")
	var sess *terminal.Session

	if sessionID != "" {
		// Reattach to existing persistent session
		existing, ok := s.termManager.GetSession(sessionID)
		if !ok {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31m[-] Session not found: "+sessionID+"\x1b[0m\r\n"))
			return
		}
		sess = existing
	} else {
		// Auto-spawn a new persistent session
		target := r.URL.Query().Get("target")
		container := r.URL.Query().Get("container")
		user := r.URL.Query().Get("user")
		title := r.URL.Query().Get("title")

		created, err := s.termManager.CreateSession(target, container, user, title)
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31m[-] Failed to initialize PTY: "+err.Error()+"\x1b[0m\r\n"))
			return
		}
		sess = created
	}

	clientID := generateClientID()
	broadcaster := newWSClientBroadcaster(conn)
	defer broadcaster.Close()

	// Attach client: immediately replays the 512 KB buffer and stops idle timer
	sess.Attach(clientID, broadcaster)
	defer sess.Detach(clientID)

	// WebSocket input loop -> writes keystrokes into PTY or resizes terminal
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if len(msg) == 0 {
			continue
		}

		// Binary prefix framing:
		// '1' or 0x01: Resize command
		// '0' or 0x00: Stdin data
		switch msg[0] {
		case '1', 0x01:
			var rmsg resizeMessage
			if err := json.Unmarshal(msg[1:], &rmsg); err == nil && rmsg.Cols > 0 && rmsg.Rows > 0 {
				_ = sess.Resize(rmsg.Cols, rmsg.Rows)
			}
			continue
		case '0', 0x00:
			if err := sess.WriteInput(msg[1:]); err != nil {
				break
			}
			continue
		default:
			// Fallback compatibility for clients without prefix
			if msgType == websocket.TextMessage && msg[0] == '{' {
				var rmsg resizeMessage
				if err := json.Unmarshal(msg, &rmsg); err == nil && rmsg.Type == "resize" && rmsg.Cols > 0 && rmsg.Rows > 0 {
					_ = sess.Resize(rmsg.Cols, rmsg.Rows)
					continue
				}
			}
			if err := sess.WriteInput(msg); err != nil {
				break
			}
		}
	}
}
