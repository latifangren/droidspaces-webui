package api

import (
	"encoding/json"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/latifangren/droidspaces-webui/internal/auth"
	"github.com/latifangren/droidspaces-webui/internal/hardware"
	"github.com/latifangren/droidspaces-webui/internal/marketplace"
	"github.com/latifangren/droidspaces-webui/internal/model"
	"github.com/latifangren/droidspaces-webui/internal/runner"
	"github.com/latifangren/droidspaces-webui/internal/terminal"
	"github.com/latifangren/droidspaces-webui/web"
)

type Server struct {
	mux             *http.ServeMux
	client          *runner.Client
	termManager     *terminal.Manager
	marketplaceJobs *marketplace.JobManager
	marketplaceShow func() (*model.ShowResult, error)
	authMgr         *auth.AuthManager
	loginLimiter    *rateLimiter
	portMu          sync.RWMutex
	port            int
}

func (s *Server) GetPort() int {
	s.portMu.RLock()
	defer s.portMu.RUnlock()
	return s.port
}

func (s *Server) SetPort(port int) {
	s.portMu.Lock()
	defer s.portMu.Unlock()
	s.port = port
}

func NewServer(port int) *Server {
	client := runner.NewClient()
	s := &Server{
		mux:             http.NewServeMux(),
		client:          client,
		termManager:     terminal.NewManager(client),
		marketplaceJobs: marketplace.NewJobManager(client),
		marketplaceShow: client.Show,
		authMgr:         auth.NewAuthManager(),

		loginLimiter: newLoginRateLimiter(),
		port:         port,
	}
	s.routes()
	StartLogAutoTruncate()
	hardware.StartContainerCPUSampler(func() map[string]int {
		show, err := client.Show()
		if err != nil || show == nil {
			return nil
		}
		res := make(map[string]int)
		for _, r := range show.Running {
			if r.PID > 0 && r.Name != "" {
				res[r.Name] = r.PID
			}
		}
		return res
	})
	return s
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Enforce authentication on all /api/* routes except public auth status and login
		if strings.HasPrefix(path, "/api/") {
			if path != "/api/auth/status" && path != "/api/auth/login" {
				authenticated := s.authMgr.AuthenticateRequest(r)
				if !authenticated {
					s.sendJSON(w, http.StatusUnauthorized, nil, "unauthorized")
					return
				}
				if path == "/api/auth/change-password" || path == "/api/auth/logout" {
					s.mux.ServeHTTP(w, r)
					return
				}
				if s.authMgr.IsUsingDefaultPassword() {
					s.sendJSON(w, http.StatusForbidden, nil, "change the default password before using the API")
					return
				}
			}
		}
		s.mux.ServeHTTP(w, r)
	})
}

func (s *Server) routes() {
	// Authentication
	s.mux.HandleFunc("/api/auth/status", s.handleAuthStatus)
	s.mux.HandleFunc("/api/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("/api/auth/change-password", s.handleChangePassword)

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
	s.mux.HandleFunc("/api/containers/restore", s.handleContainerRestore)
	s.mux.HandleFunc("/api/containers", s.handleContainers)
	s.mux.HandleFunc("/api/containers/", s.handleContainerAction)

	// Global Backups Management
	s.mux.HandleFunc("/api/backups", s.handleAllBackups)
	s.mux.HandleFunc("/api/backups/download", s.handleBackupDownload)

	// Templates & RootFS Store
	s.mux.HandleFunc("/api/templates", s.handleTemplates)
	s.mux.HandleFunc("/api/templates/download", s.handleTemplateDownload)
	s.mux.HandleFunc("/api/templates/delete", s.handleTemplateDelete)
	s.mux.HandleFunc("/api/marketplace/setup-jobs", s.handleMarketplaceSetupJobs)
	s.mux.HandleFunc("/api/marketplace/setup-jobs/", s.handleMarketplaceSetupJob)

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

	if path == "index.html" || strings.HasSuffix(path, ".html") {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	} else if strings.HasPrefix(path, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}

	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
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
