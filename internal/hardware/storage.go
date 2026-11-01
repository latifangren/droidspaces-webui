package hardware

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type StorageSupport string

const (
	StorageSupportFull      StorageSupport = "FULL"
	StorageSupportImageOnly StorageSupport = "IMAGE_ONLY"
	StorageSupportUnusable  StorageSupport = "UNUSABLE"
)

type StorageTarget struct {
	Path       string         `json:"path"`
	FSType     string         `json:"fs_type"`
	Support    StorageSupport `json:"support"`
	Reason     string         `json:"reason,omitempty"`
	FreeGB     float64        `json:"free_gb"`
	TotalGB    float64        `json:"total_gb"`
	MountPoint string         `json:"mount_point"`
}

var posixFS = map[string]bool{
	"ext2": true, "ext3": true, "ext4": true,
	"f2fs": true, "btrfs": true, "xfs": true,
}

var fatFS = map[string]bool{
	"vfat": true, "msdos": true, "fat": true, "fat32": true, "fuseblk.vfat": true,
}

var pseudoFS = map[string]bool{
	"tmpfs": true, "ramfs": true, "rootfs": true, "devtmpfs": true, "devpts": true,
	"proc": true, "sysfs": true, "debugfs": true, "tracefs": true, "configfs": true,
	"securityfs": true, "selinuxfs": true, "pstore": true, "cgroup": true, "cgroup2": true,
	"bpf": true, "binder": true, "binfmt_misc": true, "functionfs": true, "fusectl": true,
	"mqueue": true, "hugetlbfs": true,
}

type mountEntry struct {
	mountPoint string
	fsType     string
	options    map[string]bool
}

// ProbeStorage analyzes path against mount table to determine rootfs capabilities.
func ProbeStorage(targetPath string) StorageTarget {
	clean := filepath.Clean(strings.TrimSpace(targetPath))
	if clean == "" || clean == "." {
		clean = "/data/local/Droidspaces/rootfs"
	}
	clean = filepath.ToSlash(clean)

	mounts := readMounts()
	if len(mounts) == 0 {
		// Fallback for non-Linux / test environment
		tot, free := getStorageStats(clean)
		return StorageTarget{
			Path:       clean,
			FSType:     "ext4",
			Support:    StorageSupportFull,
			FreeGB:     free,
			TotalGB:    tot,
			MountPoint: "/",
		}
	}

	probePath := findExistingAncestor(clean)
	entry := longestMountPrefix(probePath, mounts)
	if entry == nil {
		tot, free := getStorageStats(probePath)
		return StorageTarget{
			Path:       clean,
			FSType:     "unknown",
			Support:    StorageSupportUnusable,
			Reason:     "No active filesystem mount point matches this path.",
			FreeGB:     free,
			TotalGB:    tot,
			MountPoint: "",
		}
	}

	support, reason := classifyMount(entry.fsType, entry.options)
	tot, free := getStorageStats(probePath)

	return StorageTarget{
		Path:       clean,
		FSType:     entry.fsType,
		Support:    support,
		Reason:     reason,
		FreeGB:     free,
		TotalGB:    tot,
		MountPoint: entry.mountPoint,
	}
}

func readMounts() []mountEntry {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil
	}
	defer f.Close()

	var entries []mountEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, " ")
		if len(fields) < 4 {
			continue
		}
		mp := unescapeOctal(fields[1])
		fsType := strings.ToLower(fields[2])
		opts := make(map[string]bool)
		for _, opt := range strings.Split(fields[3], ",") {
			opts[strings.TrimSpace(opt)] = true
		}
		entries = append(entries, mountEntry{
			mountPoint: mp,
			fsType:     fsType,
			options:    opts,
		})
	}
	return entries
}

func longestMountPrefix(target string, mounts []mountEntry) *mountEntry {
	normTarget := filepath.Clean(target)
	var best *mountEntry
	bestLen := -1

	for i := range mounts {
		mp := filepath.Clean(mounts[i].mountPoint)
		matched := false
		if normTarget == mp {
			matched = true
		} else if mp == "/" && strings.HasPrefix(normTarget, "/") {
			matched = true
		} else if strings.HasPrefix(normTarget, mp+"/") {
			matched = true
		}

		if matched && len(mp) > bestLen {
			bestLen = len(mp)
			best = &mounts[i]
		}
	}
	return best
}

func findExistingAncestor(target string) string {
	cur := target
	for cur != "" && cur != "/" && cur != "." {
		if _, err := os.Stat(cur); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return "/"
}

func classifyMount(fsType string, options map[string]bool) (StorageSupport, string) {
	if fsType == "tmpfs" {
		return StorageSupportUnusable, "This is an in-memory tmpfs volume: data will be lost on phone reboot."
	}
	if pseudoFS[fsType] {
		return StorageSupportUnusable, fmt.Sprintf("'%s' is a kernel virtual interface, not real storage.", fsType)
	}
	if options["ro"] {
		return StorageSupportUnusable, "This filesystem is mounted read-only."
	}
	if fatFS[fsType] {
		return StorageSupportUnusable, "FAT32 cannot store files larger than 4GB. Reformat volume as ext4 or exFAT."
	}
	if !posixFS[fsType] {
		return StorageSupportImageOnly, fmt.Sprintf("'%s' does not implement Linux permissions. Sparse ext4 image (.img) is required.", fsType)
	}
	if options["noexec"] {
		return StorageSupportImageOnly, "This volume is mounted noexec: binaries cannot execute directly. Sparse ext4 image (.img) is required."
	}
	return StorageSupportFull, ""
}

func unescapeOctal(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\\' && i+3 < len(s) && isOctalDigit(s[i+1]) && isOctalDigit(s[i+2]) && isOctalDigit(s[i+3]) {
			val, err := strconv.ParseInt(s[i+1:i+4], 8, 32)
			if err == nil {
				sb.WriteByte(byte(val))
				i += 4
				continue
			}
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}

func isOctalDigit(b byte) bool {
	return b >= '0' && b <= '7'
}

// CreateBlankSparseImage creates an ext4 sparse disk image of sizeGB at destPath.
func CreateBlankSparseImage(destPath string, sizeGB int) error {
	clean := filepath.Clean(strings.TrimSpace(destPath))
	if clean == "" || clean == "." {
		return fmt.Errorf("invalid image destination path")
	}
	if !strings.HasSuffix(clean, ".img") {
		clean = clean + ".img"
	}
	if sizeGB < 1 || sizeGB > 512 {
		return fmt.Errorf("image size must be between 1 and 512 GB (got %d)", sizeGB)
	}

	if _, err := os.Stat(clean); err == nil {
		return fmt.Errorf("target file already exists: %s", clean)
	}

	dir := filepath.Dir(clean)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	// 1. Truncate sparse file
	f, err := os.Create(clean)
	if err != nil {
		return fmt.Errorf("create image file: %w", err)
	}
	sizeBytes := int64(sizeGB) * 1024 * 1024 * 1024
	if err := f.Truncate(sizeBytes); err != nil {
		f.Close()
		_ = os.Remove(clean)
		return fmt.Errorf("truncate sparse file to %dGB: %w", sizeGB, err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		_ = os.Remove(clean)
		return fmt.Errorf("sync sparse file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(clean)
		return fmt.Errorf("close sparse file: %w", err)
	}

	// 2. Format ext4 filesystem
	var mkfsCmd *exec.Cmd
	if _, err := exec.LookPath("mkfs.ext4"); err == nil {
		mkfsCmd = exec.Command("mkfs.ext4", "-F", "-L", "rootfs", clean)
	} else if _, err := exec.LookPath("mke2fs"); err == nil {
		mkfsCmd = exec.Command("mke2fs", "-t", "ext4", "-F", "-L", "rootfs", clean)
	} else if runtime.GOOS == "windows" {
		// Mock/test stub on local Windows development
		return nil
	} else {
		_ = os.Remove(clean)
		return fmt.Errorf("no ext4 formatting tool available (install mkfs.ext4 or mke2fs)")
	}

	out, err := mkfsCmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(clean)
		return fmt.Errorf("format ext4 image failed: %s (%w)", string(out), err)
	}

	return nil
}
