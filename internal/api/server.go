package api

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/runner"
	"github.com/latifangren/droidspaces-webui/internal/terminal"
	"github.com/latifangren/droidspaces-webui/web"
)

type Server struct {
	mux         *http.ServeMux
	client      *runner.Client
	termManager *terminal.Manager
	port        int
}

func NewServer(port int) *Server {
	client := runner.NewClient()
	s := &Server{
		mux:         http.NewServeMux(),
		client:      client,
		termManager: terminal.NewManager(client),
		port:        port,
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	// System & Telemetry
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/hardware", s.handleHardware)
	s.mux.HandleFunc("/api/host/exec", s.handleHostExec)
	s.mux.HandleFunc("/api/boot-priority", s.handleBootPriority)
	s.mux.HandleFunc("/api/host/interfaces", s.handleHostInterfaces)
	s.mux.HandleFunc("/api/check", s.handleCheck)
	s.mux.HandleFunc("/api/logs", s.handleLogs)
	s.mux.HandleFunc("/api/settings", s.handleSettings)

	// Containers Lifecycle & Inspection
	s.mux.HandleFunc("/api/containers", s.handleContainers)
	s.mux.HandleFunc("/api/containers/", s.handleContainerAction)

	// Templates & RootFS Store
	s.mux.HandleFunc("/api/templates", s.handleTemplates)
	s.mux.HandleFunc("/api/templates/download", s.handleTemplateDownload)
	s.mux.HandleFunc("/api/templates/delete", s.handleTemplateDelete)

	// Interactive WebSocket Terminal & Persistent Sessions
	s.mux.HandleFunc("/api/terminal/sessions", s.handleTerminalSessions)
	s.mux.HandleFunc("/api/terminal/sessions/", s.handleTerminalSessionDelete)
	s.mux.HandleFunc("/api/ws/terminal", s.handleTerminalWS)

	// Embedded Static Frontend
	s.mux.HandleFunc("/", s.handleStaticEmbed)
}

func (s *Server) handleStaticEmbed(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "index.html"
	} else {
		path = strings.TrimPrefix(path, "/")
	}

	distPath := "dist/" + path
	data, err := web.DistFS.ReadFile(distPath)
	if err != nil {
		data, err = web.DistFS.ReadFile("dist/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		path = "index.html"
	}

	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		switch ext {
		case ".js", ".mjs":
			mimeType = "application/javascript"
		case ".css":
			mimeType = "text/css"
		case ".html":
			mimeType = "text/html; charset=utf-8"
		case ".json":
			mimeType = "application/json"
		case ".svg":
			mimeType = "image/svg+xml"
		case ".woff2":
			mimeType = "font/woff2"
		default:
			mimeType = "application/octet-stream"
		}
	}

	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, string(data))
}

func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]interface{}{
		"success": status >= 200 && status < 300,
	}

	if errMsg != "" {
		resp["error"] = errMsg
	}
	if data != nil {
		resp["data"] = data
	}

	_ = json.NewEncoder(w).Encode(resp)
}
