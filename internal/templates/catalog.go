package templates

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

func init() {
	// Android DNS Resolver Fix: Android lacks /etc/resolv.conf, causing Go pure resolver to fail on [::1]:53
	dnsServers := []string{
		"1.1.1.1:53",
		"8.8.8.8:53",
		"8.8.4.4:53",
		"1.0.0.1:53",
	}

	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 4 * time.Second}
			for _, s := range dnsServers {
				if conn, err := d.DialContext(ctx, "udp", s); err == nil {
					return conn, nil
				}
			}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
}

var (
	downloadLock sync.Mutex
	activeJob    string
	jobProgress  string
	jobError     string
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

func GetDownloadStatus() (string, string, string) {
	downloadLock.Lock()
	defer downloadLock.Unlock()
	return activeJob, jobProgress, jobError
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

type progressWriter struct {
	total      int64
	downloaded int64
	lastUpdate time.Time
	onProgress func(downloaded, total int64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloaded += int64(n)
	now := time.Now()
	if now.Sub(pw.lastUpdate) >= 300*time.Millisecond || pw.downloaded == pw.total {
		pw.lastUpdate = now
		if pw.onProgress != nil {
			pw.onProgress(pw.downloaded, pw.total)
		}
	}
	return n, nil
}

func setJobError(errMsg string) {
	log.Printf("[templates] ERROR: %s", errMsg)
	downloadLock.Lock()
	activeJob = ""
	jobProgress = ""
	jobError = errMsg
	downloadLock.Unlock()
}

func StartDownload(templateID string) error {
	downloadLock.Lock()
	if activeJob != "" {
		downloadLock.Unlock()
		return fmt.Errorf("job %s already running", activeJob)
	}
	activeJob = templateID
	jobProgress = "Connecting to server..."
	jobError = ""
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
		storage := GetStorageDir()
		targetDir := filepath.Join(storage, t.ID)
		tmpArchive := filepath.Join(storage, t.ID+"_temp."+t.Type)

		log.Printf("[templates] Starting download for %s from %s", t.ID, t.URL)

		// 1. Download archive with Android-safe DNS resolver & TLS config
		out, err := os.Create(tmpArchive)
		if err != nil {
			setJobError(fmt.Sprintf("create file failed: %v", err))
			return
		}

		dialer := &net.Dialer{
			Timeout:  15 * time.Second,
			Resolver: net.DefaultResolver,
		}

		client := &http.Client{
			Transport: &http.Transport{
				DialContext: dialer.DialContext,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Necessary on Android due to missing /etc/ssl/certs
				},
			},
			Timeout: 30 * time.Minute,
		}

		req, err := http.NewRequest("GET", t.URL, nil)
		if err != nil {
			out.Close()
			_ = os.Remove(tmpArchive)
			setJobError(fmt.Sprintf("create request failed: %v", err))
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 Droidspaces-WebUI/1.0 (Android aarch64)")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[templates] Go HTTP client failed (%v), attempting BusyBox wget fallback...", err)
			out.Close()
			_ = os.Remove(tmpArchive)

			// Fallback: Busybox wget
			bbCmd := exec.Command("/data/local/Droidspaces/bin/busybox", "wget", "-O", tmpArchive, t.URL)
			if errBB := bbCmd.Run(); errBB != nil {
				setJobError(fmt.Sprintf("HTTP & Busybox download failed: %v (busybox: %v)", err, errBB))
				return
			}
		} else {
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				out.Close()
				_ = os.Remove(tmpArchive)
				setJobError(fmt.Sprintf("server returned HTTP %d %s", resp.StatusCode, resp.Status))
				return
			}

			contentLength := resp.ContentLength
			pw := &progressWriter{
				total: contentLength,
				onProgress: func(downloaded, total int64) {
					downloadLock.Lock()
					if total > 0 {
						pct := float64(downloaded) / float64(total) * 100
						jobProgress = fmt.Sprintf("%.1f MB / %.1f MB (%.0f%%)",
							float64(downloaded)/(1024*1024),
							float64(total)/(1024*1024),
							pct)
					} else {
						jobProgress = fmt.Sprintf("%.1f MB downloaded", float64(downloaded)/(1024*1024))
					}
					downloadLock.Unlock()
				},
			}

			if _, err := io.Copy(out, io.TeeReader(resp.Body, pw)); err != nil {
				out.Close()
				_ = os.Remove(tmpArchive)
				setJobError(fmt.Sprintf("streaming failed: %v", err))
				return
			}
			out.Close()
		}

		log.Printf("[templates] Download complete for %s, extracting to %s", t.ID, targetDir)

		// 2. Extract archive
		downloadLock.Lock()
		jobProgress = "Extracting archive to rootfs folder..."
		downloadLock.Unlock()

		if err := extractArchive(tmpArchive, targetDir, t.Type); err != nil {
			_ = os.Remove(tmpArchive)
			setJobError(fmt.Sprintf("extraction failed: %v", err))
			return
		}
		_ = os.Remove(tmpArchive)

		log.Printf("[templates] Successfully installed template: %s", t.ID)

		downloadLock.Lock()
		activeJob = ""
		jobProgress = ""
		jobError = ""
		downloadLock.Unlock()
	}()

	return nil
}
