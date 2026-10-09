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
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024 * 2, "2.0 GB"},
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
}

func TestParsePortMappings(t *testing.T) {
	raw := "80:80/tcp, 2222:22/tcp 443:443"
	res := ParsePortMappings(raw)
	if len(res) != 3 {
		t.Fatalf("expected 3 mappings, got %d", len(res))
	}
	if res[0] != "80:80/tcp" || res[1] != "2222:22/tcp" || res[2] != "443:443" {
		t.Errorf("unexpected parse result: %v", res)
	}
}

func TestParsePortMatrix(t *testing.T) {
	running := []model.ContainerSummary{
		{Name: "c1", IP: "172.28.0.2", Status: "running"},
	}
	stopped := []model.ContainerSummary{
		{Name: "c2", IP: "172.28.0.3", Status: "stopped"},
	}

	matrix := ParsePortMatrix(running, stopped)
	// Even if config doesn't exist in tmp, function should not crash
	_ = matrix
}
