package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/latifangren/droidspaces-webui/internal/terminal"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for WebUI
	},
}

type resizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type wsClientBroadcaster struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsClientBroadcaster) WriteMessage(messageType int, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(messageType, data)
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
	broadcaster := &wsClientBroadcaster{conn: conn}

	// Attach client: immediately replays the 512 KB buffer and stops idle timer
	sess.Attach(clientID, broadcaster)
	defer sess.Detach(clientID)

	// WebSocket input loop -> writes keystrokes into PTY or resizes terminal
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Handle resize message
		if msgType == websocket.TextMessage && len(msg) > 0 && msg[0] == '{' {
			var rmsg resizeMessage
			if err := json.Unmarshal(msg, &rmsg); err == nil && rmsg.Type == "resize" {
				_ = sess.Resize(rmsg.Cols, rmsg.Rows)
				continue
			}
		}

		if err := sess.WriteInput(msg); err != nil {
			break
		}
	}
}
