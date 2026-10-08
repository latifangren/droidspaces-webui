package model

type ShowResult struct {
	Total      int                `json:"total"`
	RAMTotalKB int64              `json:"ram_total_kb"`
	Running    []ContainerSummary `json:"running"`
	Stopped    []ContainerSummary `json:"stopped,omitempty"`
}

type ContainerSummary struct {
	Name              string  `json:"name"`
	PID               int     `json:"pid"`
	OS                string  `json:"os,omitempty"`
	Hostname          string  `json:"hostname,omitempty"`
	IP                string  `json:"ip,omitempty"`
	IP6               string  `json:"ip6,omitempty"`
	UptimeSec         int64   `json:"uptime_sec,omitempty"`
	Uptime            string  `json:"uptime,omitempty"`
	RAMUsedKB         int64   `json:"ram_used_kb,omitempty"`
	CPUPermill        int64   `json:"cpu_permill,omitempty"`
	CPUPercent        float64 `json:"cpu_percent,omitempty"`
	RAMLimitKB        int64   `json:"ram_limit_kb,omitempty"`
	CPULimitPermill   int64   `json:"cpu_limit_permill,omitempty"`
	Status            string  `json:"status,omitempty"` // "running" | "stopped"
	RootFS            string  `json:"rootfs,omitempty"`
	InitSystem        string  `json:"init_system,omitempty"` // "systemd" | "openrc" | "procd" | "unknown"
	RunAtBoot         bool    `json:"run_at_boot,omitempty"`
	RunAtBootPriority int     `json:"run_at_boot_priority,omitempty"`
}

type StartRequest struct {
	Name              string   `json:"name"`
	RootFS            string   `json:"rootfs,omitempty"`
	RootFSImg         string   `json:"rootfs_img,omitempty"`
	Hostname          string   `json:"hostname,omitempty"`
	Conf              string   `json:"conf,omitempty"`
	Net               string   `json:"net,omitempty"` // host, nat, none, gateway
	Gateway           string   `json:"gateway,omitempty"`
	GatewayNet        string   `json:"gateway_net,omitempty"`
	GatewayIface      string   `json:"gateway_iface,omitempty"`
	GatewayBridge     string   `json:"gateway_bridge,omitempty"`
	NATIP             string   `json:"nat_ip,omitempty"`
	Upstream          string   `json:"upstream,omitempty"`
	Port              []string `json:"port,omitempty"`
	DNS               string   `json:"dns,omitempty"`
	DisableIPv6       bool     `json:"disable_ipv6,omitempty"`
	AndroidStorage    bool     `json:"android_storage,omitempty"`
	HWAccess          bool     `json:"hw_access,omitempty"`
	GPU               bool     `json:"gpu,omitempty"`
	TermuxX11         bool     `json:"termux_x11,omitempty"`
	VirGL             bool     `json:"virgl,omitempty"`
	PulseAudio        bool     `json:"pulse_audio,omitempty"`
	SELinuxPermissive bool     `json:"selinux_permissive,omitempty"`
	Volatile          bool     `json:"volatile,omitempty"`
	ForceCgroupV1     bool     `json:"force_cgroupv1,omitempty"`
	Memory            string   `json:"memory,omitempty"`
	CPUs              string   `json:"cpus,omitempty"`
	PIDsLimit         string   `json:"pids_limit,omitempty"`
	Privileged        string   `json:"privileged,omitempty"`
	AllowSandboxing   bool     `json:"allow_sandboxing,omitempty"` // Docker/Podman support
	Binds             []string `json:"binds,omitempty"`
	RunAtBoot         bool     `json:"run_at_boot,omitempty"`
	RunAtBootPriority int      `json:"run_at_boot_priority,omitempty"`
	CustomInit        string   `json:"custom_init,omitempty"`
	EnvVars           []string `json:"env_vars,omitempty"`
}

type ExecRequest struct {
	Container string `json:"container"`
	Command   string `json:"command"`
	User      string `json:"user,omitempty"`
}

type ExecResponse struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
}

type ServiceInfo struct {
	Name        string `json:"name"`
	State       string `json:"state"`   // "running", "stopped", "failed", "active", "inactive"
	Enabled     string `json:"enabled"` // "enabled", "disabled", "masked", "static", "unknown"
	Description string `json:"description"`
}

type ServiceActionRequest struct {
	Service string `json:"service"`
	Action  string `json:"action"` // "start", "stop", "restart", "enable", "disable", "mask", "unmask"
}

type ProcessInfo struct {
	PID     int    `json:"pid"`
	User    string `json:"user"`
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Command string `json:"command"`
}

type KillProcessRequest struct {
	PID    int `json:"pid"`
	Signal int `json:"signal,omitempty"` // Default 9 (SIGKILL)
}

type BootPriorityItem struct {
	Name              string `json:"name"`
	RunAtBoot         bool   `json:"run_at_boot"`
	RunAtBootPriority int    `json:"run_at_boot_priority"`
	Status            string `json:"status"` // "running" | "stopped"
}

type UpdateBootPriorityRequest struct {
	Items []BootPriorityItem `json:"items"`
}

type NetworkInterfaceInfo struct {
	Name  string `json:"name"`
	IP    string `json:"ip,omitempty"`
	Type  string `json:"type"`  // "wifi", "cellular", "ethernet", "bridge", "other"
	State string `json:"state"` // "up", "down"
}

type TemplateInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Distro      string `json:"distro"`
	Category    string `json:"category"`    // "server" | "desktop" | "network"
	Environment string `json:"environment"` // "Headless Server (CLI)" | "Desktop GUI (X11)" | etc.
	InitSystem  string `json:"init_system"` // "systemd" | "openrc" | "procd"
	MinRAM      string `json:"min_ram"`     // "256 MB" | "1.5 GB"
	Arch        string `json:"arch"`
	Version     string `json:"version"`
	URL         string `json:"url"`
	Type        string `json:"type"` // "tar.xz", "tar.gz", "img"
	SizeMB      int    `json:"size_mb"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
	LocalPath   string `json:"local_path,omitempty"`
}

type SettingsConfig struct {
	Port       int    `json:"port"`
	BinaryPath string `json:"binary_path"`
}

type TerminalSessionInfo struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Target        string `json:"target"` // "container" | "host"
	Container     string `json:"container,omitempty"`
	User          string `json:"user,omitempty"`
	CreatedAt     int64  `json:"created_at"` // Unix timestamp
	ActiveClients int    `json:"active_clients"`
	Cols          uint16 `json:"cols"`
	Rows          uint16 `json:"rows"`
	IsRunning     bool   `json:"is_running"`
}

type CreateSessionRequest struct {
	Title     string `json:"title,omitempty"`
	Target    string `json:"target"` // "container" | "host"
	Container string `json:"container,omitempty"`
	User      string `json:"user,omitempty"`
}

type KernelCheckItem struct {
	Name      string `json:"name"`
	Passed    bool   `json:"passed"`
	Hint      string `json:"hint,omitempty"`
	HumanDesc string `json:"human_desc,omitempty"`
}

type KernelCheckGroup struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Required    bool              `json:"required"`
	PassedCount int               `json:"passed_count"`
	TotalCount  int               `json:"total_count"`
	Items       []KernelCheckItem `json:"items"`
}

type KernelCheckResult struct {
	Summary           string             `json:"summary"`
	AllRequiredPassed bool               `json:"all_required_passed"`
	TotalFeatures     int                `json:"total_features"`
	PassedFeatures    int                `json:"passed_features"`
	Groups            []KernelCheckGroup `json:"groups"`
	RawOutput         string             `json:"raw_output"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
