package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var safeContainerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// SafeContainerName reports whether name is safe for workspace path construction.
func SafeContainerName(name string) bool {
	return safeContainerNameRegex.MatchString(name)
}

type configCacheEntry struct {
	modTime time.Time
	size    int64
	data    map[string]string
}

var (
	cacheMu    sync.RWMutex
	cfgCache   = make(map[string]configCacheEntry)
	writeMu    sync.Mutex
	writeLocks = make(map[string]*configLock)
)

type configLock struct {
	mu   sync.Mutex
	refs int
}

func lockConfigPath(path string) func() {
	writeMu.Lock()
	lock := writeLocks[path]
	if lock == nil {
		lock = &configLock{}
		writeLocks[path] = lock
	}
	lock.refs++
	writeMu.Unlock()
	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		writeMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(writeLocks, path)
		}
		writeMu.Unlock()
	}
}

// GetContainersDirs returns candidate directories where Droidspaces workspaces live.
func GetContainersDirs() []string {
	if custom := os.Getenv("DROIDSPACES_CONTAINERS_DIR"); custom != "" {
		return []string{custom}
	}
	return []string{
		"/data/local/Droidspaces/Containers",
		"/var/lib/Droidspaces/Containers",
		"/tmp/droidspaces/Containers",
	}
}

// ReadContainerConfig parses key=value from a container's container.config with in-memory caching.
func ReadContainerConfig(name string) (map[string]string, error) {
	if !safeContainerNameRegex.MatchString(name) {
		return nil, fmt.Errorf("invalid container name: %s", name)
	}

	for _, cdir := range GetContainersDirs() {
		cfgPath := filepath.Join(cdir, name, "container.config")
		stat, err := os.Stat(cfgPath)
		if err != nil {
			continue
		}

		cacheMu.RLock()
		entry, found := cfgCache[cfgPath]
		cacheMu.RUnlock()
		if found && stat.ModTime().Equal(entry.modTime) && stat.Size() == entry.size {
			return cloneConfig(entry.data), nil
		}

		f, err := os.Open(cfgPath)
		if err != nil {
			continue
		}
		openedInfo, err := f.Stat()
		if err != nil {
			f.Close()
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
		scanErr := scanner.Err()
		closeErr := f.Close()
		if scanErr != nil {
			return nil, fmt.Errorf("read container config %s: %w", cfgPath, scanErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close container config %s: %w", cfgPath, closeErr)
		}
		latestInfo, err := os.Stat(cfgPath)
		if err != nil || !os.SameFile(openedInfo, latestInfo) || !latestInfo.ModTime().Equal(openedInfo.ModTime()) || latestInfo.Size() != openedInfo.Size() {
			continue
		}

		cacheMu.Lock()
		cfgCache[cfgPath] = configCacheEntry{modTime: latestInfo.ModTime(), size: latestInfo.Size(), data: cfg}
		cacheMu.Unlock()
		return cloneConfig(cfg), nil
	}
	// Fallback: another writer replaced the file mid-read. Re-stat each directory
	// to detect a freshly renamed replacement we may have missed above.
	for _, cdir := range GetContainersDirs() {
		cfgPath := filepath.Join(cdir, name, "container.config")
		stat, err := os.Stat(cfgPath)
		if err != nil {
			continue
		}
		cacheMu.RLock()
		entry, found := cfgCache[cfgPath]
		cacheMu.RUnlock()
		if found && stat.ModTime().Equal(entry.modTime) && stat.Size() == entry.size {
			return cloneConfig(entry.data), nil
		}
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
		scanErr := scanner.Err()
		closeErr := f.Close()
		if scanErr != nil {
			return nil, fmt.Errorf("read container config %s: %w", cfgPath, scanErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close container config %s: %w", cfgPath, closeErr)
		}
		latestInfo, err := os.Stat(cfgPath)
		if err != nil {
			continue
		}
		cacheMu.Lock()
		cfgCache[cfgPath] = configCacheEntry{modTime: latestInfo.ModTime(), size: latestInfo.Size(), data: cfg}
		cacheMu.Unlock()
		return cloneConfig(cfg), nil
	}
	return nil, fmt.Errorf("container.config not found for %s", name)
}

func cloneConfig(cfg map[string]string) map[string]string {
	res := make(map[string]string, len(cfg))
	for k, v := range cfg {
		res[k] = v
	}
	return res
}

// WriteContainerConfigKeys updates or appends key-value pairs in container.config and invalidates cache.
func WriteContainerConfigKeys(name string, updates map[string]string) error {
	if !safeContainerNameRegex.MatchString(name) {
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
		targetPath = filepath.Join(dirs[0], name, "container.config")
	}

	unlock := lockConfigPath(targetPath)
	defer unlock()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create container directory: %w", err)
	}
	var existingLines []string
	mode := os.FileMode(0644)
	if info, err := os.Stat(targetPath); err == nil {
		mode = info.Mode().Perm()
		data, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("read existing container config: %w", err)
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			existingLines = append(existingLines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("scan existing container config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat existing container config: %w", err)
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
	if err := atomicReplaceFile(targetPath, []byte(content), mode); err != nil {
		return err
	}
	cacheMu.Lock()
	delete(cfgCache, targetPath)
	cacheMu.Unlock()
	return nil
}

// WriteConfigFile atomically replaces a complete container configuration file.
func WriteConfigFile(path string, data []byte, mode os.FileMode) error {
	unlock := lockConfigPath(path)
	defer unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := atomicReplaceFile(path, data, mode); err != nil {
		return err
	}
	cacheMu.Lock()
	delete(cfgCache, path)
	cacheMu.Unlock()
	return nil
}

func atomicReplaceFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".container-config-*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("set temporary config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace container config: %w", err)
	}
	return nil
}

// InvalidateCache clears cached container.config entries for the given container name.
func InvalidateCache(name string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	for _, cdir := range GetContainersDirs() {
		targetPath := filepath.Join(cdir, name, "container.config")
		delete(cfgCache, targetPath)
	}
}
