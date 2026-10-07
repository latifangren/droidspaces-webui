package model

type ShowResult struct {
	Total      int                `json:"total"`
	RAMTotalKB int64              `json:"ram_total_kb"`
	Running    []ContainerSummary `json:"running"`
	Stopped    []ContainerSummary `json:"stopped,omitempty"`
}

type ContainerSummary struct {
	Name            string  `json:"name"`
	PID             int     `json:"pid"`
	OS              string  `json:"os,omitempty"`
	Hostname        string  `json:"hostname,omitempty"`
	IP              string  `json:"ip,omitempty"`
	IP6             string  `json:"ip6,omitempty"`
	UptimeSec       int64   `json:"uptime_sec,omitempty"`
	Uptime          string  `json:"uptime,omitempty"`
	RAMUsedKB       int64   `json:"ram_used_kb,omitempty"`
	CPUPermill      int64   `json:"cpu_permill,omitempty"`
	CPUPercent      float64 `json:"cpu_percent,omitempty"`
	RAMLimitKB      int64   `json:"ram_limit_kb,omitempty"`
	CPULimitPermill int64   `json:"cpu_limit_permill,omitempty"`
	Status          string  `json:"status,omitempty"` // "running" | "stopped"
	RootFS          string  `json:"rootfs,omitempty"`
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

type TemplateInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Distro      string `json:"distro"`
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

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
