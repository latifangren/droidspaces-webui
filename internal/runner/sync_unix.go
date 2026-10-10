//go:build !windows

package runner

import "syscall"

func syncFileSystem() {
	syscall.Sync()
}
