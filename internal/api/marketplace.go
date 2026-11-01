package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/marketplace"
)

type setupJobRequest struct {
	AppID         string `json:"app_id"`
	ContainerName string `json:"container_name"`
}

func (s *Server) handleMarketplaceSetupJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}
	var req setupJobRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	if err := decoder.Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid setup job request")
		return
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid setup job request")
		return
	}
	if _, ok := marketplace.ByID(req.AppID); !ok {
		s.sendJSON(w, http.StatusBadRequest, nil, "unknown app preset")
		return
	}
	if err := marketplace.ValidateName(req.ContainerName); err != nil {
		s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
		return
	}

	preset, _ := marketplace.ByID(req.AppID)
	showFn := s.marketplaceShow
	if showFn == nil {
		showFn = s.client.Show
	}
	show, err := showFn()
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, "failed to inspect containers: "+err.Error())
		return
	}
	if show == nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, "failed to inspect containers: empty result")
		return
	}
	running := false
	for _, item := range show.Running {
		if item.Name == req.ContainerName {
			if !preset.RequiresRootFS(item.RootFS) {
				s.sendJSON(w, http.StatusConflict, nil, "container rootfs does not match the selected app preset")
				return
			}
			running = true
			break
		}
	}
	if !running {
		s.sendJSON(w, http.StatusConflict, nil, "container must be running before marketplace setup")
		return
	}

	job, err := s.marketplaceJobs.Start(req.AppID, req.ContainerName)
	if err != nil {
		s.sendJSON(w, http.StatusConflict, nil, err.Error())
		return
	}
	s.sendJSON(w, http.StatusAccepted, job, "")
}

func (s *Server) handleMarketplaceSetupJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendJSON(w, http.StatusMethodNotAllowed, nil, "method not allowed")
		return
	}
	jobID := strings.TrimPrefix(r.URL.Path, "/api/marketplace/setup-jobs/")
	if jobID == "" || strings.Contains(jobID, "/") {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid setup job ID")
		return
	}
	job, ok := s.marketplaceJobs.Get(jobID)
	if !ok {
		s.sendJSON(w, http.StatusNotFound, nil, "setup job not found")
		return
	}
	s.sendJSON(w, http.StatusOK, job, "")
}
