package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/config"
	"github.com/latifangren/droidspaces-webui/internal/model"
)

type diskUsageCacheEntry struct {
	formatted string
	bytes     int64
	cachedAt  time.Time
}

var (
	diskUsageCache   = make(map[string]diskUsageCacheEntry)
	diskUsageCacheMu sync.RWMutex
)

// ExtractJSON safely isolates the first outer JSON object {...}
// to strip any ANSI escape codes or command banners printed by the runtime.
func ExtractJSON(s string) string {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}

// FormatBytes converts raw byte count to human-readable string (e.g. 2.1 GB, 450 MB)
func FormatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// GetContainerDiskSize calculates or retrieves cached disk usage for a rootfs path
func GetContainerDiskSize(rootfsPath string) (string, int64) {
	if rootfsPath == "" {
		return "0 B", 0
	}

	diskUsageCacheMu.RLock()
	if entry, ok := diskUsageCache[rootfsPath]; ok {
		if time.Since(entry.cachedAt) < 30*time.Second {
			diskUsageCacheMu.RUnlock()
			return entry.formatted, entry.bytes
		}
	}
	diskUsageCacheMu.RUnlock()

	stat, err := os.Stat(rootfsPath)
	if err != nil {
		return "0 B", 0
	}

	var totalBytes int64
	if !stat.IsDir() {
		// Standalone image (.img)
		totalBytes = stat.Size()
	} else {
		// Try using busybox du for high speed
		bbCandidates := []string{
			"/data/local/Droidspaces/bin/busybox",
			"/data/adb/modules/droidspaces/bin/busybox",
			"busybox",
		}
		var bbPath string
		for _, bb := range bbCandidates {
			if _, err := os.Stat(bb); err == nil {
				bbPath = bb
				break
			}
		}

		calculated := false
		if bbPath != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, bbPath, "du", "-sk", rootfsPath).Output()
			if err == nil {
				parts := strings.Fields(string(out))
				if len(parts) > 0 {
					if kb, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
						totalBytes = kb * 1024
						calculated = true
					}
				}
			}
		}

		if !calculated {
			// Fast WalkDir fallback with entry cap
			var count int
			_ = filepath.WalkDir(rootfsPath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				count++
				if count > 30000 {
					return filepath.SkipAll
				}
				if !d.IsDir() {
					info, err := d.Info()
					if err == nil {
						totalBytes += info.Size()
					}
				}
				return nil
			})
		}
	}

	formatted := FormatBytes(totalBytes)
	diskUsageCacheMu.Lock()
	diskUsageCache[rootfsPath] = diskUsageCacheEntry{
		formatted: formatted,
		bytes:     totalBytes,
		cachedAt:  time.Now(),
	}
	diskUsageCacheMu.Unlock()

	return formatted, totalBytes
}

// ParsePortMappings extracts individual port forward strings from config
func ParsePortMappings(rawPorts string) []string {
	if rawPorts == "" {
		return nil
	}
	entries := strings.FieldsFunc(rawPorts, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	var result []string
	for _, e := range entries {
		trimmed := strings.TrimSpace(e)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ParsePortMatrix builds an aggregated matrix of all port forwards across containers
func ParsePortMatrix(running []model.ContainerSummary, stopped []model.ContainerSummary) []model.PortMatrixEntry {
	var matrix []model.PortMatrixEntry
	all := append(append([]model.ContainerSummary{}, running...), stopped...)

	for _, c := range all {
		cfg, err := config.ReadContainerConfig(c.Name)
		if err != nil {
			continue
		}
		rawPorts := cfg["port_forwards"]
		if rawPorts == "" {
			rawPorts = cfg["port"]
		}
		if rawPorts == "" {
			continue
		}

		entries := ParsePortMappings(rawPorts)
		for _, p := range entries {
			proto := "TCP"
			cleanP := p
			if strings.Contains(cleanP, "/") {
				parts := strings.SplitN(cleanP, "/", 2)
				cleanP = parts[0]
				proto = strings.ToUpper(parts[1])
			}

			var hostPort, containerPort string
			if strings.Contains(cleanP, ":") {
				parts := strings.SplitN(cleanP, ":", 2)
				hostPort = parts[0]
				containerPort = parts[1]
			} else {
				hostPort = cleanP
				containerPort = cleanP
			}

			status := "inactive"
			if c.Status == "running" {
				status = "active"
			}

			matrix = append(matrix, model.PortMatrixEntry{
				HostPort:      hostPort,
				Protocol:      proto,
				ContainerName: c.Name,
				ContainerPort: containerPort,
				ContainerIP:   c.IP,
				Status:        status,
			})
		}
	}

	sort.Slice(matrix, func(i, j int) bool {
		if matrix[i].HostPort == matrix[j].HostPort {
			return matrix[i].ContainerName < matrix[j].ContainerName
		}
		// Try numeric sort on host port if integer
		pi, errI := strconv.Atoi(matrix[i].HostPort)
		pj, errJ := strconv.Atoi(matrix[j].HostPort)
		if errI == nil && errJ == nil {
			return pi < pj
		}
		return matrix[i].HostPort < matrix[j].HostPort
	})

	return matrix
}
