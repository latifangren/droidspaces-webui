package templates

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

var (
	downloadLock sync.Mutex
	activeJob    string
	jobProgress  string
)

var defaultCatalog = []model.TemplateInfo{
	{
		ID:          "alpine-3.20",
		Name:        "Alpine Linux 3.20 (Minimal Edge)",
		Distro:      "alpine",
		Category:    "server",
		Environment: "Minimal Headless CLI",
		InitSystem:  "OpenRC",
		MinRAM:      "64 MB",
		Arch:        "aarch64",
		Version:     "3.20.3",
		URL:         "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/aarch64/alpine-minirootfs-3.20.3-aarch64.tar.gz",
		Type:        "tar.gz",
		SizeMB:      4,
		Description: "[SERVER / HEADLESS] Ultra-lightweight ~4MB! Sub-second boot, minimal RAM (<20MB). Designed for microservices, background workers, and Docker engine. Pure CLI, zero GUI overhead.",
	},
	{
		ID:          "debian-12",
		Name:        "Debian 12 Bookworm (Full Systemd)",
		Distro:      "debian",
		Category:    "server",
		Environment: "Full Headless Server (Systemd)",
		InitSystem:  "systemd (PID 1)",
		MinRAM:      "256 MB",
		Arch:        "arm64",
		Version:     "12",
		URL:         "https://images.linuxcontainers.org/images/debian/bookworm/arm64/default/rootfs.tar.xz",
		Type:        "tar.xz",
		SizeMB:      95,
		Description: "[SERVER / HEADLESS] Homelab standard with full systemd PID 1 support. Rock-solid stability for containers, databases, and Docker. (XFCE desktop can be installed via apt).",
	},
	{
		ID:          "ubuntu-24.04",
		Name:        "Ubuntu 24.04 LTS Noble Numbat",
		Distro:      "ubuntu",
		Category:    "server",
		Environment: "Full Headless Server (Systemd)",
		InitSystem:  "systemd (PID 1)",
		MinRAM:      "256 MB",
		Arch:        "arm64",
		Version:     "24.04",
		URL:         "https://images.linuxcontainers.org/images/ubuntu/noble/arm64/default/rootfs.tar.xz",
		Type:        "tar.xz",
		SizeMB:      115,
		Description: "[SERVER / HEADLESS] Latest LTS server with modern packages, Python 3.12, systemd, and broad toolchain. Pure headless without graphical memory footprint.",
	},
	{
		ID:          "arch-linux",
		Name:        "Arch Linux ARM (Rolling Release)",
		Distro:      "arch",
		Category:    "server",
		Environment: "Rolling Headless CLI",
		InitSystem:  "systemd (PID 1)",
		MinRAM:      "256 MB",
		Arch:        "aarch64",
		Version:     "rolling",
		URL:         "http://os.archlinuxarm.org/os/ArchLinuxARM-aarch64-latest.tar.gz",
		Type:        "tar.gz",
		SizeMB:      420,
		Description: "[SERVER / HEADLESS] Rolling-release distribution with pacman package manager. Bleeding-edge packages for advanced developers. Default headless CLI.",
	},
	{
		ID:          "openwrt-23.05",
		Name:        "OpenWrt 23.05 (Network Gateway)",
		Distro:      "openwrt",
		Category:    "network",
		Environment: "Network Appliance (LuCI Web)",
		InitSystem:  "procd",
		MinRAM:      "64 MB",
		Arch:        "arm64",
		Version:     "23.05.4",
		URL:         "https://downloads.openwrt.org/releases/23.05.4/targets/armsr/armv8/openwrt-23.05.4-armsr-armv8-rootfs.tar.gz",
		Type:        "tar.gz",
		SizeMB:      25,
		Description: "[NETWORK ROUTER] Dedicated network routing & firewall appliance with LuCI Web UI on port 80. Lightweight (<30MB RAM) for VPN gateways, WireGuard, and DNS sinkholes.",
	},
}

func GetStorageDir() string {
	dirs := []string{
		"/data/local/Droidspaces/rootfs",
		"/data/adb/droidspaces/rootfs",
		"/var/lib/Droidspaces/rootfs",
		"/tmp/droidspaces/rootfs",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err == nil {
			return d
		}
	}
	return "/tmp/droidspaces/rootfs"
}

func ListTemplates() []model.TemplateInfo {
	storage := GetStorageDir()
	res := make([]model.TemplateInfo, len(defaultCatalog))
	copy(res, defaultCatalog)

	for i := range res {
		targetDir := filepath.Join(storage, res[i].ID)
		if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
			res[i].Installed = true
			if res[i].Type == "img" {
				imgFile := filepath.Join(targetDir, "rootfs.img")
				if _, err := os.Stat(imgFile); err == nil {
					res[i].LocalPath = imgFile
				} else {
					res[i].LocalPath = targetDir
				}
			} else {
				res[i].LocalPath = targetDir
			}
		}
	}
	return res
}

func GetDownloadStatus() (string, string) {
	downloadLock.Lock()
	defer downloadLock.Unlock()
	return activeJob, jobProgress
}

func DeleteTemplate(templateID string) error {
	storage := GetStorageDir()
	targetDir := filepath.Join(storage, templateID)
	return os.RemoveAll(targetDir)
}

func extractArchive(archivePath, targetDir, archiveType string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	if archiveType == "img" {
		destImg := filepath.Join(targetDir, "rootfs.img")
		return os.Rename(archivePath, destImg)
	}

	bbCandidates := []string{
		"/data/local/Droidspaces/bin/busybox",
		"/data/adb/ksu/bin/busybox",
		"/data/adb/magisk/busybox",
		"busybox",
	}
	var busyboxPath string
	for _, b := range bbCandidates {
		if _, err := os.Stat(b); err == nil {
			busyboxPath = b
			break
		} else if p, err := exec.LookPath(b); err == nil {
			busyboxPath = p
			break
		}
	}

	if busyboxPath != "" {
		cmd := exec.Command(busyboxPath, "tar", "-xf", archivePath, "-C", targetDir)
		if _, err := cmd.CombinedOutput(); err == nil {
			return nil
		}
	}

	var cmd *exec.Cmd
	if archiveType == "tar.gz" {
		cmd = exec.Command("tar", "-xzf", archivePath, "-C", targetDir)
	} else {
		cmd = exec.Command("tar", "-xJf", archivePath, "-C", targetDir)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar extraction failed: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func StartDownload(templateID string) error {
	downloadLock.Lock()
	if activeJob != "" {
		downloadLock.Unlock()
		return fmt.Errorf("job %s already running", activeJob)
	}
	activeJob = templateID
	jobProgress = "Downloading..."
	downloadLock.Unlock()

	var t *model.TemplateInfo
	for _, item := range defaultCatalog {
		if item.ID == templateID {
			t = &item
			break
		}
	}
	if t == nil {
		downloadLock.Lock()
		activeJob = ""
		downloadLock.Unlock()
		return fmt.Errorf("template %s not found", templateID)
	}

	go func() {
		defer func() {
			downloadLock.Lock()
			activeJob = ""
			jobProgress = ""
			downloadLock.Unlock()
		}()

		storage := GetStorageDir()
		targetDir := filepath.Join(storage, t.ID)
		tmpArchive := filepath.Join(storage, t.ID+"_temp."+t.Type)

		// 1. Download archive
		out, err := os.Create(tmpArchive)
		if err != nil {
			return
		}

		resp, err := http.Get(t.URL)
		if err != nil {
			out.Close()
			_ = os.Remove(tmpArchive)
			return
		}

		if _, err := io.Copy(out, resp.Body); err != nil {
			resp.Body.Close()
			out.Close()
			_ = os.Remove(tmpArchive)
			return
		}
		resp.Body.Close()
		out.Close()

		// 2. Extract archive
		downloadLock.Lock()
		jobProgress = "Extracting..."
		downloadLock.Unlock()

		_ = extractArchive(tmpArchive, targetDir, t.Type)
		_ = os.Remove(tmpArchive)
	}()

	return nil
}
