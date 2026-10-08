package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetContainersDirs returns candidate directories where Droidspaces workspaces live.
func GetContainersDirs() []string {
	return []string{
		"/data/local/Droidspaces/Containers",
		"/var/lib/Droidspaces/Containers",
		"/tmp/droidspaces/Containers",
	}
}

// ReadContainerConfig parses key=value from a container's container.config.
func ReadContainerConfig(name string) (map[string]string, error) {
	cfg := make(map[string]string)
	for _, cdir := range GetContainersDirs() {
		cfgPath := filepath.Join(cdir, name, "container.config")
		f, err := os.Open(cfgPath)
		if err != nil {
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			cfg[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
		return cfg, nil
	}
	return cfg, fmt.Errorf("container.config not found for %s", name)
}

// WriteContainerConfigKeys updates or appends key-value pairs in container.config.
func WriteContainerConfigKeys(name string, updates map[string]string) error {
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
	return os.WriteFile(targetPath, []byte(content), 0644)
}
