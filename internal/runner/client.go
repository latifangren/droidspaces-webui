package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

type Client struct {
	binPath string
}

func NewClient() *Client {
	candidates := []string{
		"/data/local/Droidspaces/bin/droidspaces",
		"/data/adb/ksu/bin/droidspaces",
		"/system/bin/droidspaces",
		"droidspaces",
	}

	bin := "droidspaces"
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			bin = c
			break
		}
	}

	return &Client{binPath: bin}
}

func (c *Client) BinaryPath() string {
	return c.binPath
}

func (c *Client) run(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		outErr := strings.TrimSpace(stderr.String())
		if outErr == "" {
			outErr = strings.TrimSpace(stdout.String())
		}
		return stdout.String(), fmt.Errorf("%w: %s", err, outErr)
	}
	return stdout.String(), nil
}

func (c *Client) getContainersDirs() []string {
	return []string{
		"/data/local/Droidspaces/Containers",
		"/var/lib/Droidspaces/Containers",
		"/tmp/droidspaces/Containers",
	}
}

func (c *Client) Show() (*model.ShowResult, error) {
	out, err := c.run("show", "--format")
	if err != nil {
		return nil, err
	}

	var res model.ShowResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
		return nil, fmt.Errorf("failed to parse show output: %w", err)
	}

	seenPID := make(map[int]bool)
	seenName := make(map[string]bool)
	var dedupedRunning []model.ContainerSummary

	for _, item := range res.Running {
		if item.PID > 0 && seenPID[item.PID] {
			continue
		}
		if item.Name != "" && seenName[item.Name] {
			continue
		}
		item.Status = "running"
		if item.CPUPermill > 0 {
			item.CPUPercent = float64(item.CPUPermill) / 10.0
		}
		if item.PID > 0 {
			seenPID[item.PID] = true
		}
		if item.Name != "" {
			seenName[item.Name] = true
		}
		dedupedRunning = append(dedupedRunning, item)
	}
	res.Running = dedupedRunning

	// Scan workspace Containers directory for stopped containers
	for _, cdir := range c.getContainersDirs() {
		entries, err := os.ReadDir(cdir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			name := ent.Name()
			if seenName[name] {
				continue
			}

			// Read container.config
			cfgPath := filepath.Join(cdir, name, "container.config")
			stoppedSummary := model.ContainerSummary{
				Name:   name,
				Status: "stopped",
				PID:    0,
			}

			if f, err := os.Open(cfgPath); err == nil {
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
						continue
					}
					kv := strings.SplitN(line, "=", 2)
					key := strings.TrimSpace(kv[0])
					val := strings.TrimSpace(kv[1])
					switch key {
					case "hostname":
						stoppedSummary.Hostname = val
					case "rootfs_path":
						stoppedSummary.RootFS = val
					case "ip":
						stoppedSummary.IP = val
					}
				}
				f.Close()
			}

			res.Stopped = append(res.Stopped, stoppedSummary)
			seenName[name] = true
		}
	}

	res.Total = len(res.Running) + len(res.Stopped)
	return &res, nil
}

func (c *Client) Info(name string) (map[string]interface{}, error) {
	out, err := c.run("info", "--name="+name, "--format")
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
		return nil, fmt.Errorf("failed to parse info output: %w", err)
	}
	return res, nil
}
func (c *Client) cleanupRogueConfigs() {
	dirs := []string{
		"/data/local/Droidspaces/rootfs",
		"/data/adb/droidspaces/rootfs",
		"/var/lib/Droidspaces/rootfs",
		"/tmp/droidspaces/rootfs",
	}
	for _, d := range dirs {
		_ = os.Remove(filepath.Join(d, "container.config"))
	}
}

func (c *Client) pruneStalePID(name string) {
	c.cleanupRogueConfigs()
	pidsDirs := []string{
		"/data/local/Droidspaces/Pids",
		"/var/lib/Droidspaces/Pids",
	}
	for _, pdir := range pidsDirs {
		pidFile := filepath.Join(pdir, name+".pid")
		data, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}
		pidStr := strings.TrimSpace(string(data))
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			_ = os.Remove(pidFile)
			continue
		}

		// Check if process is dead in /proc
		procPath := fmt.Sprintf("/proc/%d", pid)
		if _, err := os.Stat(procPath); err != nil {
			_ = os.Remove(pidFile)
			continue
		}

		// Check container.config inside the process root
		cfgPath := filepath.Join(procPath, "root", "run", "droidspaces", "container.config")
		if cfgData, err := os.ReadFile(cfgPath); err == nil {
			matched := false
			for _, line := range strings.Split(string(cfgData), "\n") {
				if strings.TrimSpace(line) == "name="+name {
					matched = true
					break
				}
			}
			if !matched {
				// The running PID belongs to a different container!
				_ = os.Remove(pidFile)
			}
		} else {
			// Not a droidspaces container process
			_ = os.Remove(pidFile)
		}
	}
}

func (c *Client) Start(req model.StartRequest) error {
	c.cleanupRogueConfigs()
	c.pruneStalePID(req.Name)
	defer c.cleanupRogueConfigs()
	args := []string{"start", "--name=" + req.Name}
	if req.RootFS != "" {
		args = append(args, "--rootfs="+req.RootFS)
	}
	if req.RootFSImg != "" {
		args = append(args, "--rootfs-img="+req.RootFSImg)
	}
	if req.Hostname != "" {
		args = append(args, "--hostname="+req.Hostname)
	}
	if req.Conf != "" {
		args = append(args, "--conf="+req.Conf)
	}
	if req.Net != "" {
		args = append(args, "--net="+req.Net)
	}
	if req.Gateway != "" {
		args = append(args, "--gateway="+req.Gateway)
	}
	if req.NATIP != "" {
		args = append(args, "--nat-ip="+req.NATIP)
	}
	if req.Upstream != "" {
		args = append(args, "--upstream="+req.Upstream)
	}
	for _, p := range req.Port {
		args = append(args, "--port="+p)
	}
	if req.DNS != "" {
		args = append(args, "--dns="+req.DNS)
	}
	if req.DisableIPv6 {
		args = append(args, "--disable-ipv6")
	}
	if req.AndroidStorage {
		args = append(args, "-S")
	}
	if req.HWAccess {
		args = append(args, "-H")
	}
	if req.GPU {
		args = append(args, "--gpu")
	}
	if req.TermuxX11 {
		args = append(args, "--termux-x11")
	}
	if req.VirGL {
		args = append(args, "--virgl")
	}
	if req.PulseAudio {
		args = append(args, "--pulse-audio")
	}
	if req.SELinuxPermissive {
		args = append(args, "-P")
	}
	if req.Volatile {
		args = append(args, "-V")
	}
	if req.ForceCgroupV1 {
		args = append(args, "--force-cgroupv1")
	}
	if req.Memory != "" {
		args = append(args, "--memory="+req.Memory)
	}
	if req.CPUs != "" {
		args = append(args, "--cpus="+req.CPUs)
	}
	if req.PIDsLimit != "" {
		args = append(args, "--pids-limit="+req.PIDsLimit)
	}
	if req.Privileged != "" {
		args = append(args, "--privileged="+req.Privileged)
	}
	if req.AllowSandboxing {
		args = append(args, "--allow-sandboxing")
	}
	for _, b := range req.Binds {
		args = append(args, "-B", b)
	}

	_, err := c.run(args...)
	return err
}

func (c *Client) Stop(name string) error {
	_, err := c.run("stop", "--name="+name)
	return err
}

func (c *Client) Restart(name string) error {
	_, err := c.run("restart", "--name="+name)
	return err
}

func (c *Client) Delete(name string) error {
	// 1. Stop if running
	_ = c.Stop(name)

	// 2. Remove container workspace directory
	var lastErr error
	deleted := false
	for _, cdir := range c.getContainersDirs() {
		target := filepath.Join(cdir, name)
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			if err := os.RemoveAll(target); err != nil {
				lastErr = err
			} else {
				deleted = true
			}
		}
	}
	if !deleted && lastErr != nil {
		return lastErr
	}
	return nil
}

func (c *Client) Check() (string, error) {
	return c.run("check")
}

func (c *Client) Scan() (string, error) {
	return c.run("scan")
}

func (c *Client) Exec(container, command, user string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	args := []string{"--name=" + container, "run"}
	if user != "" {
		args = append(args, "-u", user)
	}
	args = append(args, "sh", "-c", command)

	cmd := exec.CommandContext(ctx, c.binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, output)
	}
	return output, nil
}

func (c *Client) findHostShell() string {
	shells := []string{"/system/bin/sh", "/bin/sh", "/bin/bash", "sh"}
	for _, s := range shells {
		if strings.HasPrefix(s, "/") {
			if _, err := os.Stat(s); err == nil {
				return s
			}
		} else {
			if p, err := exec.LookPath(s); err == nil {
				return p
			}
		}
	}
	return "/bin/sh"
}

func (c *Client) ExecHost(command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sh := c.findHostShell()
	cmd := exec.CommandContext(ctx, sh, "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, output)
	}
	return output, nil
}
