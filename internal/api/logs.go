package api

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/config"
)

const (
	MaxLogSizeBytes   = 2 * 1024 * 1024 // 2 MB
	RetainedLogLines  = 2500            // Keep last 2500 lines on truncate
)

func getLogDirectories() []string {
	if custom := os.Getenv("DROIDSPACES_LOGS_DIR"); custom != "" {
		return []string{custom}
	}
	return []string{
		"/data/local/Droidspaces/Logs",
		"/var/lib/Droidspaces/Logs",
		"/data/adb/modules/droidspaces",
	}
}

// StartLogAutoTruncate runs periodic log inspection to prevent storage bloat.
func StartLogAutoTruncate() {
	go func() {
		// Run once shortly after startup
		time.Sleep(10 * time.Second)
		RotateAndTruncateLogs()

		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			RotateAndTruncateLogs()
		}
	}()
}

// RotateAndTruncateLogs inspects log files and trims any exceeding MaxLogSizeBytes.
func RotateAndTruncateLogs() {
	for _, dir := range getLogDirectories() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, ent := range entries {
			fullPath := filepath.Join(dir, ent.Name())
			if ent.IsDir() {
				// Container subdirectories: check console and log
				for _, sub := range []string{"console", "log"} {
					subFile := filepath.Join(fullPath, sub)
					truncateFileIfNeeded(subFile)
				}
			} else if strings.HasSuffix(ent.Name(), ".log") {
				truncateFileIfNeeded(fullPath)
			}
		}
	}
}

func truncateFileIfNeeded(path string) {
	stat, err := os.Stat(path)
	if err != nil || stat.Size() <= MaxLogSizeBytes {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) <= RetainedLogLines {
		return
	}

	// Keep last RetainedLogLines lines
	keptLines := lines[len(lines)-RetainedLogLines:]
	trimmedContent := strings.Join(keptLines, "\n") + "\n"

	err = os.WriteFile(path, []byte(trimmedContent), 0644)
	if err != nil {
		log.Printf("[logs] failed to truncate %s: %v", path, err)
	} else {
		log.Printf("[logs] auto-truncated %s from %d bytes down to %d lines", path, stat.Size(), len(keptLines))
	}
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logDirs := getLogDirectories()

	// Clear/truncate action: POST /api/logs?file=...&action=clear
	if r.Method == http.MethodPost {
		fileParam := r.URL.Query().Get("file")
		action := r.URL.Query().Get("action")
		if action == "clear" && fileParam != "" {
			if !safeLogPathRegex.MatchString(fileParam) || strings.Contains(fileParam, "..") {
				s.sendJSON(w, http.StatusBadRequest, nil, "invalid log file path")
				return
			}
			cleanParam := filepath.Clean(filepath.FromSlash(fileParam))
			for _, dir := range logDirs {
				candidate := filepath.Join(dir, cleanParam)
				if _, err := os.Stat(candidate); err == nil {
					_ = os.WriteFile(candidate, []byte(""), 0644)
					s.sendJSON(w, http.StatusOK, map[string]string{"message": "log cleared"}, "")
					return
				}
			}
			s.sendJSON(w, http.StatusNotFound, nil, "log file not found")
			return
		}
	}

	fileParam := r.URL.Query().Get("file")
	if fileParam == "" {
		systemFiles := make(map[string]bool)
		containerFiles := make(map[string]bool)

		// Discover active containers to filter out orphans
		knownContainers := make(map[string]bool)
		for _, cdir := range config.GetContainersDirs() {
			entries, err := os.ReadDir(cdir)
			if err != nil {
				continue
			}
			for _, ent := range entries {
				if ent.IsDir() {
					knownContainers[ent.Name()] = true
				}
			}
		}

		for _, dir := range logDirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, ent := range entries {
				name := ent.Name()
				if !ent.IsDir() {
					if strings.HasSuffix(name, ".log") {
						systemFiles[name] = true
					}
				} else {
					// Only list logs for containers that currently exist
					if !knownContainers[name] {
						continue
					}
					subDir := filepath.Join(dir, name)
					for _, target := range []string{"console", "log"} {
						if _, err := os.Stat(filepath.Join(subDir, target)); err == nil {
							containerFiles[filepath.Join(name, target)] = true
						}
					}
				}
			}
		}

		var sysList []string
		for f := range systemFiles {
			sysList = append(sysList, f)
		}
		sort.Strings(sysList)

		var cList []string
		for f := range containerFiles {
			cList = append(cList, filepath.ToSlash(f))
		}
		sort.Strings(cList)

		allFiles := append([]string{}, sysList...)
		allFiles = append(allFiles, cList...)

		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"files":           allFiles,
			"system_files":    sysList,
			"container_files": cList,
		}, "")
		return
	}

	if !safeLogPathRegex.MatchString(fileParam) || strings.Contains(fileParam, "..") {
		s.sendJSON(w, http.StatusBadRequest, nil, "invalid log file path")
		return
	}
	cleanParam := filepath.Clean(filepath.FromSlash(fileParam))

	var targetPath string
	for _, dir := range logDirs {
		candidate := filepath.Join(dir, cleanParam)
		if _, err := os.Stat(candidate); err == nil {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		s.sendJSON(w, http.StatusNotFound, nil, "log file not found: "+fileParam)
		return
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		line = ansiRegex.ReplaceAllString(line, "")
		lines = append(lines, line)
	}

	totalLines := len(lines)
	maxLines := 500
	if totalLines > maxLines {
		lines = lines[totalLines-maxLines:]
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"content":     strings.Join(lines, "\n"),
		"total_lines": totalLines,
		"shown_lines": len(lines),
		"size_bytes":  len(data),
	}, "")
}
