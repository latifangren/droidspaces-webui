package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/config"
	"github.com/latifangren/droidspaces-webui/internal/hardware"
	"github.com/latifangren/droidspaces-webui/internal/model"
	"github.com/latifangren/droidspaces-webui/internal/network"
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

// Config Delegation
func (c *Client) ReadContainerConfig(name string) (map[string]string, error) {
	return config.ReadContainerConfig(name)
}

func (c *Client) WriteContainerConfigKeys(name string, updates map[string]string) error {
	return config.WriteContainerConfigKeys(name, updates)
}

// Boot Priority Delegation
func (c *Client) GetBootPriorities() ([]model.BootPriorityItem, error) {
	show, _ := c.Show()
	runningMap := make(map[string]bool)
	if show != nil {
		for _, r := range show.Running {
			runningMap[r.Name] = true
		}
	}
	return config.GetBootPriorities(runningMap)
}

func (c *Client) SetBootPriorities(items []model.BootPriorityItem) error {
	return config.SetBootPriorities(items)
}

// Network Delegation
func (c *Client) ListHostNetworkInterfaces() ([]model.NetworkInterfaceInfo, error) {
	return network.ListHostNetworkInterfaces()
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
		item.CPUPercent = hardware.GetContainerCPU(item.Name, item.PID)
		if item.CPUPercent == 0 && item.CPUPermill > 0 {
			item.CPUPercent = float64(item.CPUPermill) / 10.0
		}
		if item.PID > 0 {
			seenPID[item.PID] = true
		}
		if item.Name != "" {
			seenName[item.Name] = true
			if cfg, err := config.ReadContainerConfig(item.Name); err == nil {
				if cfg["run_at_boot"] == "1" || cfg["run_at_boot"] == "true" {
					item.RunAtBoot = true
				}
				if prio, err := strconv.Atoi(cfg["run_at_boot_priority"]); err == nil {
					item.RunAtBootPriority = prio
				}
				if item.RootFS == "" {
					item.RootFS = cfg["rootfs_path"]
				}
				rawPorts := cfg["port_forwards"]
				if rawPorts == "" {
					rawPorts = cfg["port"]
				}
				item.PortMappings = ParsePortMappings(rawPorts)
			}
			if item.RootFS != "" {
				item.DiskSize, item.DiskSizeBytes = GetContainerDiskSize(item.RootFS)
			}
			item.InitSystem = c.DetectInitSystem(item.Name)
		}
		dedupedRunning = append(dedupedRunning, item)
	}
	res.Running = dedupedRunning

	// Scan workspace Containers directory for stopped containers
	for _, cdir := range config.GetContainersDirs() {
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

			stoppedSummary := model.ContainerSummary{
				Name:   name,
				Status: "stopped",
				PID:    0,
			}

			if cfg, err := config.ReadContainerConfig(name); err == nil {
				stoppedSummary.Hostname = cfg["hostname"]
				stoppedSummary.RootFS = cfg["rootfs_path"]
				stoppedSummary.IP = cfg["ip"]
				if cfg["run_at_boot"] == "1" || cfg["run_at_boot"] == "true" {
					stoppedSummary.RunAtBoot = true
				}
				if prio, err := strconv.Atoi(cfg["run_at_boot_priority"]); err == nil {
					stoppedSummary.RunAtBootPriority = prio
				}
				rawPorts := cfg["port_forwards"]
				if rawPorts == "" {
					rawPorts = cfg["port"]
				}
				stoppedSummary.PortMappings = ParsePortMappings(rawPorts)
			}
			if stoppedSummary.RootFS != "" {
				stoppedSummary.DiskSize, stoppedSummary.DiskSizeBytes = GetContainerDiskSize(stoppedSummary.RootFS)
			}

			stoppedSummary.InitSystem = c.DetectInitSystem(name)
			res.Stopped = append(res.Stopped, stoppedSummary)
			seenName[name] = true
		}
	}

	res.PortMatrix = ParsePortMatrix(res.Running, res.Stopped)
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
	if cfg, err := config.ReadContainerConfig(name); err == nil {
		rootfs := cfg["rootfs_path"]
		if rootfs != "" {
			res["rootfs_path"] = rootfs
			dSize, dBytes := GetContainerDiskSize(rootfs)
			res["disk_size"] = dSize
			res["disk_size_bytes"] = dBytes
		}
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

		procPath := fmt.Sprintf("/proc/%d", pid)
		if _, err := os.Stat(procPath); err != nil {
			_ = os.Remove(pidFile)
			continue
		}

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
				_ = os.Remove(pidFile)
			}
		} else {
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
	if req.GatewayNet != "" {
		args = append(args, "--gateway-net="+req.GatewayNet)
	}
	if req.GatewayIface != "" {
		args = append(args, "--gateway-lan-ifname="+req.GatewayIface)
	}
	if req.GatewayBridge != "" {
		args = append(args, "--gateway-bridge="+req.GatewayBridge)
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
	if req.CustomInit != "" {
		args = append(args, "--init="+req.CustomInit)
	}
	for _, env := range req.EnvVars {
		args = append(args, "-e", env)
	}
	for _, b := range req.Binds {
		args = append(args, "-B", b)
	}

	_, err := c.run(args...)
	if err != nil {
		return err
	}

	// Persist boot settings if specified
	updates := make(map[string]string)
	if req.RunAtBoot {
		updates["run_at_boot"] = "1"
		updates["run_at_boot_priority"] = strconv.Itoa(req.RunAtBootPriority)
	} else {
		updates["run_at_boot"] = "0"
		updates["run_at_boot_priority"] = "0"
	}
	_ = config.WriteContainerConfigKeys(req.Name, updates)

	return nil
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
	_ = c.Stop(name)

	var lastErr error
	deleted := false
	for _, cdir := range config.GetContainersDirs() {
		target := filepath.Join(cdir, name)
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			if err := os.RemoveAll(target); err != nil {
				lastErr = err
			} else {
				deleted = true
			}
		}
	}

	for _, pdir := range []string{"/data/local/Droidspaces/Pids", "/var/lib/Droidspaces/Pids"} {
		_ = os.Remove(filepath.Join(pdir, name+".pid"))
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

func (c *Client) Export(name, outputPath string) error {
	scriptCandidates := []string{
		"/data/local/Droidspaces/bin/export_container.sh",
		"/data/adb/modules/droidspaces/bin/export_container.sh",
		"deploy/magisk/export_container.sh",
	}

	var scriptPath string
	for _, sc := range scriptCandidates {
		if _, err := os.Stat(sc); err == nil {
			scriptPath = sc
			break
		}
	}

	if scriptPath != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sh", scriptPath, name, outputPath)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("export script error (%v): %s", err, stderr.String())
		}
		return nil
	}

	cfg, err := config.ReadContainerConfig(name)
	if err != nil {
		return fmt.Errorf("failed to read container config: %w", err)
	}
	rootfs := cfg["rootfs_path"]
	if rootfs == "" {
		return fmt.Errorf("rootfs_path not found for %s", name)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "tar", "-czf", outputPath, "-C", rootfs, ".")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar fallback export failed: %w - %s", err, stderr.String())
	}
	return nil
}

func (c *Client) ListBackups(name string) ([]model.ContainerBackupInfo, error) {
	backupsDir := "/data/local/Droidspaces/Backups"
	_ = os.MkdirAll(backupsDir, 0755)

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return nil, err
	}

	prefix := name + "_"
	var list []model.ContainerBackupInfo

	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		fname := ent.Name()
		if (strings.HasPrefix(fname, prefix) || fname == name+".tar.gz") && (strings.HasSuffix(fname, ".tar.gz") || strings.HasSuffix(fname, ".tar")) {
			info, err := ent.Info()
			if err != nil {
				continue
			}
			list = append(list, model.ContainerBackupInfo{
				Filename:  fname,
				Path:      filepath.Join(backupsDir, fname),
				Size:      FormatBytes(info.Size()),
				SizeBytes: info.Size(),
				ModTime:   info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].ModTime > list[j].ModTime
	})
	return list, nil
}

func (c *Client) DeleteBackup(filename string) error {
	cleanName := filepath.Base(filename)
	if !strings.HasSuffix(cleanName, ".tar.gz") && !strings.HasSuffix(cleanName, ".tar") {
		return fmt.Errorf("invalid backup filename")
	}
	target := filepath.Join("/data/local/Droidspaces/Backups", cleanName)
	return os.Remove(target)
}
