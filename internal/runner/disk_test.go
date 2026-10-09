package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{-10, "0 B"},
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024 * 2, "2.0 GB"},
		{1024 * 1024 * 1024 * 1024 * 3, "3.0 TB"},
	}

	for _, tt := range tests {
		res := FormatBytes(tt.input)
		if res != tt.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", tt.input, res, tt.expected)
		}
	}
}

func TestGetContainerDiskSize(t *testing.T) {
	tmpDir := t.TempDir()
	fpath := filepath.Join(tmpDir, "test.img")
	if err := os.WriteFile(fpath, make([]byte, 2048), 0644); err != nil {
		t.Fatalf("failed to create dummy file: %v", err)
	}

	formatted, bytes := GetContainerDiskSize(fpath)
	if bytes != 2048 || formatted != "2.0 KB" {
		t.Errorf("expected 2048 bytes / 2.0 KB, got %d / %s", bytes, formatted)
	}

	// Cache hit
	f2, b2 := GetContainerDiskSize(fpath)
	if b2 != bytes || f2 != formatted {
		t.Errorf("expected cache hit to return same values")
	}

	// Empty path
	fEmpty, bEmpty := GetContainerDiskSize("")
	if bEmpty != 0 || fEmpty != "0 B" {
		t.Errorf("expected 0 B for empty path")
	}
}

func TestParsePortMappings(t *testing.T) {
	if res := ParsePortMappings(""); res != nil {
		t.Errorf("expected nil for empty port mappings")
	}

	res := ParsePortMappings("8080:80,443:443/udp,22:22")
	if len(res) != 3 {
		t.Fatalf("expected 3 mappings, got %d", len(res))
	}
	if res[0] != "8080:80" || res[1] != "443:443/udp" || res[2] != "22:22" {
		t.Errorf("unexpected parse result: %v", res)
	}
}

func TestParsePortMatrix(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DROIDSPACES_CONTAINERS_DIR", tmpDir)

	c1Dir := filepath.Join(tmpDir, "c1")
	_ = os.MkdirAll(c1Dir, 0755)
	_ = os.WriteFile(filepath.Join(c1Dir, "container.config"), []byte("port_forwards=8080:80/tcp,9090:90/udp,3000\n"), 0644)

	c2Dir := filepath.Join(tmpDir, "c2")
	_ = os.MkdirAll(c2Dir, 0755)
	_ = os.WriteFile(filepath.Join(c2Dir, "container.config"), []byte("port=8000:8000\n"), 0644)

	running := []model.ContainerSummary{
		{Name: "c1", IP: "172.28.0.2", Status: "running"},
	}
	stopped := []model.ContainerSummary{
		{Name: "c2", Status: "stopped"},
	}

	matrix := ParsePortMatrix(running, stopped)
	if len(matrix) != 4 {
		t.Fatalf("expected 4 port matrix entries, got %d: %+v", len(matrix), matrix)
	}
}
