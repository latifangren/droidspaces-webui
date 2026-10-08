//go:build !windows

package hardware

import "syscall"

func getStorageStats(path string) (float64, float64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err == nil {
		totalBytes := stat.Blocks * uint64(stat.Bsize)
		freeBytes := stat.Bavail * uint64(stat.Bsize)
		return float64(totalBytes) / (1024 * 1024 * 1024), float64(freeBytes) / (1024 * 1024 * 1024)
	}
	return 0, 0
}
