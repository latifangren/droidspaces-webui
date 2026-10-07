package templates

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
		Arch:        "aarch64",
		Version:     "3.20.3",
		URL:         "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/aarch64/alpine-minirootfs-3.20.3-aarch64.tar.gz",
		Type:        "tar.gz",
		SizeMB:      4,
		Description: "Super lightweight, instant startup, ideal for microservices and Docker engine.",
	},
	{
		ID:          "debian-12",
		Name:        "Debian 12 Bookworm (Full Systemd)",
		Distro:      "debian",
		Arch:        "arm64",
		Version:     "12",
		URL:         "https://images.linuxcontainers.org/images/debian/bookworm/arm64/default/rootfs.tar.xz",
		Type:        "tar.xz",
		SizeMB:      95,
		Description: "Rock-solid stability, full systemd PID 1 support, perfect homelab base.",
	},
	{
		ID:          "ubuntu-24.04",
		Name:        "Ubuntu 24.04 LTS Noble Numbat",
		Distro:      "ubuntu",
		Arch:        "arm64",
		Version:     "24.04",
		URL:         "https://images.linuxcontainers.org/images/ubuntu/noble/arm64/default/rootfs.tar.xz",
		Type:        "tar.xz",
		SizeMB:      115,
		Description: "Latest LTS Ubuntu with modern packages, Python 3.12, systemd, and broad toolchain.",
	},
	{
		ID:          "arch-linux",
		Name:        "Arch Linux ARM (Rolling Release)",
		Distro:      "arch",
		Arch:        "aarch64",
		Version:     "rolling",
		URL:         "http://os.archlinuxarm.org/os/ArchLinuxARM-aarch64-latest.tar.gz",
		Type:        "tar.gz",
		SizeMB:      450,
		Description: "Bleeding-edge rolling release, pacman package manager, full flexibility.",
	},
	{
		ID:          "openwrt-23.05",
		Name:        "OpenWrt 23.05 (Network Gateway)",
		Distro:      "openwrt",
		Arch:        "arm64",
		Version:     "23.05.4",
		URL:         "https://downloads.openwrt.org/releases/23.05.4/targets/armsr/armv8/openwrt-23.05.4-armsr-armv8-rootfs.tar.gz",
		Type:        "tar.gz",
		SizeMB:      30,
		Description: "Dedicated router/firewall container for Droidspaces --net=gateway VPN killswitch.",
	},
}

func GetStorageDir() string {
	dirs := []string{
		"/data/local/Droidspaces/rootfs",
		"/data/adb/droidspaces/rootfs",
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
		}
	}
	return res
}

func GetDownloadStatus() (string, string) {
	downloadLock.Lock()
	defer downloadLock.Unlock()
	return activeJob, jobProgress
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
		defer out.Close()

		resp, err := http.Get(t.URL)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		if _, err := io.Copy(out, resp.Body); err != nil {
			return
		}
		out.Close()

		// 2. Extract archive
		_ = os.MkdirAll(targetDir, 0755)
		var cmd *exec.Cmd
		if t.Type == "tar.gz" {
			cmd = exec.Command("tar", "-xzf", tmpArchive, "-C", targetDir)
		} else {
			cmd = exec.Command("tar", "-xJf", tmpArchive, "-C", targetDir)
		}
		_ = cmd.Run()
		_ = os.Remove(tmpArchive)
	}()

	return nil
}
