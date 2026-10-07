package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/hardware"
	"github.com/latifangren/droidspaces-webui/internal/model"
	"github.com/latifangren/droidspaces-webui/internal/runner"
	"github.com/latifangren/droidspaces-webui/internal/templates"
	"github.com/latifangren/droidspaces-webui/web"
)

type Server struct {
	client *runner.Client
	mux    *http.ServeMux
	port   int
}

func NewServer(port int) *Server {
	s := &Server{
		client: runner.NewClient(),
		mux:    http.NewServeMux(),
		port:   port,
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/hardware", s.handleHardware)
	s.mux.HandleFunc("/api/host/exec", s.handleHostExec)
	s.mux.HandleFunc("/api/containers", s.handleContainers)
	s.mux.HandleFunc("/api/containers/", s.handleContainerAction)
	s.mux.HandleFunc("/api/templates", s.handleTemplates)
	s.mux.HandleFunc("/api/templates/download", s.handleTemplateDownload)
	s.mux.HandleFunc("/api/templates/delete", s.handleTemplateDelete)
	s.mux.HandleFunc("/api/check", s.handleCheck)
	s.mux.HandleFunc("/api/logs", s.handleLogs)
	s.mux.HandleFunc("/api/settings", s.handleSettings)

	// Embedded Static Frontend
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		data, err := web.DistFS.ReadFile("dist/" + path)
		if err != nil {
			// Fallback to index.html for SPA router
			data, err = web.DistFS.ReadFile("dist/index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(data)
			return
		}

		// Set mime types
		if strings.HasSuffix(path, ".html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(path, ".svg") {
			w.Header().Set("Content-Type", "image/svg+xml")
		} else if strings.HasSuffix(path, ".png") {
			w.Header().Set("Content-Type", "image/png")
		}
		_, _ = w.Write(data)
	})
}

func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := model.APIResponse{
		Success: errMsg == "",
		Data:    data,
		Error:   errMsg,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	show, _ := s.client.Show()
	total := 0
	var ramTotal int64
	if show != nil {
		total = show.Total
		ramTotal = show.RAMTotalKB
	}
	hw := hardware.GetStats()

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"containers_count": total,
		"ram_total_kb":     ramTotal,
		"binary_path":      s.client.BinaryPath(),
		"port":             s.port,
		"daemon_active":    true,
		"hardware":         hw,
	}, "")
}

func (s *Server) handleHardware(w http.ResponseWriter, r *http.Request) {
	hw := hardware.GetStats()
	s.sendJSON(w, http.StatusOK, hw, "")
}

func (s *Server) handleHostExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}
	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Command) == "" {
		s.sendJSON(w, http.StatusBadRequest, nil, "command is required")
		return
	}

	out, err := s.client.ExecHost(req.Command)
	if err != nil {
		s.sendJSON(w, http.StatusOK, model.ExecResponse{Output: out, ExitCode: 1}, "")
		return
	}
	s.sendJSON(w, http.StatusOK, model.ExecResponse{Output: out, ExitCode: 0}, "")
}

func (s *Server) handleContainers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		res, err := s.client.Show()
		if err != nil {
			s.sendJSON(w, http.StatusOK, model.ShowResult{Total: 0, Running: []model.ContainerSummary{}, Stopped: []model.ContainerSummary{}}, "")
			return
		}
		s.sendJSON(w, http.StatusOK, res, "")
	case http.MethodPost:
		var req model.StartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}
		if req.Name == "" {
			s.sendJSON(w, http.StatusBadRequest, nil, "container name is required")
			return
		}
		if err := s.client.Start(req); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "container started"}, "")
	default:
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
	}
}

func (s *Server) handleContainerAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid container url")
		return
	}
	name := parts[2]

	if len(parts) == 3 && r.Method == http.MethodGet {
		info, err := s.client.Info(name)
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, info, "")
		return
	}

	if len(parts) == 3 && r.Method == http.MethodDelete {
		err := s.client.Delete(name)
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "deleted"}, "")
		return
	}

	if len(parts) == 4 && r.Method == http.MethodPost {
		action := parts[3]
		switch action {
		case "start":
			err := s.client.Start(model.StartRequest{Name: name})
			if err != nil {
				s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
				return
			}
			s.sendJSON(w, http.StatusOK, map[string]string{"message": "started"}, "")
		case "stop":
			err := s.client.Stop(name)
			if err != nil {
				s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
				return
			}
			s.sendJSON(w, http.StatusOK, map[string]string{"message": "stopped"}, "")
		case "restart":
			err := s.client.Restart(name)
			if err != nil {
				s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
				return
			}
			s.sendJSON(w, http.StatusOK, map[string]string{"message": "restarted"}, "")
		case "delete":
			err := s.client.Delete(name)
			if err != nil {
				s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
				return
			}
			s.sendJSON(w, http.StatusOK, map[string]string{"message": "deleted"}, "")
		case "exec":
			var req model.ExecRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
				return
			}
			out, err := s.client.Exec(name, req.Command, req.User)
			if err != nil {
				s.sendJSON(w, http.StatusOK, model.ExecResponse{Output: out, ExitCode: 1}, "")
				return
			}
			s.sendJSON(w, http.StatusOK, model.ExecResponse{Output: out, ExitCode: 0}, "")
		default:
			s.sendJSON(w, http.StatusBadRequest, nil, "unknown action: "+action)
		}
		return
	}

	s.sendJSON(w, http.StatusNotFound, nil, "endpoint not found")
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	list := templates.ListTemplates()
	activeJob, progress := templates.GetDownloadStatus()
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"templates":   list,
		"active_job":  activeJob,
		"progress":    progress,
		"storage_dir": templates.GetStorageDir(),
	}, "")
}

func (s *Server) handleTemplateDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		s.sendJSON(w, http.StatusBadRequest, nil, "template ID required")
		return
	}

	if err := templates.StartDownload(req.ID); err != nil {
		s.sendJSON(w, http.StatusConflict, nil, err.Error())
		return
	}
	s.sendJSON(w, http.StatusOK, map[string]string{"message": "download started"}, "")
}

func (s *Server) handleTemplateDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		s.sendJSON(w, http.StatusBadRequest, nil, "template ID required")
		return
	}

	if err := templates.DeleteTemplate(req.ID); err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}
	s.sendJSON(w, http.StatusOK, map[string]string{"message": "deleted"}, "")
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	out, err := s.client.Check()
	if err != nil && out == "" {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}
	s.sendJSON(w, http.StatusOK, map[string]string{"output": out}, "")
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logDirs := []string{
		"/data/local/Droidspaces/Logs",
		"/var/lib/Droidspaces/Logs",
		"/data/adb/modules/droidspaces-webui",
	}

	fileParam := r.URL.Query().Get("file")
	if fileParam == "" {
		fileSet := make(map[string]bool)
		var files []string
		for _, dir := range logDirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, ent := range entries {
				if !ent.IsDir() && strings.HasSuffix(ent.Name(), ".log") {
					if !fileSet[ent.Name()] {
						fileSet[ent.Name()] = true
						files = append(files, ent.Name())
					}
				}
			}
		}
		s.sendJSON(w, http.StatusOK, map[string]interface{}{"files": files}, "")
		return
	}

	safeName := filepath.Base(fileParam)
	var targetPath string
	for _, dir := range logDirs {
		candidate := filepath.Join(dir, safeName)
		if _, err := os.Stat(candidate); err == nil {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		s.sendJSON(w, http.StatusNotFound, nil, "log file not found: "+safeName)
		return
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	lines := strings.Split(string(data), "\n")
	limit := 300
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"file":    safeName,
		"content": strings.Join(lines, "\n"),
	}, "")
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.sendJSON(w, http.StatusOK, model.SettingsConfig{
			Port:       s.port,
			BinaryPath: s.client.BinaryPath(),
		}, "")
	case http.MethodPost:
		var cfg model.SettingsConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}
		if cfg.Port > 0 && cfg.Port < 65535 {
			s.port = cfg.Port
			for _, p := range []string{
				"/data/adb/modules/droidspaces-webui/port",
				"/data/local/Droidspaces/webui_port",
			} {
				_ = os.WriteFile(p, []byte(fmt.Sprintf("%d\n", cfg.Port)), 0644)
			}
		}
		s.sendJSON(w, http.StatusOK, model.SettingsConfig{
			Port:       s.port,
			BinaryPath: s.client.BinaryPath(),
		}, "")
	default:
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
	}
}
