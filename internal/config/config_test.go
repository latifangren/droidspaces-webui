package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

func TestGetContainersDirs(t *testing.T) {
	origEnv := os.Getenv("DROIDSPACES_CONTAINERS_DIR")
	defer os.Setenv("DROIDSPACES_CONTAINERS_DIR", origEnv)

	os.Setenv("DROIDSPACES_CONTAINERS_DIR", "/custom/test/containers")
	dirs := GetContainersDirs()
	if len(dirs) == 0 || dirs[0] != "/custom/test/containers" {
		t.Fatalf("expected custom dir first, got %v", dirs)
	}

	os.Unsetenv("DROIDSPACES_CONTAINERS_DIR")
	dirs = GetContainersDirs()
	if len(dirs) == 0 {
		t.Fatalf("expected default dirs, got empty")
	}
}

func TestReadAndWriteContainerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origEnv := os.Getenv("DROIDSPACES_CONTAINERS_DIR")
	defer os.Setenv("DROIDSPACES_CONTAINERS_DIR", origEnv)
	os.Setenv("DROIDSPACES_CONTAINERS_DIR", tmpDir)

	// Test invalid container name
	if _, err := ReadContainerConfig("../malicious"); err == nil {
		t.Errorf("expected error on invalid container name")
	}
	if err := WriteContainerConfigKeys("../malicious", map[string]string{"foo": "bar"}); err == nil {
		t.Errorf("expected error on invalid container name for write")
	}

	// Test container not found
	if _, err := ReadContainerConfig("nonexistent"); err == nil {
		t.Errorf("expected error for nonexistent container")
	}

	// Create a container directory and config
	containerName := "test-box"
	containerDir := filepath.Join(tmpDir, containerName)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		t.Fatalf("failed to create container dir: %v", err)
	}

	cfgContent := `# Sample config
name=test-box
run_at_boot=1
run_at_boot_priority="10"
isolated_network = true
# Empty line next

other_key='quoted_value'
invalid_line_no_equals
`
	cfgPath := filepath.Join(containerDir, "container.config")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Read container config
	cfg, err := ReadContainerConfig(containerName)
	if err != nil {
		t.Fatalf("ReadContainerConfig failed: %v", err)
	}

	if cfg["name"] != "test-box" {
		t.Errorf("expected name 'test-box', got %s", cfg["name"])
	}
	if cfg["run_at_boot"] != "1" {
		t.Errorf("expected run_at_boot '1', got %s", cfg["run_at_boot"])
	}
	if cfg["run_at_boot_priority"] != "\"10\"" {
		t.Errorf("expected run_at_boot_priority '\"10\"', got %s", cfg["run_at_boot_priority"])
	}
	if cfg["isolated_network"] != "true" {
		t.Errorf("expected isolated_network 'true', got %s", cfg["isolated_network"])
	}
	if cfg["other_key"] != "'quoted_value'" {
		t.Errorf("expected other_key ''quoted_value'', got %s", cfg["other_key"])
	}

	// Read again to hit cache
	cfgCached, err := ReadContainerConfig(containerName)
	if err != nil || cfgCached["name"] != "test-box" {
		t.Fatalf("ReadContainerConfig from cache failed: %v", err)
	}

	// Invalidate cache
	InvalidateCache(containerName)

	// Write new keys (updating existing and adding new)
	updates := map[string]string{
		"run_at_boot": "0",
		"brand_new":   "custom_value",
	}
	if err := WriteContainerConfigKeys(containerName, updates); err != nil {
		t.Fatalf("WriteContainerConfigKeys failed: %v", err)
	}

	// Read back and verify
	cfgUpdated, err := ReadContainerConfig(containerName)
	if err != nil {
		t.Fatalf("ReadContainerConfig after write failed: %v", err)
	}
	if cfgUpdated["run_at_boot"] != "0" {
		t.Errorf("expected run_at_boot '0', got %s", cfgUpdated["run_at_boot"])
	}
	if cfgUpdated["brand_new"] != "custom_value" {
		t.Errorf("expected brand_new 'custom_value', got %s", cfgUpdated["brand_new"])
	}

	// Test write to container with no existing container.config
	newBox := "new-box"
	if err := WriteContainerConfigKeys(newBox, map[string]string{"foo": "bar"}); err != nil {
		t.Fatalf("WriteContainerConfigKeys for new box failed: %v", err)
	}
	newCfg, err := ReadContainerConfig(newBox)
	if err != nil || newCfg["foo"] != "bar" {
		t.Fatalf("failed to read created config: %v", err)
	}
}

func TestBootPriorities(t *testing.T) {
	tmpDir := t.TempDir()
	origEnv := os.Getenv("DROIDSPACES_CONTAINERS_DIR")
	defer os.Setenv("DROIDSPACES_CONTAINERS_DIR", origEnv)
	os.Setenv("DROIDSPACES_CONTAINERS_DIR", tmpDir)

	// Create 3 containers with different priorities
	c1Dir := filepath.Join(tmpDir, "box-alpha")
	c2Dir := filepath.Join(tmpDir, "box-beta")
	c3Dir := filepath.Join(tmpDir, "box-gamma")
	_ = os.MkdirAll(c1Dir, 0755)
	_ = os.MkdirAll(c2Dir, 0755)
	_ = os.MkdirAll(c3Dir, 0755)

	_ = os.WriteFile(filepath.Join(c1Dir, "container.config"), []byte("run_at_boot=1\nrun_at_boot_priority=20\n"), 0644)
	_ = os.WriteFile(filepath.Join(c2Dir, "container.config"), []byte("run_at_boot=1\nrun_at_boot_priority=10\n"), 0644)
	_ = os.WriteFile(filepath.Join(c3Dir, "container.config"), []byte("run_at_boot=0\n"), 0644)

	running := map[string]bool{
		"box-beta": true,
	}

	items, err := GetBootPriorities(running)
	if err != nil {
		t.Fatalf("GetBootPriorities failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Verify sorting: priority 10 (beta) should be before priority 20 (alpha), and unprioritized (gamma) last
	if items[0].Name != "box-beta" {
		t.Errorf("expected box-beta first due to priority 10, got %s", items[0].Name)
	}
	if items[0].Status != "running" {
		t.Errorf("expected box-beta status to be running")
	}
	if items[1].Name != "box-alpha" {
		t.Errorf("expected box-alpha second due to priority 20, got %s", items[1].Name)
	}
	if items[2].Name != "box-gamma" {
		t.Errorf("expected box-gamma third, got %s", items[2].Name)
	}

	// Test SetBootPriorities
	newPriorities := []model.BootPriorityItem{
		{Name: "box-gamma", RunAtBoot: true, RunAtBootPriority: 5},
		{Name: "box-beta", RunAtBoot: false, RunAtBootPriority: 0},
	}
	if err := SetBootPriorities(newPriorities); err != nil {
		t.Fatalf("SetBootPriorities failed: %v", err)
	}

	updatedItems, err := GetBootPriorities(running)
	if err != nil {
		t.Fatalf("GetBootPriorities after update failed: %v", err)
	}
	if updatedItems[0].Name != "box-gamma" {
		t.Errorf("expected box-gamma first with priority 5, got %s", updatedItems[0].Name)
	}
}
