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
)

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

func (s *Server) handleBootPriority(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.client.GetBootPriorities()
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, items, "")
	case http.MethodPost:
		var req model.UpdateBootPriorityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}
		if err := s.client.SetBootPriorities(req.Items); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "boot priorities updated"}, "")
	default:
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
	}
}

func (s *Server) handleHostInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}

	ifaces, err := s.client.ListHostNetworkInterfaces()
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}
	s.sendJSON(w, http.StatusOK, ifaces, "")
}
