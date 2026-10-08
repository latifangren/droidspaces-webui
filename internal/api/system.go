package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/hardware"
	"github.com/latifangren/droidspaces-webui/internal/model"
)

var (
	ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
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

	result := parseKernelCheckOutput(out)
	s.sendJSON(w, http.StatusOK, result, "")
}

func parseKernelCheckOutput(raw string) model.KernelCheckResult {
	clean := ansiRegex.ReplaceAllString(raw, "")
	lines := strings.Split(clean, "\n")

	var groups []model.KernelCheckGroup
	var curGroup *model.KernelCheckGroup
	var curItem *model.KernelCheckItem

	summary := "System requirement check completed."
	allRequiredPassed := true

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "[MUST HAVE]"):
			groups = append(groups, model.KernelCheckGroup{
				Title:       "Critical Kernel Requirements (Must Have)",
				Description: "Essential kernel namespaces and security capabilities required for container execution.",
				Required:    true,
			})
			curGroup = &groups[len(groups)-1]
			curItem = nil

		case strings.HasPrefix(trimmed, "[RECOMMENDED]"):
			groups = append(groups, model.KernelCheckGroup{
				Title:       "Performance & Integration (Recommended)",
				Description: "Subsystems that accelerate filesystem I/O, memory limits, and hardware binding.",
				Required:    false,
			})
			curGroup = &groups[len(groups)-1]
			curItem = nil

		case strings.HasPrefix(trimmed, "[OPTIONAL]"):
			groups = append(groups, model.KernelCheckGroup{
				Title:       "Extensions & Advanced Virtualization (Optional)",
				Description: "Optional kernel modules for network overlays, Docker sandboxing, and IPv6.",
				Required:    false,
			})
			curGroup = &groups[len(groups)-1]
			curItem = nil

		case strings.HasPrefix(trimmed, "Summary:"):
			curGroup = nil
			curItem = nil

		case strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "]") && curGroup != nil:
			idx := strings.Index(trimmed, "]")
			statusPart := trimmed[:idx+1]
			namePart := strings.TrimSpace(trimmed[idx+1:])

			passed := strings.Contains(statusPart, "✓") || strings.Contains(statusPart, "v")

			desc := getHumanFeatureDesc(namePart)
			curGroup.Items = append(curGroup.Items, model.KernelCheckItem{
				Name:      namePart,
				Passed:    passed,
				HumanDesc: desc,
			})
			curItem = &curGroup.Items[len(curGroup.Items)-1]

			if !passed && curGroup.Required {
				allRequiredPassed = false
			}

		case curItem != nil && strings.HasPrefix(line, "      "):
			curItem.Hint = trimmed

		case strings.Contains(trimmed, "All required features found"):
			summary = "All critical kernel requirements met. System is ready to host Linux containers."
		}
	}

	totalFeatures := 0
	passedFeatures := 0
	for i := range groups {
		pCount := 0
		for _, it := range groups[i].Items {
			totalFeatures++
			if it.Passed {
				pCount++
				passedFeatures++
			}
		}
		groups[i].PassedCount = pCount
		groups[i].TotalCount = len(groups[i].Items)
	}

	return model.KernelCheckResult{
		Summary:           summary,
		AllRequiredPassed: allRequiredPassed,
		TotalFeatures:     totalFeatures,
		PassedFeatures:    passedFeatures,
		Groups:            groups,
		RawOutput:         strings.TrimSpace(clean),
	}
}

func getHumanFeatureDesc(name string) string {
	switch name {
	case "Root privileges":
		return "Root access (uid 0) is verified."
	case "Linux version":
		return "Host Linux kernel meets minimum version requirements."
	case "PID namespace":
		return "Container process tree isolation is fully supported."
	case "Mount namespace":
		return "Private mount points and chroot isolation active."
	case "UTS namespace":
		return "Isolated hostnames per container enabled."
	case "IPC namespace":
		return "Inter-process communication and shared memory isolation active."
	case "pivot_root syscall":
		return "Kernel supports atomic root filesystem swapping."
	case "/proc filesystem":
		return "Host kernel exposes standard virtual process filesystem."
	case "/sys filesystem":
		return "Kernel sysfs hardware abstraction available."
	case "Seccomp support":
		return "Secure computing mode filter support active."
	case "Cgroup v2 support":
		return "Unified cgroup v2 hierarchy ready for memory and swap quotas."
	case "ext4 filesystem":
		return "Native ext4 filesystem direct I/O mounting supported."
	case "OverlayFS support":
		return "Overlay filesystem active for volatile container write layers."
	case "Network namespace":
		return "Private network interfaces and routing isolation supported."
	case "Bridge device support":
		return "Kernel Ethernet bridging driver ready."
	case "Veth pair support":
		return "Virtual Ethernet peer tunnels available for container networking."
	case "Memory limit support":
		return "Kernel memory controller ready for RAM throttling."
	case "CPU limit support":
		return "CFS bandwidth quota not set in kernel. Containers run with full, unthrottled access across all CPU cores."
	case "Sandboxing (user namespaces)":
		return "Unprivileged user namespaces disabled in kernel. Docker and Podman run normally in rootful mode."
	case "IPv6 NAT support":
		return "IPv6 NAT masquerading not configured in kernel. Containers default to high-speed IPv4 NAT."
	default:
		return ""
	}
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logDirs := []string{
		"/data/local/Droidspaces/Logs",
		"/var/lib/Droidspaces/Logs",
		"/data/adb/modules/droidspaces-webui",
	}

	fileParam := r.URL.Query().Get("file")
	if fileParam == "" {
		systemFiles := make(map[string]bool)
		containerFiles := make(map[string]bool)

		for _, dir := range logDirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, ent := range entries {
				name := ent.Name()
				if !ent.IsDir() {
					if strings.HasSuffix(name, ".log") {
						systemFiles[name] = true
					}
				} else {
					subDir := filepath.Join(dir, name)
					for _, target := range []string{"console", "log"} {
						if _, err := os.Stat(filepath.Join(subDir, target)); err == nil {
							containerFiles[filepath.Join(name, target)] = true
						}
					}
				}
			}
		}

		var sysList []string
		for f := range systemFiles {
			sysList = append(sysList, f)
		}
		sort.Strings(sysList)

		var cList []string
		for f := range containerFiles {
			cList = append(cList, filepath.ToSlash(f))
		}
		sort.Strings(cList)

		allFiles := append([]string{}, sysList...)
		allFiles = append(allFiles, cList...)

		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"files":           allFiles,
			"system_files":    sysList,
			"container_files": cList,
		}, "")
		return
	}

	cleanParam := filepath.Clean(filepath.FromSlash(fileParam))
	if strings.Contains(cleanParam, "..") {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid log file path")
		return
	}

	var targetPath string
	for _, dir := range logDirs {
		candidate := filepath.Join(dir, cleanParam)
		if _, err := os.Stat(candidate); err == nil {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		s.sendJSON(w, http.StatusNotFound, nil, "log file not found: "+fileParam)
		return
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)
	limit := 500
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"file":        fileParam,
		"content":     strings.Join(lines, "\n"),
		"total_lines": totalLines,
		"shown_lines": len(lines),
		"size_bytes":  len(data),
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
