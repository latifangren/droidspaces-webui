package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/hardware"
	"github.com/latifangren/droidspaces-webui/internal/model"
)

var (
	ansiRegex        = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	safeLogPathRegex = regexp.MustCompile(`^[a-zA-Z0-9_./-]+$`)
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
		"port":             s.GetPort(),
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
	// Try native v7.0.0+ JSON machine-readable probe first
	if rawJSON, err := s.client.CheckFormat(); err == nil && rawJSON != nil {
		result := buildKernelCheckResultFromJSON(rawJSON)
		s.sendJSON(w, http.StatusOK, result, "")
		return
	}

	out, err := s.client.Check()
	if err != nil && out == "" {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	result := parseKernelCheckOutput(out)
	s.sendJSON(w, http.StatusOK, result, "")
}

func isFlagPassed(val interface{}) bool {
	switch v := val.(type) {
	case float64:
		return v > 0
	case int:
		return v > 0
	case bool:
		return v
	case string:
		return v == "1" || strings.ToLower(v) == "true" || strings.ToLower(v) == "yes"
	default:
		return false
	}
}

func buildKernelCheckResultFromJSON(raw map[string]interface{}) model.KernelCheckResult {
	type itemDef struct {
		key      string
		name     string
		desc     string
		hint     string
		required bool
	}

	criticalItems := []itemDef{
		{"root", "Root Privileges", "Executing with Linux superuser authority (UID 0)", "Requires Magisk / KernelSU / APatch root", true},
		{"kernel_version", "Kernel Version >= 3.10", "Linux kernel 3.10 or newer with namespace APIs", "Update device kernel / ROM", true},
		{"pid_ns", "Process Namespace (PID)", "Isolates container process trees (CLONE_NEWPID)", "Kernel CONFIG_PID_NS=y", true},
		{"mnt_ns", "Mount Namespace (MNT)", "Isolates container filesystem mounts (CLONE_NEWNS)", "Kernel CONFIG_NAMESPACES=y", true},
		{"uts_ns", "Hostname Namespace (UTS)", "Independent container hostnames (CLONE_NEWUTS)", "Kernel CONFIG_UTS_NS=y", true},
		{"ipc_ns", "IPC Namespace (IPC)", "Isolates System V IPC and POSIX message queues", "Kernel CONFIG_IPC_NS=y", true},
		{"pivot_root", "Pivot Root Support", "Safely swaps container rootfs into place", "Kernel pivot_root syscall", true},
		{"procfs", "Proc Virtual Filesystem", "Mounts isolated /proc directory", "Kernel CONFIG_PROC_FS=y", true},
		{"sysfs", "Sys Virtual Filesystem", "Mounts container /sys subsystem", "Kernel CONFIG_SYSFS=y", true},
		{"pty", "Pseudo-Terminal (PTY)", "Allocates interactive terminal pair (/dev/ptmx)", "Kernel CONFIG_UNIX98_PTYS=y", true},
		{"devpts", "DevPTS Filesystem", "Mounts slave PTY device nodes (/dev/pts)", "Kernel CONFIG_DEVPTS_MULTIPLE_INSTANCES=y", true},
		{"loop", "Loop Block Device", "Mounts ext4 sparse disk images (.img)", "Kernel CONFIG_BLK_DEV_LOOP=y", true},
		{"ext4", "Ext4 Filesystem Support", "Native Linux ext4 filesystem driver for containers", "Kernel CONFIG_EXT4_FS=y", true},
	}

	isolationItems := []itemDef{
		{"cgroup2", "Cgroup v2 Controller", "Unified control group hierarchy (memory, CPU)", "Kernel CONFIG_CGROUPS=y", false},
		{"cgroup_ns", "Cgroup Namespace", "Isolates container cgroup view (CLONE_NEWCGROUP)", "Kernel CONFIG_CGROUP_NET_PRIO=y", false},
		{"user_ns", "User Namespace (Docker)", "Enables rootless Docker and nested containers", "Kernel CONFIG_USER_NS=y", false},
		{"seccomp", "Seccomp BPF Filter", "Filters dangerous syscalls inside containers", "Kernel CONFIG_SECCOMP=y", false},
		{"overlayfs", "OverlayFS Driver", "Required for Docker overlay2 storage and volatile mode", "Kernel CONFIG_OVERLAY_FS=y", false},
	}

	networkItems := []itemDef{
		{"net_ns", "Network Namespace (NET)", "Isolates network stack, interfaces, and routes", "Kernel CONFIG_NET_NS=y", true},
		{"bridge", "Bridge Network Support", "Virtual switch for NAT and Gateway routing", "Kernel CONFIG_BRIDGE=y", true},
		{"veth", "Virtual Ethernet (VETH)", "Virtual Ethernet cables connecting containers to host", "Kernel CONFIG_VETH=y", true},
		{"macvlan", "Macvlan Physical LAN", "Connects container directly to physical Wi-Fi/Ethernet", "Kernel CONFIG_MACVLAN=y", false},
		{"tun", "Universal TUN/TAP Device", "Supports VPNs, WireGuard, and OpenWrt tunnels", "Kernel CONFIG_TUN=y", false},
		{"ipv6_nat", "IPv6 NAT (NAT66)", "Dual-stack IPv6 routing with stateful masquerade", "Kernel CONFIG_IP6_NF_NAT=y", false},
	}

	cgroupItems := []itemDef{
		{"memory_limit", "Memory Limit Controller", "Restricts maximum container RAM quota", "Kernel CONFIG_MEMCG=y", false},
		{"cpu_limit", "CPU Bandwidth Quota", "Restricts container CPU execution percentage", "Kernel CONFIG_CFS_BANDWIDTH=y", false},
		{"cpu_accounting", "CPU Usage Accounting", "Real-time container CPU load & stat tracking", "Kernel CONFIG_CGROUP_CPUACCT=y", false},
		{"pids_limit", "Process Number Limit", "Guards against fork bombs (max PID count)", "Kernel CONFIG_CGROUP_PIDS=y", false},
	}

	makeGroup := func(title, desc string, required bool, defs []itemDef) model.KernelCheckGroup {
		var items []model.KernelCheckItem
		passed := 0
		for _, d := range defs {
			ok := false
			if v, exists := raw[d.key]; exists {
				ok = isFlagPassed(v)
			}
			if ok {
				passed++
			}
			items = append(items, model.KernelCheckItem{
				Name:      d.name,
				Passed:    ok,
				HumanDesc: d.desc,
				Hint:      d.hint,
			})
		}
		return model.KernelCheckGroup{
			Title:       title,
			Description: desc,
			Required:    required,
			PassedCount: passed,
			TotalCount:  len(defs),
			Items:       items,
		}
	}

	groups := []model.KernelCheckGroup{
		makeGroup("Critical Kernel Requirements (Must Have)", "Essential namespaces and filesystem drivers required to run containers.", true, criticalItems),
		makeGroup("Container Isolation & Sandboxing", "Kernel features for security isolation and Docker container nesting.", false, isolationItems),
		makeGroup("Advanced Networking & Routing", "Virtual bridges, veth cables, and network namespace routing.", true, networkItems),
		makeGroup("Resource Limits & Cgroups", "Hardware resource quotas (Memory limits, CPU quotas, PID caps).", false, cgroupItems),
	}

	total := 0
	passed := 0
	allReq := true
	for _, g := range groups {
		total += g.TotalCount
		passed += g.PassedCount
		if g.Required && g.PassedCount < g.TotalCount {
			allReq = false
		}
	}

	summary := fmt.Sprintf("%d of %d system capabilities verified on your Android kernel.", passed, total)
	if !allReq {
		summary = "Warning: Some required kernel capabilities are missing on this device."
	}

	rawBytes, _ := json.Marshal(raw)

	return model.KernelCheckResult{
		Summary:           summary,
		AllRequiredPassed: allReq,
		TotalFeatures:     total,
		PassedFeatures:    passed,
		Capabilities:      raw,
		Groups:            groups,
		RawOutput:         string(rawBytes),
	}
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


func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.sendJSON(w, http.StatusOK, model.SettingsConfig{
			Port:       s.GetPort(),
			BinaryPath: s.client.BinaryPath(),
		}, "")
	case http.MethodPost:
		var cfg model.SettingsConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			s.sendJSON(w, http.StatusBadRequest, nil, err.Error())
			return
		}

		if cfg.Port > 0 && cfg.Port <= 65535 {
			s.SetPort(cfg.Port)
			for _, p := range []string{
				"/data/local/Droidspaces/webui_port",
				"/data/adb/modules/droidspaces/port",
				"/data/adb/modules/droidspaces-webui/port",
			} {
				_ = os.WriteFile(p, []byte(fmt.Sprintf("%d\n", cfg.Port)), 0644)
			}
		}

		s.sendJSON(w, http.StatusOK, model.SettingsConfig{
			Port:       s.GetPort(),
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
