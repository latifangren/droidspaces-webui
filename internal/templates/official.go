package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

const officialRepoURL = "https://raw.githubusercontent.com/Droidspaces/Droidspaces-rootfs-builder/main/rootfs.json"

type officialRootfsEntry struct {
	Author       string `json:"author"`
	Name         string `json:"name"`
	Distro       string `json:"distro"`
	Description  string `json:"description"`
	Architecture string `json:"architecture"`
	File         string `json:"file"`
	DownloadURL  string `json:"download_url"`
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"size_bytes"`
	Version      string `json:"version"`
	BuildDate    string `json:"build_date"`
}

var (
	officialCacheLock sync.RWMutex
	officialCached    []model.TemplateInfo
	officialLastFetch time.Time
)

// GetOfficialTemplates returns pre-built official Droidspaces rootfs templates.
// It tries to fetch the latest catalog from Droidspaces-rootfs-builder and falls back to embedded data.
func GetOfficialTemplates() []model.TemplateInfo {
	officialCacheLock.RLock()
	if len(officialCached) > 0 && time.Since(officialLastFetch) < 1*time.Hour {
		res := make([]model.TemplateInfo, len(officialCached))
		copy(res, officialCached)
		officialCacheLock.RUnlock()
		return res
	}
	officialCacheLock.RUnlock()

	// Fetch or fallback
	entries := fetchOfficialEntries()
	var templates []model.TemplateInfo

	for _, entry := range entries {
		// Filter for aarch64 (or 64-bit arm) by default
		if entry.Architecture != "aarch64" && entry.Architecture != "" {
			continue
		}
		t := convertOfficialEntry(entry)
		templates = append(templates, t)
	}

	officialCacheLock.Lock()
	officialCached = templates
	officialLastFetch = time.Now()
	res := make([]model.TemplateInfo, len(officialCached))
	copy(res, officialCached)
	officialCacheLock.Unlock()

	return res
}

func fetchOfficialEntries() []officialRootfsEntry {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", officialRepoURL, nil)
	if err == nil {
		req.Header.Set("User-Agent", "Droidspaces-WebUI/1.0")
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var entries []officialRootfsEntry
			if err := json.NewDecoder(resp.Body).Decode(&entries); err == nil && len(entries) > 0 {
				return entries
			}
		}
	}

	// Fallback to embedded data if offline or unreachable
	var fallback []officialRootfsEntry
	if err := json.Unmarshal([]byte(embeddedOfficialRootfsJSON), &fallback); err != nil {
		log.Printf("[templates] Failed to parse embedded official rootfs JSON: %v", err)
	}
	return fallback
}

func convertOfficialEntry(entry officialRootfsEntry) model.TemplateInfo {
	slug := strings.ToLower(entry.File)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	id := "official-" + slug

	nameLower := strings.ToLower(entry.Name)
	distroLower := strings.ToLower(entry.Distro)

	category := "server"
	environment := "Headless Server (CLI)"
	initSys := "systemd (PID 1)"
	minRAM := "256 MB"

	if strings.Contains(nameLower, "xfce") {
		category = "desktop"
		environment = "Desktop GUI (XFCE / Anland / Termux:X11)"
		minRAM = "1 GB"
	} else if strings.Contains(nameLower, "bspwm") {
		category = "desktop"
		environment = "Desktop GUI (Touch Catppuccin bspwm)"
		minRAM = "768 MB"
	} else if distroLower == "openwrt" {
		category = "network"
		environment = "Router (iptables-legacy + fw3 for Android)"
		initSys = "procd"
		minRAM = "64 MB"
	} else if strings.Contains(nameLower, "minimal") {
		environment = "Minimal Headless Server"
		minRAM = "128 MB"
	} else if strings.Contains(nameLower, "base") {
		environment = "Base Server (Docker + Dev Tools)"
		minRAM = "512 MB"
	}

	if distroLower == "alpine" || distroLower == "artix" {
		initSys = "openrc"
	}

	sizeMB := int(entry.SizeBytes / (1024 * 1024))
	if sizeMB == 0 && entry.SizeBytes > 0 {
		sizeMB = 1
	}

	desc := fmt.Sprintf("[OFFICIAL DROIDSPACES] %s", entry.Description)

	return model.TemplateInfo{
		ID:          id,
		Name:        entry.Name,
		Distro:      distroLower,
		Category:    category,
		Environment: environment,
		InitSystem:  initSys,
		MinRAM:      minRAM,
		Arch:        entry.Architecture,
		Version:     entry.Version,
		URL:         entry.DownloadURL,
		Type:        "tar.xz",
		SizeMB:      sizeMB,
		Description: desc,
		Origin:      "official",
		SHA256:      entry.SHA256,
		Author:      entry.Author,
	}
}

// verifySHA256 checks whether a downloaded file matches the expected hex hash.
func verifySHA256(filePath, expectedHash string) error {
	if expectedHash == "" {
		return nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}
