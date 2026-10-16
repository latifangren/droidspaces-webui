// Package marketplace provides the local App Marketplace catalog and the
// server-side job manager that runs the curated setup steps for each preset.
//
// Design goals:
//   - The catalog is bundled in the binary and only exposes safe metadata to
//     the frontend. Commands and runner options are kept server-side.
//   - Setup steps are the only authority for shell execution triggered by the
//     marketplace. They are static, do not interpolate user input, and only run
//     inside an existing container via the runner API.
//   - No client-supplied shell command is ever accepted by the marketplace API.
package marketplace

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

// AppPreset is the server-authored description of a deployable app.
//
// Commands live in SetupSteps and are never sent to the browser. The frontend
// receives only the descriptive fields and StartRequest defaults so the user
// can confirm via the existing CreateModal flow.
type AppPreset struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	Category         string             `json:"category"`
	Icon             string             `json:"icon"`
	RequiredRootFSID string             `json:"required_rootfs_id"`
	DefaultName      string             `json:"default_name"`
	StartRequest     model.StartRequest `json:"start_request"`
	Prerequisites    []string           `json:"prerequisites"`
	SetupSteps       []SetupStep        `json:"-"`
	PostSetupNote    string             `json:"post_setup_note"`
}

// SetupStep is a single server-authorized command to run inside the container
// after it has been started by the user via the standard container create
// flow. Commands are static strings; do not interpolate user input.
type SetupStep struct {
	Name    string `json:"-"`
	Command string `json:"-"`
	Hint    string `json:"-"`
}

// Manifest is the slice of presets returned to the UI. It is the only public
// view of the catalog. The commands are stripped before marshalling.
type Manifest struct {
	Presets []AppPreset `json:"presets"`
}

var (
	catalogOnce sync.Once
	catalog     []AppPreset
)

// DefaultCatalog returns a defensive copy of the bundled catalog.
//
// Each call returns a fresh slice; callers cannot mutate the package state.
func DefaultCatalog() []AppPreset {
	catalogOnce.Do(initCatalog)
	out := make([]AppPreset, len(catalog))
	copy(out, catalog)
	return out
}

// ByID returns a copy of the preset with the given id, or false if absent.
// The SetupSteps remain on the returned copy so the job manager can execute
// them; the API layer strips them before sending the preset to the browser.
func ByID(id string) (AppPreset, bool) {
	for _, p := range DefaultCatalog() {
		if p.ID == id {
			return p, true
		}
	}
	return AppPreset{}, false
}

// PublicView returns the catalog with SetupSteps removed. The frontend uses it
// to render cards, run validation, and to construct the CreateModal prefill.
func PublicView() Manifest {
	presets := DefaultCatalog()
	out := make([]AppPreset, 0, len(presets))
	for _, p := range presets {
		p.SetupSteps = nil
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return Manifest{Presets: out}
}

// ValidateName enforces the project-wide container name regex. Centralised
// here so the catalog and the API share the same rule.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("container name is required")
	}
	if len(name) > 64 {
		return fmt.Errorf("container name too long")
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("invalid container name: must be alphanumeric, _ or -")
	}
	return nil
}

// RequiresRootFS reports whether a container rootfs path or ID identifies the
// distro required by this preset. Container summaries may provide an ID or the
// configured path; cloned container paths can be identified from os-release.
func (p AppPreset) RequiresRootFS(rootfs string) bool {
	if p.RequiredRootFSID == "" {
		return true
	}

	rootfs = strings.TrimSpace(rootfs)
	if rootfs == "" {
		return false
	}
	if strings.EqualFold(rootfs, p.RequiredRootFSID) || strings.EqualFold(filepath.Base(filepath.Clean(rootfs)), p.RequiredRootFSID) {
		return true
	}

	if info, err := os.Stat(rootfs); err == nil && info.IsDir() {
		return rootFSMatchesOSRelease(rootfs, p.RequiredRootFSID)
	}
	return false
}

func rootFSMatchesOSRelease(rootfs, requiredID string) bool {
	parts := strings.SplitN(requiredID, "-", 2)
	if len(parts) != 2 {
		return false
	}

	file, err := os.Open(filepath.Join(rootfs, "etc", "os-release"))
	if err != nil {
		return false
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), "\\\"'")
	}
	if scanner.Err() != nil {
		return false
	}

	id := strings.ToLower(values["ID"])
	version := strings.ToLower(values["VERSION_ID"])
	distro := strings.ToLower(parts[0])
	requiredVersion := strings.ToLower(parts[1])
	return id == distro && (version == requiredVersion || strings.HasPrefix(version, requiredVersion+"."))
}

func initCatalog() {
	catalog = []AppPreset{
		nginxStaticPreset(),
		wireGuardClientPreset(),
		homeAssistantPreset(),
	}
}

func nginxStaticPreset() AppPreset {
	port8080 := "8080:80"
	return AppPreset{
		ID:               "nginx-static",
		Name:             "Nginx Static",
		Description:      "Lightweight HTTP server for static sites served from Alpine Linux.",
		Category:         "server",
		Icon:             "Globe",
		RequiredRootFSID: "alpine-3.20",
		DefaultName:      "nginx-static-01",
		StartRequest: model.StartRequest{
			Net:       "nat",
			Memory:    "128M",
			Port:      []string{port8080},
			RunAtBoot: false,
		},
		Prerequisites: []string{
			"Alpine 3.20 rootfs must be installed from the RootFS Store.",
			"The container needs outbound network access to fetch packages.",
		},
		SetupSteps: []SetupStep{
			{
				Name:    "install-nginx",
				Command: "apk add --no-cache nginx",
				Hint:    "Installs nginx from the Alpine package repository.",
			},
			{
				Name:    "write-index",
				Command: "printf '%s\\n' '<!doctype html><html><body><h1>Nginx Static</h1><p>Deployed via App Marketplace.</p></body></html>' > /var/www/localhost/htdocs/index.html",
				Hint:    "Writes a placeholder index page.",
			},
		},
		PostSetupNote: "Setup installs nginx and writes a placeholder page; it does not start the server. Start nginx inside the container, then open host port 8080.",
	}
}

func wireGuardClientPreset() AppPreset {
	return AppPreset{
		ID:               "wireguard-client",
		Name:             "WireGuard Client",
		Description:      "Debian container with wireguard-tools installed; user supplies VPN configuration.",
		Category:         "network",
		Icon:             "Shield",
		RequiredRootFSID: "debian-12",
		DefaultName:      "wireguard-client-01",
		StartRequest: model.StartRequest{
			Net:    "host",
			Memory: "256M",
		},
		Prerequisites: []string{
			"Debian 12 rootfs must be installed from the RootFS Store.",
			"Host kernel and runner must support WireGuard and TUN access; userspace tools alone cannot provide kernel support.",
			"A WireGuard configuration (.conf) and matching private key must be supplied by the user after setup completes.",
		},
		SetupSteps: []SetupStep{
			{
				Name:    "refresh-package-index",
				Command: "apt-get update -y",
				Hint:    "Refreshes the Debian package index.",
			},
			{
				Name:    "install-wireguard-tools",
				Command: "DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends wireguard-tools",
				Hint:    "Installs the wg and wg-quick userspace tools.",
			},
			{
				Name:    "stage-config-dir",
				Command: "install -d -m 0700 /etc/wireguard",
				Hint:    "Creates a private directory for the user-provided configuration.",
			},
		},
		PostSetupNote: "Setup installs the tools only. Activate the tunnel with `wg-quick up <name>` after placing a configuration in /etc/wireguard.",
	}
}

func homeAssistantPreset() AppPreset {
	port8123 := "8123:8123"
	return AppPreset{
		ID:               "home-assistant",
		Name:             "Home Assistant",
		Description:      "Installs pinned Home Assistant Core in a Debian container; start it manually after setup.",
		Category:         "automation",
		Icon:             "Home",
		RequiredRootFSID: "debian-12",
		DefaultName:      "home-assistant-01",
		StartRequest: model.StartRequest{
			Net:    "nat",
			Memory: "1G",
			Port:   []string{port8123},
			EnvVars: []string{
				"PYTHONUNBUFFERED=1",
			},
		},
		Prerequisites: []string{
			"Debian 12 rootfs must be installed from the RootFS Store.",
			"Container requires at least 1 GB RAM and ~2 GB of free disk for dependencies.",
			"Installing Python dependencies can take several minutes. Home Assistant is not started automatically.",
		},
		SetupSteps: []SetupStep{
			{
				Name:    "refresh-package-index",
				Command: "apt-get update -y",
				Hint:    "Refreshes the Debian package index.",
			},
			{
				Name:    "install-system-deps",
				Command: "DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends sudo python3 python3-venv python3-pip libffi-dev libssl-dev build-essential",
				Hint:    "Installs sudo and the Python toolchain required to bootstrap Home Assistant.",
			},
			{
				Name:    "create-ha-user",
				Command: "id -u hass >/dev/null 2>&1 || useradd --create-home --system hass",
				Hint:    "Creates a dedicated system user that owns the install directory.",
			},
			{
				Name:    "create-venv",
				Command: "install -d -o hass -g hass /opt/homeassistant && sudo -u hass python3 -m venv /opt/homeassistant/venv",
				Hint:    "Creates an isolated Python virtual environment owned by hass.",
			},
			{
				Name:    "install-homeassistant",
				Command: "sudo -u hass /opt/homeassistant/venv/bin/pip install --no-cache-dir homeassistant==2024.9.3",
				Hint:    "Installs a pinned Home Assistant Core release to keep builds reproducible.",
			},
		},
		PostSetupNote: "Setup installs Home Assistant Core but does not start it. Run `sudo -u hass /opt/homeassistant/venv/bin/hass` inside the container; after it starts successfully, open host port 8123.",
	}
}
