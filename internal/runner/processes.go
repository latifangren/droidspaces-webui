package runner

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/config"
	"github.com/latifangren/droidspaces-webui/internal/model"
)

// ListProcesses returns the active process tree from inside the container.
func (c *Client) ListProcesses(name string) ([]model.ProcessInfo, error) {
	cmd := "ps -eo pid,user,%cpu,%mem,comm 2>/dev/null || ps aux 2>/dev/null || ps -ef 2>/dev/null"
	out, err := c.Exec(name, cmd, "")
	if err != nil {
		return nil, err
	}

	var procs []model.ProcessInfo
	scanner := bufio.NewScanner(strings.NewReader(out))
	isFirstLine := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if isFirstLine {
			isFirstLine = false
			continue // skip header
		}

		fields := strings.Fields(line)
		if len(fields) >= 4 {
			pid, err := strconv.Atoi(fields[0])
			var user, cpu, mem, pcmd string
			if err == nil {
				// Format: PID USER %CPU %MEM COMM...
				if len(fields) >= 2 {
					user = fields[1]
				}
				if len(fields) >= 3 {
					cpu = fields[2]
				}
				if len(fields) >= 4 {
					mem = fields[3]
				}
				if len(fields) >= 5 {
					pcmd = strings.Join(fields[4:], " ")
				}
			} else {
				// Format: USER PID %CPU %MEM ...
				if len(fields) >= 2 {
					user = fields[0]
					pid, _ = strconv.Atoi(fields[1])
				}
				if len(fields) >= 4 {
					cpu = fields[2]
					mem = fields[3]
				}
				if len(fields) >= 11 {
					pcmd = strings.Join(fields[10:], " ")
				} else if len(fields) >= 5 {
					pcmd = strings.Join(fields[4:], " ")
				}
			}

			if pid > 0 {
				procs = append(procs, model.ProcessInfo{
					PID:     pid,
					User:    user,
					CPU:     cpu,
					Memory:  mem,
					Command: pcmd,
				})
			}
		}
	}

	return procs, nil
}

// KillProcess sends a signal to terminate a process inside the container.
func (c *Client) KillProcess(name string, pid int, signal int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}
	if signal <= 0 {
		signal = 9
	}
	cmd := fmt.Sprintf("kill -%d %d", signal, pid)
	_, err := c.Exec(name, cmd, "")
	return err
}

// ListUsers discovers accounts defined in /etc/passwd inside the container.
// It prioritizes direct native host reads of <rootfs>/etc/passwd (0 fork/exec latency, works even when container is stopped).
func (c *Client) ListUsers(name string) ([]string, error) {
	var content []byte

	// 1. Try reading directly from rootfs on host
	cfg, err := config.ReadContainerConfig(name)
	var passwdPath string
	if err == nil {
		rootfsPath := cfg["rootfs_path"]
		if rootfsPath != "" && !strings.HasSuffix(rootfsPath, ".img") {
			passwdPath = filepath.Join(rootfsPath, "etc", "passwd")
		}
	}
	if passwdPath == "" {
		dirs := config.GetContainersDirs()
		for _, d := range dirs {
			candidate := filepath.Join(d, name, "rootfs", "etc", "passwd")
			if _, err := os.Stat(candidate); err == nil {
				passwdPath = candidate
				break
			}
		}
	}

	if passwdPath != "" {
		content, _ = os.ReadFile(passwdPath)
	}

	// 2. Fallback to container namespace exec if rootfs is an image or unreadable
	if len(content) == 0 {
		out, errExec := c.Exec(name, "cut -d: -f1 /etc/passwd 2>/dev/null", "")
		if errExec != nil {
			return []string{"root"}, nil
		}
		content = []byte(out)
	}

	userMap := make(map[string]bool)
	var users []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		u := strings.TrimSpace(parts[0])
		if u != "" && !userMap[u] {
			userMap[u] = true
			if u != "root" {
				users = append(users, u)
			}
		}
	}

	sort.Strings(users)
	res := []string{"root"}
	res = append(res, users...)
	return res, nil
}
