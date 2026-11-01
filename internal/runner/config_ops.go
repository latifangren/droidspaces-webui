package runner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/config"
	"github.com/latifangren/droidspaces-webui/internal/model"
)

// IsRunning reports whether container with name is currently running.
func (c *Client) IsRunning(name string) bool {
	show, err := c.Show()
	if err != nil || show == nil {
		return false
	}
	for _, r := range show.Running {
		if r.Name == name {
			return true
		}
	}
	return false
}

// GetContainerConfigDetails reads and parses container.config into a structured model.
func (c *Client) GetContainerConfigDetails(name string) (*model.ContainerConfigDetails, error) {
	if !safeContainerNameRegex.MatchString(name) {
		return nil, fmt.Errorf("invalid container name: %s", name)
	}

	cfg, err := config.ReadContainerConfig(name)
	if err != nil {
		return nil, fmt.Errorf("read container config: %w", err)
	}

	details := &model.ContainerConfigDetails{
		Name:     name,
		Hostname: cfg["hostname"],
		RootFS:   cfg["rootfs_path"],
		Raw:      cfg,
	}
	if details.Hostname == "" {
		details.Hostname = name
	}

	// Network Mode
	netMode := cfg["net_mode"]
	if netMode == "" {
		netMode = cfg["net"]
	}
	if netMode == "" {
		netMode = "nat"
	}
	details.Net = netMode

	// Network IPs and Interfaces
	if ip, ok := cfg["static_nat_ip"]; ok && ip != "" {
		details.NATIP = ip
	} else if ip, ok := cfg["nat_ip"]; ok && ip != "" {
		details.NATIP = ip
	} else if ip, ok := cfg["ip"]; ok && ip != "" {
		details.NATIP = ip
	}

	if up, ok := cfg["upstream_interfaces"]; ok {
		details.Upstream = up
	} else {
		details.Upstream = cfg["upstream"]
	}

	if gw, ok := cfg["gateway_container"]; ok {
		details.Gateway = gw
	} else {
		details.Gateway = cfg["gateway"]
	}
	details.GatewayNet = cfg["gateway_net"]
	if iface, ok := cfg["gateway_lan_ifname"]; ok {
		details.GatewayIface = iface
	} else {
		details.GatewayIface = cfg["gateway_iface"]
	}
	details.GatewayBridge = cfg["gateway_bridge"]

	details.MacvlanParent = cfg["macvlan_parent"]
	details.MacvlanMode = cfg["macvlan_mode"]
	if details.MacvlanMode == "" {
		details.MacvlanMode = "bridge"
	}

	// Port Mappings
	rawPorts := cfg["port_forwards"]
	if rawPorts == "" {
		rawPorts = cfg["port"]
	}
	if rawPorts != "" {
		details.Port = ParsePortMappings(rawPorts)
	}

	// DNS
	if dns, ok := cfg["dns_servers"]; ok {
		details.DNS = dns
	} else {
		details.DNS = cfg["dns"]
	}

	// Booleans
	details.DisableIPv6 = isTruthy(cfg["disable_ipv6"])
	details.AndroidStorage = isTruthy(cfg["enable_android_storage"])
	details.HWAccess = isTruthy(cfg["enable_hw_access"])
	details.GPU = isTruthy(cfg["enable_gpu_mode"])
	details.TermuxX11 = isTruthy(cfg["enable_x11"]) || isTruthy(cfg["enable_termux_x11"])
	details.VirGL = isTruthy(cfg["enable_virgl"])
	details.PulseAudio = isTruthy(cfg["enable_pulseaudio"])
	details.Anland = isTruthy(cfg["enable_anland"])
	details.SELinuxPermissive = isTruthy(cfg["selinux_permissive"])
	details.Volatile = isTruthy(cfg["volatile_mode"])
	details.ForceCgroupV1 = isTruthy(cfg["force_cgroupv1"])
	details.AllowSwap = isTruthy(cfg["allow_swap"])
	details.AllowSandboxing = isTruthy(cfg["allow_sandboxing"]) || isTruthy(cfg["allow_userns"])

	// Resources: Memory
	rawMem := cfg["memory_limit"]
	if rawMem != "" {
		details.Memory = normalizeMemory(rawMem)
	}

	// Resources: CPUs
	rawCPUs := cfg["cpus"]
	if rawCPUs == "" && cfg["cpu_quota"] != "" && cfg["cpu_period"] != "" {
		quota, qErr := strconv.ParseFloat(cfg["cpu_quota"], 64)
		period, pErr := strconv.ParseFloat(cfg["cpu_period"], 64)
		if qErr == nil && pErr == nil && period > 0 {
			cores := quota / period
			if cores == float64(int(cores)) {
				rawCPUs = strconv.Itoa(int(cores))
			} else {
				rawCPUs = fmt.Sprintf("%.1f", cores)
			}
		}
	}
	details.CPUs = rawCPUs
	details.PIDsLimit = cfg["pids_limit"]
	details.Privileged = cfg["privileged"]

	// Custom Binds
	rawBinds := cfg["bind_mounts"]
	if rawBinds == "" {
		rawBinds = cfg["binds"]
	}
	if rawBinds != "" {
		bindParts := strings.FieldsFunc(rawBinds, func(r rune) bool {
			return r == ',' || r == '\n' || r == ';'
		})
		for _, b := range bindParts {
			trimmed := strings.TrimSpace(b)
			if trimmed != "" {
				details.Binds = append(details.Binds, trimmed)
			}
		}
	}

	// Run at boot
	details.RunAtBoot = isTruthy(cfg["run_at_boot"])
	if prio, err := strconv.Atoi(cfg["run_at_boot_priority"]); err == nil {
		details.RunAtBootPriority = prio
	}

	details.CustomInit = cfg["custom_init"]

	// Environment Variables
	if envFile := cfg["env_file"]; envFile != "" {
		if data, err := os.ReadFile(envFile); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(data)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !strings.HasPrefix(line, "#") {
					details.EnvVars = append(details.EnvVars, line)
				}
			}
		}
	}

	details.IsRunning = c.IsRunning(name)

	return details, nil
}

// UpdateContainerConfig writes updated configuration options to container.config.
func (c *Client) UpdateContainerConfig(name string, req model.UpdateConfigRequest) error {
	if !safeContainerNameRegex.MatchString(name) {
		return fmt.Errorf("invalid container name: %s", name)
	}

	// Ensure container exists
	existing, err := config.ReadContainerConfig(name)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	updates := make(map[string]string)

	if req.Hostname != "" {
		updates["hostname"] = req.Hostname
	} else if existing["hostname"] != "" {
		updates["hostname"] = existing["hostname"]
	}

	// Network mode
	if req.Net != "" {
		updates["net_mode"] = req.Net
		updates["net"] = req.Net
	}

	updates["static_nat_ip"] = req.NATIP
	updates["upstream_interfaces"] = req.Upstream
	updates["gateway_container"] = req.Gateway
	updates["gateway"] = req.Gateway
	updates["gateway_net"] = req.GatewayNet
	updates["gateway_lan_ifname"] = req.GatewayIface
	updates["gateway_iface"] = req.GatewayIface
	updates["gateway_bridge"] = req.GatewayBridge
	updates["macvlan_parent"] = req.MacvlanParent
	if req.MacvlanMode != "" {
		updates["macvlan_mode"] = req.MacvlanMode
	}

	// Ports
	var cleanedPorts []string
	for _, p := range req.Port {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			cleanedPorts = append(cleanedPorts, trimmed)
		}
	}
	updates["port_forwards"] = strings.Join(cleanedPorts, ",")
	updates["dns_servers"] = req.DNS

	// Booleans
	updates["disable_ipv6"] = boolToStr(req.DisableIPv6)
	updates["enable_android_storage"] = boolToStr(req.AndroidStorage)
	updates["enable_hw_access"] = boolToStr(req.HWAccess)
	updates["enable_gpu_mode"] = boolToStr(req.GPU)
	updates["enable_x11"] = boolToStr(req.TermuxX11)
	updates["enable_virgl"] = boolToStr(req.VirGL)
	updates["enable_pulseaudio"] = boolToStr(req.PulseAudio)
	updates["enable_anland"] = boolToStr(req.Anland)
	updates["selinux_permissive"] = boolToStr(req.SELinuxPermissive)
	updates["volatile_mode"] = boolToStr(req.Volatile)
	updates["force_cgroupv1"] = boolToStr(req.ForceCgroupV1)
	updates["allow_swap"] = boolToStr(req.AllowSwap)
	updates["allow_sandboxing"] = boolToStr(req.AllowSandboxing)

	// Resources
	updates["memory_limit"] = req.Memory
	updates["cpus"] = req.CPUs
	updates["pids_limit"] = req.PIDsLimit
	updates["privileged"] = req.Privileged

	// Custom Binds
	var cleanedBinds []string
	for _, b := range req.Binds {
		trimmed := strings.TrimSpace(b)
		if trimmed != "" {
			cleanedBinds = append(cleanedBinds, trimmed)
		}
	}
	updates["bind_mounts"] = strings.Join(cleanedBinds, ",")

	// Autostart
	if req.RunAtBoot {
		updates["run_at_boot"] = "1"
		updates["run_at_boot_priority"] = strconv.Itoa(req.RunAtBootPriority)
	} else {
		updates["run_at_boot"] = "0"
		updates["run_at_boot_priority"] = "0"
	}

	updates["custom_init"] = req.CustomInit

	// Environment Variables handling
	targetDir := findContainerDir(name)
	if targetDir != "" {
		envPath := filepath.Join(targetDir, "container.env")
		if len(req.EnvVars) > 0 {
			var sb strings.Builder
			for _, e := range req.EnvVars {
				trimmed := strings.TrimSpace(e)
				if trimmed != "" {
					sb.WriteString(trimmed)
					sb.WriteString("\n")
				}
			}
			if err := os.WriteFile(envPath, []byte(sb.String()), 0644); err == nil {
				updates["env_file"] = envPath
			}
		} else {
			// Clear or remove env file if user cleared env vars
			if _, err := os.Stat(envPath); err == nil {
				_ = os.Remove(envPath)
			}
			updates["env_file"] = ""
		}
	}

	if err := config.WriteContainerConfigKeys(name, updates); err != nil {
		return fmt.Errorf("write container config: %w", err)
	}

	config.InvalidateCache(name)

	// Restart if requested and container is running
	if req.RestartNow && c.IsRunning(name) {
		if err := c.Restart(name); err != nil {
			return fmt.Errorf("config saved, but restart failed: %w", err)
		}
	}

	return nil
}

func findContainerDir(name string) string {
	for _, cdir := range config.GetContainersDirs() {
		dir := filepath.Join(cdir, name)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	dirs := config.GetContainersDirs()
	if len(dirs) > 0 {
		return filepath.Join(dirs[0], name)
	}
	return ""
}

func isTruthy(val string) bool {
	v := strings.TrimSpace(strings.ToLower(val))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func normalizeMemory(raw string) string {
	trimmed := strings.TrimSpace(raw)
	// If already has suffix like 512M, 1G, return as is
	upper := strings.ToUpper(trimmed)
	if strings.HasSuffix(upper, "M") || strings.HasSuffix(upper, "G") || strings.HasSuffix(upper, "K") {
		return upper
	}
	// Try parsing numeric bytes to human size
	if bytesVal, err := strconv.ParseInt(trimmed, 10, 64); err == nil && bytesVal > 0 {
		const mb = 1024 * 1024
		const gb = 1024 * mb
		if bytesVal >= gb && bytesVal%gb == 0 {
			return fmt.Sprintf("%dG", bytesVal/gb)
		}
		if bytesVal >= mb && bytesVal%mb == 0 {
			return fmt.Sprintf("%dM", bytesVal/mb)
		}
	}
	return trimmed
}
