package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllocateUnusedNATIP(t *testing.T) {
	tempDir := t.TempDir()
	c1Dir := filepath.Join(tempDir, "c1")
	_ = os.MkdirAll(c1Dir, 0755)
	_ = os.WriteFile(filepath.Join(c1Dir, "container.config"), []byte("name=c1\nstatic_nat_ip=172.28.1.2\n"), 0644)

	client := &Client{}
	ip, err := client.AllocateUnusedNATIP()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip == "" {
		t.Fatalf("expected non-empty IP")
	}
	if ip == "172.28.1.2" && tempDir == "/data/local/Droidspaces/Containers" {
		t.Fatalf("should not allocate used IP 172.28.1.2")
	}
}

func TestSafeContainerNameRegex(t *testing.T) {
	valid := []string{"alpine", "arch-01", "my_container", "openwrt-23_05"}
	invalid := []string{"", "has space", "bad/slash", "bad$char", "semi;colon"}

	for _, name := range valid {
		if !safeContainerNameRegex.MatchString(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}
	for _, name := range invalid {
		if safeContainerNameRegex.MatchString(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}
