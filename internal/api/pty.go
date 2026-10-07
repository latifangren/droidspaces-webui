package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
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

func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[terminal-ws] upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	target := r.URL.Query().Get("target")
	containerName := r.URL.Query().Get("container")
	user := r.URL.Query().Get("user")

	var cmd *exec.Cmd
	if target == "container" && containerName != "" {
		args := []string{"--name=" + containerName, "enter"}
		if user != "" {
			args = append(args, user)
		}
		cmd = exec.Command(s.client.BinaryPath(), args...)
	} else {
		// Host root shell
		shell := "/system/bin/sh"
		if _, err := os.Stat(shell); err != nil {
			shell = "/bin/sh"
		}
		cmd = exec.Command(shell)
	}

	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"HOME=/root",
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31m[-] Failed to initialize PTY: "+err.Error()+"\x1b[0m\r\n"))
		return
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	var once sync.Once
	closeSession := func() {
		once.Do(func() {
			_ = ptmx.Close()
			_ = conn.Close()
		})
	}

	// PTY output -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				closeSession()
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				closeSession()
				return
			}
		}
	}()

	// WebSocket input -> PTY
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			closeSession()
			break
		}

		// Handle resize message
		if msgType == websocket.TextMessage && len(msg) > 0 && msg[0] == '{' {
			var rmsg resizeMessage
			if err := json.Unmarshal(msg, &rmsg); err == nil && rmsg.Type == "resize" && rmsg.Cols > 0 && rmsg.Rows > 0 {
				_ = pty.Setsize(ptmx, &pty.Winsize{
					Rows: rmsg.Rows,
					Cols: rmsg.Cols,
				})
				continue
			}
		}

		if _, err := ptmx.Write(msg); err != nil {
			closeSession()
			break
		}
	}
}
