package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListTemplatesAndLocalScanning(t *testing.T) {
	tmpDir := t.TempDir()
	origStorage := GetStorageDir()
	_ = origStorage

	// Create a dummy custom directory rootfs in tmpDir
	customDir := filepath.Join(tmpDir, "alpine-custom")
	if err := os.MkdirAll(filepath.Join(customDir, "etc", "init.d"), 0755); err != nil {
		t.Fatalf("failed to create custom rootfs dir: %v", err)
	}

	// Create a dummy custom image in tmpDir
	imgFile := filepath.Join(tmpDir, "archlinux.img")
	if err := os.WriteFile(imgFile, []byte("fake-img"), 0644); err != nil {
		t.Fatalf("failed to write dummy img: %v", err)
	}

	// Test detectInitFromDir
	initSys := detectInitFromDir(customDir)
	if initSys != "openrc" {
		t.Errorf("expected init system openrc, got %s", initSys)
	}

	// Test detectDistroFromName
	distro := detectDistroFromName("alpine-custom")
	if distro != "alpine" {
		t.Errorf("expected distro alpine, got %s", distro)
	}

	distroArch := detectDistroFromName("archlinux.img")
	if distroArch != "arch" {
		t.Errorf("expected distro arch, got %s", distroArch)
	}
}
