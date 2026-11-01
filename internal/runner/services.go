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
			if _, err := os.Stat(filepath.Join(rootfs, "etc/runit")); err == nil {
				c.setInitSysCache(name, "runit")
				return "runit"
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
		"elif [ -d /run/runit ] || command -v sv >/dev/null 2>&1; then echo runit; " +
		"elif command -v systemctl >/dev/null 2>&1; then echo systemd; " +
		"elif [ -d /etc/init.d ]; then echo openrc; " +
		"else echo unknown; fi"

	out, err := c.Exec(name, checkCmd, "")
	if err == nil {
		res := strings.TrimSpace(out)
		switch res {
		case "systemd", "openrc", "procd", "runit":
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

	case "runit":
		runitCmd := `for d in /etc/service /service /var/service /run/runit/service; do
			if [ -d "$d" ]; then
				for p in "$d"/*; do
					[ -d "$p" ] || continue
					sname=$(basename "$p")
					st=$(SVDIR="$d" sv status "$sname" 2>/dev/null)
					state="inactive"
					case "$st" in run:*) state="running" ;; esac
					boot="enabled"
					[ -f "$p/down" ] && boot="disabled"
					echo "SVC|$sname|$state|$boot|$st"
				done
				break
			fi
		done
		for d in /etc/sv /etc/runit/sv /usr/local/etc/sv; do
			if [ -d "$d" ]; then
				for p in "$d"/*; do
					[ -d "$p" ] || continue
					echo "INSTALLED|$(basename "$p")"
				done
				break
			fi
		done`
		if out, err := c.Exec(name, runitCmd, ""); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(out))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "SVC|") {
					parts := strings.Split(line, "|")
					if len(parts) >= 4 {
						sname := parts[1]
						state := parts[2]
						enabled := parts[3]
						desc := ""
						if len(parts) >= 5 {
							desc = parts[4]
						}
						serviceMap[sname] = &model.ServiceInfo{
							Name:        sname,
							State:       state,
							Enabled:     enabled,
							Description: desc,
						}
					}
				} else if strings.HasPrefix(line, "INSTALLED|") {
					sname := strings.TrimPrefix(line, "INSTALLED|")
					if _, exists := serviceMap[sname]; !exists && sname != "" {
						serviceMap[sname] = &model.ServiceInfo{
							Name:    sname,
							State:   "inactive",
							Enabled: "disabled",
						}
					}
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
	case "runit":
		switch action {
		case "start":
			cmd = fmt.Sprintf("sv up %s", service)
		case "stop":
			cmd = fmt.Sprintf("sv down %s", service)
		case "restart":
			cmd = fmt.Sprintf("sv restart %s", service)
		case "reload":
			cmd = fmt.Sprintf("sv reload %s", service)
		case "enable":
			cmd = fmt.Sprintf(`for ad in /etc/service /service /var/service /run/runit/service; do
				if [ -d "$ad" ]; then
					for sd in /etc/sv /etc/runit/sv /usr/local/etc/sv; do
						if [ -d "$sd/%s" ]; then
							ln -sfn "$sd/%s" "$ad/%s"
							rm -f "$ad/%s/down" 2>/dev/null
							break 2
						fi
					done
				fi
			done`, service, service, service, service)
		case "disable":
			cmd = fmt.Sprintf(`for ad in /etc/service /service /var/service /run/runit/service; do
				if [ -L "$ad/%s" ] || [ -e "$ad/%s" ]; then
					sv down "$ad/%s" 2>/dev/null
					rm -f "$ad/%s"
					break
				fi
			done`, service, service, service, service)
		default:
			return fmt.Errorf("action %s not supported on runit", action)
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
	} else if initSys == "runit" {
		cmd = fmt.Sprintf(`(for d in /etc/service /service /var/service /run/runit/service; do
			if [ -d "$d/%s/log/main" ]; then
				tail -n %d "$d/%s/log/main/current" 2>/dev/null
				exit 0
			fi
		done) || grep -i '%s' /var/log/messages 2>/dev/null | tail -n %d || grep -i '%s' /var/log/syslog 2>/dev/null | tail -n %d || dmesg | tail -n %d`, service, lines, service, service, lines, service, lines, lines)
	} else {
		cmd = fmt.Sprintf("grep -i '%s' /var/log/messages 2>/dev/null | tail -n %d || grep -i '%s' /var/log/syslog 2>/dev/null | tail -n %d || dmesg | tail -n %d", service, lines, service, lines, lines)
	}

	return c.Exec(name, cmd, "")
}
