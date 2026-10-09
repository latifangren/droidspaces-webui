package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/model"
	"github.com/latifangren/droidspaces-webui/internal/runner"
)

var safeContainerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (s *Server) handleContainers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		res, err := s.client.Show()
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, res, "")

	case http.MethodPost:
		var req model.StartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}
		if req.Name == "" || !safeContainerNameRegex.MatchString(req.Name) {
			s.sendJSON(w, http.StatusBadRequest, nil, "invalid container name: must be alphanumeric, _ or -")
			return
		}
		if err := s.client.Start(req); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "container created and started"}, "")

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
	if !safeContainerNameRegex.MatchString(name) {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid container name")
		return
	}
	if len(parts) == 3 && r.Method == http.MethodGet {
		info, err := s.client.Info(name)
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, info, "")
		return
	}

	// DELETE /api/containers/{name}
	if len(parts) == 3 && r.Method == http.MethodDelete {
		if err := s.client.Delete(name); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "deleted"}, "")
		return
	}

	if len(parts) >= 4 {
		action := parts[3]

		// Init System Services
		if action == "services" {
			s.handleServicesRoute(w, r, name, parts)
			return
		}

		// Container Processes
		if action == "processes" {
			s.handleProcessesRoute(w, r, name, parts)
			return
		}

		// Container Users
		if action == "users" && r.Method == http.MethodGet {
			s.handleUsersRoute(w, r, name)
			return
		}
		// Container Backups
		if action == "backups" && r.Method == http.MethodGet {
			list, err := s.client.ListBackups(name)
			if err != nil {
				s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
				return
			}
			s.sendJSON(w, http.StatusOK, list, "")
			return
		}

		if action == "backup" {
			switch r.Method {
			case http.MethodGet:
				fileParam := r.URL.Query().Get("file")
				if fileParam == "" {
					list, err := s.client.ListBackups(name)
					if err != nil {
						s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
						return
					}
					s.sendJSON(w, http.StatusOK, list, "")
					return
				}
				cleanName := filepath.Base(fileParam)
				filePath := filepath.Join("/data/local/Droidspaces/Backups", cleanName)
				if _, err := os.Stat(filePath); err != nil {
					s.sendJSON(w, http.StatusNotFound, nil, "backup file not found")
					return
				}
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", cleanName))
				w.Header().Set("Content-Type", "application/gzip")
				http.ServeFile(w, r, filePath)
				return

			case http.MethodPost:
				backupsDir := "/data/local/Droidspaces/Backups"
				_ = os.MkdirAll(backupsDir, 0755)
				timestamp := time.Now().Format("20060102_150405")
				outFilename := fmt.Sprintf("%s_%s.tar.gz", name, timestamp)
				outPath := filepath.Join(backupsDir, outFilename)

				if err := s.client.Export(name, outPath); err != nil {
					s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
					return
				}

				stat, _ := os.Stat(outPath)
				var sizeBytes int64
				if stat != nil {
					sizeBytes = stat.Size()
				}

				s.sendJSON(w, http.StatusOK, model.ContainerBackupInfo{
					Filename:  outFilename,
					Path:      outPath,
					Size:      runner.FormatBytes(sizeBytes),
					SizeBytes: sizeBytes,
					ModTime:   time.Now().Format("2006-01-02 15:04:05"),
				}, "")
				return

			case http.MethodDelete:
				fileParam := r.URL.Query().Get("file")
				if fileParam == "" {
					s.sendJSON(w, http.StatusBadRequest, nil, "file parameter required")
					return
				}
				if err := s.client.DeleteBackup(fileParam); err != nil {
					s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
					return
				}
				s.sendJSON(w, http.StatusOK, map[string]string{"message": "backup deleted"}, "")
				return
			}
		}

		// Lifecycle Actions
		if r.Method == http.MethodPost {
			s.handleLifecycleRoute(w, r, name, action)
			return
		}
	}

	s.sendJSON(w, http.StatusNotFound, nil, "endpoint not found")
}

func (s *Server) handleLifecycleRoute(w http.ResponseWriter, r *http.Request, name, action string) {
	switch action {
	case "start":
		if err := s.client.Start(model.StartRequest{Name: name}); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "started"}, "")
	case "stop":
		if err := s.client.Stop(name); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "stopped"}, "")
	case "restart":
		if err := s.client.Restart(name); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "restarted"}, "")
	case "delete":
		if err := s.client.Delete(name); err != nil {
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
	case "clone":
		var req model.CloneRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}
		if err := s.client.Clone(name, req.TargetName, req.AutoStart); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "container cloned successfully", "name": req.TargetName}, "")
	default:
		s.sendJSON(w, http.StatusBadRequest, nil, "unknown action: "+action)
	}
}

func (s *Server) handleContainerRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}
	var req model.RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
		return
	}
	if err := s.client.Restore(req); err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}
	s.sendJSON(w, http.StatusOK, map[string]string{"message": "container restored successfully", "name": req.TargetName}, "")
}

func (s *Server) handleAllBackups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.client.ListAllBackups()
		if err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, list, "")
	case http.MethodDelete:
		fileParam := r.URL.Query().Get("file")
		if fileParam == "" {
			s.sendJSON(w, http.StatusBadRequest, nil, "file parameter required")
			return
		}
		if err := s.client.DeleteBackup(fileParam); err != nil {
			s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
			return
		}
		s.sendJSON(w, http.StatusOK, map[string]string{"message": "backup deleted"}, "")
	default:
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
	}
}

func (s *Server) handleBackupDownload(w http.ResponseWriter, r *http.Request) {
	fileParam := r.URL.Query().Get("file")
	if fileParam == "" {
		s.sendJSON(w, http.StatusBadRequest, nil, "file parameter required")
		return
	}
	cleanName := filepath.Base(fileParam)
	filePath := filepath.Join("/data/local/Droidspaces/Backups", cleanName)
	if _, err := os.Stat(filePath); err != nil {
		s.sendJSON(w, http.StatusNotFound, nil, "backup file not found")
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", cleanName))
	w.Header().Set("Content-Type", "application/gzip")
	http.ServeFile(w, r, filePath)
}
