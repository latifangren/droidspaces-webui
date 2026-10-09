package runner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/latifangren/droidspaces-webui/internal/config"
	"github.com/latifangren/droidspaces-webui/internal/model"
)

var (
	safeServiceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.@-]{1,128}$`)
	initSysCacheMu       sync.RWMutex
	initSysCache         = make(map[string]string)
)
// DetectInitSystem detects whether the container is running systemd, openrc, or procd.
func (c *Client) DetectInitSystem(name string) string {
	initSysCacheMu.RLock()
	cached, found := initSysCache[name]
	initSysCacheMu.RUnlock()
	if found && cached != "" && cached != "unknown" {
		return cached
	}

	// 1. Fast static inspection of container rootfs (zero IPC overhead)
	if cfg, err := config.ReadContainerConfig(name); err == nil {
		rootfs := cfg["rootfs_path"]
		if rootfs != "" {
			if _, err := os.Stat(filepath.Join(rootfs, "etc/systemd")); err == nil {
				c.setInitSysCache(name, "systemd")
				return "systemd"
			}
			if _, err := os.Stat(filepath.Join(rootfs, "lib/systemd")); err == nil {
				c.setInitSysCache(name, "systemd")
				return "systemd"
			}
			if _, err := os.Stat(filepath.Join(rootfs, "etc/openwrt_release")); err == nil {
				c.setInitSysCache(name, "procd")
				return "procd"
			}
			if _, err := os.Stat(filepath.Join(rootfs, "etc/init.d")); err == nil {
				c.setInitSysCache(name, "openrc")
				return "openrc"
			}
		}
	}

	// 2. Fallback runtime probe only if static inspection is indeterminate
	checkCmd := "if [ -d /run/systemd/system ]; then echo systemd; " +
		"elif [ -d /run/openrc ] || [ -f /etc/init.d/openrc ] || [ -x /sbin/rc-service ]; then echo openrc; " +
		"elif [ -f /etc/openwrt_release ] || [ -x /sbin/procd ]; then echo procd; " +
		"elif command -v systemctl >/dev/null 2>&1; then echo systemd; " +
		"elif [ -d /etc/init.d ]; then echo openrc; " +
		"else echo unknown; fi"

	out, err := c.Exec(name, checkCmd, "")
	if err == nil {
		res := strings.TrimSpace(out)
		switch res {
		case "systemd", "openrc", "procd":
			c.setInitSysCache(name, res)
			return res
		}
	}

	return "unknown"
}

func (c *Client) setInitSysCache(name, initSys string) {
	initSysCacheMu.Lock()
	initSysCache[name] = initSys
	initSysCacheMu.Unlock()
}

// ListServices inspects services inside the container according to its active init system.
func (c *Client) ListServices(name string) ([]model.ServiceInfo, string, error) {
	initSys := c.DetectInitSystem(name)
	var services []model.ServiceInfo
	serviceMap := make(map[string]*model.ServiceInfo)

	switch initSys {
	case "systemd":
		unitFilesCmd := "systemctl list-unit-files --type=service --no-legend --no-pager 2>/dev/null"
		if out, err := c.Exec(name, unitFilesCmd, ""); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) >= 2 {
					sname := fields[0]
					senabled := fields[1]
					serviceMap[sname] = &model.ServiceInfo{
						Name:    sname,
						State:   "inactive",
						Enabled: senabled,
					}
				}
			}
		}

		unitsCmd := "systemctl list-units --type=service --all --no-legend --no-pager 2>/dev/null"
		if out, err := c.Exec(name, unitsCmd, ""); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) >= 4 {
					sname := fields[0]
					activeState := fields[2]
					subState := fields[3]
					var desc string
					if len(fields) > 4 {
						desc = strings.Join(fields[4:], " ")
					}
					st := activeState
					if subState == "running" {
						st = "running"
					}

					if item, ok := serviceMap[sname]; ok {
						item.State = st
						if desc != "" {
							item.Description = desc
						}
					} else {
						serviceMap[sname] = &model.ServiceInfo{
							Name:        sname,
							State:       st,
							Enabled:     "unknown",
							Description: desc,
						}
					}
				}
			}
		}

	case "openrc":
		runningCmd := "rc-status -a 2>/dev/null"
		if out, err := c.Exec(name, runningCmd, ""); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.Contains(line, "[") && strings.Contains(line, "]") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						sname := parts[0]
						state := strings.Trim(parts[1], "[]")
						serviceMap[sname] = &model.ServiceInfo{
							Name:    sname,
							State:   state,
							Enabled: "enabled",
						}
					}
				}
			}
		}

		listCmd := "rc-service -l 2>/dev/null || ls -1 /etc/init.d 2>/dev/null"
		if out, err := c.Exec(name, listCmd, ""); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				sname := strings.TrimSpace(scanner.Text())
				if sname == "" || strings.HasPrefix(sname, ".") {
					continue
				}
				if _, ok := serviceMap[sname]; !ok {
					serviceMap[sname] = &model.ServiceInfo{
						Name:    sname,
						State:   "stopped",
						Enabled: "disabled",
					}
				}
			}
		}

	case "procd":
		listCmd := "ls -1 /etc/init.d 2>/dev/null"
		if out, err := c.Exec(name, listCmd, ""); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				sname := strings.TrimSpace(scanner.Text())
				if sname == "" || strings.HasPrefix(sname, ".") {
					continue
				}
				serviceMap[sname] = &model.ServiceInfo{
					Name:    sname,
					State:   "unknown",
					Enabled: "unknown",
				}
			}
		}

	default:
		listCmd := "ls -1 /etc/init.d 2>/dev/null"
		if out, err := c.Exec(name, listCmd, ""); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				sname := strings.TrimSpace(scanner.Text())
				if sname == "" || strings.HasPrefix(sname, ".") {
					continue
				}
				serviceMap[sname] = &model.ServiceInfo{
					Name:    sname,
					State:   "unknown",
					Enabled: "unknown",
				}
			}
		}
	}

	for _, s := range serviceMap {
		services = append(services, *s)
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, initSys, nil
}

// ManageService performs lifecycle actions on a container service.
func (c *Client) ManageService(name, action, service string) error {
	if !safeServiceNameRegex.MatchString(service) {
		return fmt.Errorf("invalid service name: %s", service)
	}

	validActions := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
		"enable":  true,
		"disable": true,
		"mask":    true,
		"unmask":  true,
		"reload":  true,
	}
	if !validActions[action] {
		return fmt.Errorf("unsupported service action: %s", action)
	}

	initSys := c.DetectInitSystem(name)
	var cmd string

	switch initSys {
	case "systemd":
		cmd = fmt.Sprintf("systemctl %s %s", action, service)
	case "openrc":
		switch action {
		case "start", "stop", "restart", "reload":
			cmd = fmt.Sprintf("rc-service %s %s", service, action)
		case "enable":
			cmd = fmt.Sprintf("rc-update add %s default", service)
		case "disable":
			cmd = fmt.Sprintf("rc-update del %s default", service)
		default:
			return fmt.Errorf("action %s not supported on openrc", action)
		}
	case "procd":
		switch action {
		case "start", "stop", "restart", "enable", "disable", "reload":
			cmd = fmt.Sprintf("/etc/init.d/%s %s", service, action)
		default:
			return fmt.Errorf("action %s not supported on procd", action)
		}
	default:
		cmd = fmt.Sprintf("/etc/init.d/%s %s", service, action)
	}

	_, err := c.Exec(name, cmd, "")
	return err
}

// GetServiceJournal returns systemd journalctl entries or syslog fallback for a service.
func (c *Client) GetServiceJournal(name, service string, lines int) (string, error) {
	if !safeServiceNameRegex.MatchString(service) {
		return "", fmt.Errorf("invalid service name: %s", service)
	}
	if lines <= 0 || lines > 500 {
		lines = 100
	}

	initSys := c.DetectInitSystem(name)
	var cmd string
	if initSys == "systemd" {
		cmd = fmt.Sprintf("journalctl -u %s -n %d --no-pager 2>/dev/null", service, lines)
	} else {
		cmd = fmt.Sprintf("grep -i '%s' /var/log/messages 2>/dev/null | tail -n %d || grep -i '%s' /var/log/syslog 2>/dev/null | tail -n %d || dmesg | tail -n %d", service, lines, service, lines, lines)
	}

	return c.Exec(name, cmd, "")
}
