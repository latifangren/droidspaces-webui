package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type configCacheEntry struct {
	modTime time.Time
	data    map[string]string
}

var (
	cacheMu  sync.RWMutex
	cfgCache = make(map[string]configCacheEntry)
)

// GetContainersDirs returns candidate directories where Droidspaces workspaces live.
func GetContainersDirs() []string {
	return []string{
		"/data/local/Droidspaces/Containers",
		"/var/lib/Droidspaces/Containers",
		"/tmp/droidspaces/Containers",
	}
}

// ReadContainerConfig parses key=value from a container's container.config with in-memory caching.
func ReadContainerConfig(name string) (map[string]string, error) {
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return nil, fmt.Errorf("invalid container name: %s", name)
	}

	for _, cdir := range GetContainersDirs() {
		cfgPath := filepath.Join(cdir, name, "container.config")
		stat, err := os.Stat(cfgPath)
		if err != nil {
			continue
		}

		// Fast path: check in-memory cache if file modification time has not changed
		cacheMu.RLock()
		entry, found := cfgCache[cfgPath]
		cacheMu.RUnlock()

		if found && !stat.ModTime().After(entry.modTime) {
			res := make(map[string]string, len(entry.data))
			for k, v := range entry.data {
				res[k] = v
			}
			return res, nil
		}

		// Slow path: parse file and update cache
		f, err := os.Open(cfgPath)
		if err != nil {
			continue
		}
		cfg := make(map[string]string)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			cfg[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
		f.Close()

		cacheMu.Lock()
		cfgCache[cfgPath] = configCacheEntry{
			modTime: stat.ModTime(),
			data:    cfg,
		}
		cacheMu.Unlock()

		res := make(map[string]string, len(cfg))
		for k, v := range cfg {
			res[k] = v
		}
		return res, nil
	}
	return nil, fmt.Errorf("container.config not found for %s", name)
}

// WriteContainerConfigKeys updates or appends key-value pairs in container.config and invalidates cache.
func WriteContainerConfigKeys(name string, updates map[string]string) error {
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("invalid container name: %s", name)
	}
	var targetPath string
	for _, cdir := range GetContainersDirs() {
		candidate := filepath.Join(cdir, name, "container.config")
		if _, err := os.Stat(candidate); err == nil {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		dirs := GetContainersDirs()
		if len(dirs) == 0 {
			return fmt.Errorf("no containers directory configured")
		}
		containerDir := filepath.Join(dirs[0], name)
		_ = os.MkdirAll(containerDir, 0755)
		targetPath = filepath.Join(containerDir, "container.config")
	}

	var existingLines []string
	if data, err := os.ReadFile(targetPath); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			existingLines = append(existingLines, scanner.Text())
		}
	}

	applied := make(map[string]bool)
	var newLines []string

	for _, line := range existingLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || !strings.Contains(trimmed, "=") {
			newLines = append(newLines, line)
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		k := strings.TrimSpace(parts[0])
		if newVal, ok := updates[k]; ok {
			newLines = append(newLines, fmt.Sprintf("%s=%s", k, newVal))
			applied[k] = true
		} else {
			newLines = append(newLines, line)
		}
	}

	for k, v := range updates {
		if !applied[k] {
			newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
		}
	}

	content := strings.Join(newLines, "\n") + "\n"
	err := os.WriteFile(targetPath, []byte(content), 0644)
	if err == nil {
		// Invalidate cache immediately
		cacheMu.Lock()
		delete(cfgCache, targetPath)
		cacheMu.Unlock()
	}
	return err
}
