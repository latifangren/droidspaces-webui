package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	cmd := exec.Command(c.binPath, args...)
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

func (c *Client) Show() (*model.ShowResult, error) {
	out, err := c.run("show", "--format")
	if err != nil {
		return nil, err
	}

	var res model.ShowResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
		return nil, fmt.Errorf("failed to parse show output: %w", err)
	}
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

func (c *Client) Start(req model.StartRequest) error {
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

func (c *Client) Check() (string, error) {
	return c.run("check")
}

func (c *Client) Scan() (string, error) {
	return c.run("scan")
}

func (c *Client) Exec(container, command, user string) (string, error) {
	args := []string{"run"}
	if user != "" {
		args = append(args, "-u", user)
	}
	args = append(args, "--name="+container, "--", "/bin/sh", "-c", command)

	cmd := exec.Command(c.binPath, args...)
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

func (c *Client) ExecHost(command string) (string, error) {
	cmd := exec.Command("/system/bin/sh", "-c", command)
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
