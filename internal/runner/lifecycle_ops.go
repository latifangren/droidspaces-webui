package runner

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/latifangren/droidspaces-webui/internal/config"
	"github.com/latifangren/droidspaces-webui/internal/model"
)

var (
	safeContainerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	backupsDir             = "/data/local/Droidspaces/Backups"
)
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// AllocateUnusedNATIP scans all existing container configurations and returns
// a guaranteed conflict-free IP address in the 172.28.0.0/16 subnet.
func (c *Client) AllocateUnusedNATIP() (string, error) {
	usedIPs := make(map[string]bool)
	dirs := config.GetContainersDirs()
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			cfg, err := config.ReadContainerConfig(e.Name())
			if err == nil && cfg["static_nat_ip"] != "" {
				usedIPs[cfg["static_nat_ip"]] = true
			}
		}
	}

	// Try allocating from 172.28.1.2 up to 172.28.254.254
	for octet2 := 1; octet2 <= 254; octet2++ {
		for octet3 := 2; octet3 <= 254; octet3++ {
			ip := fmt.Sprintf("172.28.%d.%d", octet2, octet3)
			if !usedIPs[ip] {
				return ip, nil
			}
		}
	}
	return "", fmt.Errorf("no available IP addresses in 172.28.0.0/16 subnet")
}

// Clone creates an exact, independent duplicate of a container with conflict prevention.
// It assigns a new unique NAT IP, unique UUID, removes conflicting host ports,
// and copies the rootfs with sparse preservation.
func (c *Client) Clone(sourceName, targetName string, autoStart bool) error {
	if !safeContainerNameRegex.MatchString(sourceName) {
		return fmt.Errorf("invalid source container name: %s", sourceName)
	}
	if !safeContainerNameRegex.MatchString(targetName) {
		return fmt.Errorf("invalid target container name: %s (must be alphanumeric, _ or -)", targetName)
	}
	if sourceName == targetName {
		return fmt.Errorf("source and target container names cannot be identical")
	}

	// 1. Verify source exists
	srcCfg, err := config.ReadContainerConfig(sourceName)
	if err != nil {
		return fmt.Errorf("source container %s config not found: %w", sourceName, err)
	}

	// 2. Verify target does NOT exist
	dirs := config.GetContainersDirs()
	if len(dirs) == 0 {
		return fmt.Errorf("no containers directory configured")
	}
	targetDir := filepath.Join(dirs[0], targetName)
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("target container %s already exists", targetName)
	}

	// Flush filesystem buffer
	_ = exec.Command("sync").Run()

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target container directory: %w", err)
	}

	// 3. Duplicate rootfs
	srcRootFS := srcCfg["rootfs_path"]
	if srcRootFS == "" {
		_ = os.RemoveAll(targetDir)
		return fmt.Errorf("source container has no rootfs_path configured")
	}

	var targetRootFS string
	if strings.HasSuffix(srcRootFS, ".img") {
		// Sparse image clone
		targetImg := filepath.Join(targetDir, "rootfs.img")
		cmd := exec.Command("cp", "--sparse=always", "-a", srcRootFS, targetImg)
		if out, err := cmd.CombinedOutput(); err != nil {
			if errCopy := copyFile(srcRootFS, targetImg); errCopy != nil {
				_ = os.RemoveAll(targetDir)
				return fmt.Errorf("failed to clone sparse rootfs image (%v): %s", err, string(out))
			}
		}
		targetRootFS = targetImg
	} else if strings.HasPrefix(srcRootFS, "/data/local/Droidspaces/rootfs/") {
		// Points to template rootfs store. Clone into a dedicated container rootfs folder for isolation.
		targetRootFSDir := filepath.Join(targetDir, "rootfs")
		if err := os.MkdirAll(targetRootFSDir, 0755); err != nil {
			_ = os.RemoveAll(targetDir)
			return err
		}
		cmd := exec.Command("cp", "-a", srcRootFS+"/.", targetRootFSDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			cmd2 := exec.Command("cp", "-r", srcRootFS+"/.", targetRootFSDir)
			if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
				if errCopy := copyDir(srcRootFS, targetRootFSDir); errCopy != nil {
					_ = os.RemoveAll(targetDir)
					return fmt.Errorf("failed to clone rootfs directory (%v): %s / %s", err2, string(out), string(out2))
				}
			}
		}
		targetRootFS = targetRootFSDir
	} else {
		// Dedicated rootfs directory
		targetRootFSDir := filepath.Join(targetDir, "rootfs")
		if err := os.MkdirAll(targetRootFSDir, 0755); err != nil {
			_ = os.RemoveAll(targetDir)
			return err
		}
		cmd := exec.Command("cp", "-a", srcRootFS+"/.", targetRootFSDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			if errCopy := copyDir(srcRootFS, targetRootFSDir); errCopy != nil {
				_ = os.RemoveAll(targetDir)
				return fmt.Errorf("failed to clone rootfs directory (%v): %s", err, string(out))
			}
		}
		targetRootFS = targetRootFSDir
	}

	// 4. Allocate unique IP
	newIP, err := c.AllocateUnusedNATIP()
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return fmt.Errorf("failed to allocate free NAT IP: %w", err)
	}

	// 5. Construct clean container.config
	newConfig := make(map[string]string)
	for k, v := range srcCfg {
		newConfig[k] = v
	}
	newConfig["name"] = targetName
	newConfig["hostname"] = targetName
	newConfig["rootfs_path"] = targetRootFS
	newConfig["static_nat_ip"] = newIP
	newConfig["uuid"] = generateUUID()
	newConfig["run_at_boot"] = "0"
	newConfig["run_at_boot_priority"] = "0"
	delete(newConfig, "port")
	delete(newConfig, "ports")

	var cfgContent strings.Builder
	cfgContent.WriteString("# Droidspaces Container Configuration (Cloned)\n")
	for k, v := range newConfig {
		cfgContent.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	cfgFile := filepath.Join(targetDir, "container.config")
	if err := os.WriteFile(cfgFile, []byte(cfgContent.String()), 0644); err != nil {
		_ = os.RemoveAll(targetDir)
		return fmt.Errorf("failed to write cloned container config: %w", err)
	}

	config.InvalidateCache(targetName)

	if autoStart {
		return c.Start(model.StartRequest{Name: targetName})
	}
	return nil
}

// Restore extracts a .tar.gz backup archive into a container.
// If overwrite is false, it restores as a brand new isolated container with an auto-allocated IP.
// If overwrite is true, it safely stops the existing container and replaces its rootfs.
func (c *Client) Restore(req model.RestoreRequest) error {
	if !safeContainerNameRegex.MatchString(req.TargetName) {
		return fmt.Errorf("invalid target container name: %s", req.TargetName)
	}
	cleanFilename := filepath.Base(req.Filename)
	if !strings.HasSuffix(cleanFilename, ".tar.gz") && !strings.HasSuffix(cleanFilename, ".tar") {
		return fmt.Errorf("invalid archive format: must be .tar.gz or .tar")
	}

	backupPath := filepath.Join(backupsDir, cleanFilename)
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup archive not found at %s", backupPath)
	}

	dirs := config.GetContainersDirs()
	if len(dirs) == 0 {
		return fmt.Errorf("no containers directory configured")
	}
	targetDir := filepath.Join(dirs[0], req.TargetName)

	if req.Overwrite {
		// Stop if running
		_ = c.Stop(req.TargetName)
		time.Sleep(1 * time.Second)

		existingCfg, err := config.ReadContainerConfig(req.TargetName)
		var rootfsDir string
		if err == nil && existingCfg["rootfs_path"] != "" && !strings.HasSuffix(existingCfg["rootfs_path"], ".img") {
			rootfsDir = existingCfg["rootfs_path"]
		} else {
			rootfsDir = filepath.Join(targetDir, "rootfs")
		}

		_ = os.MkdirAll(rootfsDir, 0755)
		extractCmd := exec.Command("tar", "-xzf", backupPath, "-C", rootfsDir)
		if out, err := extractCmd.CombinedOutput(); err != nil {
			bbCmd := exec.Command("/data/local/Droidspaces/bin/busybox", "tar", "-xzf", backupPath, "-C", rootfsDir)
			if bbOut, bbErr := bbCmd.CombinedOutput(); bbErr != nil {
				return fmt.Errorf("restore extraction failed: %v (%s / %s)", bbErr, string(out), string(bbOut))
			}
		}

		if req.AutoStart {
			return c.Start(model.StartRequest{Name: req.TargetName})
		}
		return nil
	}

	// Restore as NEW container
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("container %s already exists. Choose a different name or enable Overwrite", req.TargetName)
	}

	targetRootFS := filepath.Join(targetDir, "rootfs")
	if err := os.MkdirAll(targetRootFS, 0755); err != nil {
		return fmt.Errorf("failed to create target rootfs directory: %w", err)
	}

	extractCmd := exec.Command("tar", "-xzf", backupPath, "-C", targetRootFS)
	if out, err := extractCmd.CombinedOutput(); err != nil {
		bbCmd := exec.Command("/data/local/Droidspaces/bin/busybox", "tar", "-xzf", backupPath, "-C", targetRootFS)
		if bbOut, bbErr := bbCmd.CombinedOutput(); bbErr != nil {
			_ = os.RemoveAll(targetDir)
			return fmt.Errorf("restore extraction failed: %v (%s / %s)", bbErr, string(out), string(bbOut))
		}
	}

	// Allocate a safe unique IP
	newIP, err := c.AllocateUnusedNATIP()
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return fmt.Errorf("failed to allocate free NAT IP: %w", err)
	}

	// Generate clean container.config
	var cfgContent strings.Builder
	cfgContent.WriteString("# Droidspaces Container Configuration (Restored)\n")
	cfgContent.WriteString(fmt.Sprintf("name=%s\n", req.TargetName))
	cfgContent.WriteString(fmt.Sprintf("hostname=%s\n", req.TargetName))
	cfgContent.WriteString(fmt.Sprintf("rootfs_path=%s\n", targetRootFS))
	cfgContent.WriteString("net_mode=nat\n")
	cfgContent.WriteString(fmt.Sprintf("static_nat_ip=%s\n", newIP))
	cfgContent.WriteString(fmt.Sprintf("uuid=%s\n", generateUUID()))
	cfgContent.WriteString("disable_ipv6=0\n")
	cfgContent.WriteString("run_at_boot=0\n")
	cfgContent.WriteString("run_at_boot_priority=0\n")

	// If arch linux, inject custom_init=/bin/bash
	if _, err := os.Stat(filepath.Join(targetRootFS, "etc/arch-release")); err == nil {
		cfgContent.WriteString("custom_init=/bin/bash\n")
	}

	cfgFile := filepath.Join(targetDir, "container.config")
	if err := os.WriteFile(cfgFile, []byte(cfgContent.String()), 0644); err != nil {
		_ = os.RemoveAll(targetDir)
		return fmt.Errorf("failed to write container config: %w", err)
	}

	config.InvalidateCache(req.TargetName)

	if req.AutoStart {
		return c.Start(model.StartRequest{Name: req.TargetName})
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}
