package marketplace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCatalogHasThreeUniquePresets(t *testing.T) {
	presets := DefaultCatalog()
	if len(presets) != 3 {
		t.Fatalf("expected 3 presets, got %d", len(presets))
	}

	seen := make(map[string]bool, len(presets))
	for _, preset := range presets {
		if preset.ID == "" || seen[preset.ID] {
			t.Fatalf("preset ID is empty or duplicated: %q", preset.ID)
		}
		seen[preset.ID] = true
		if len(preset.SetupSteps) == 0 {
			t.Errorf("preset %q has no setup steps", preset.ID)
		}
		for _, step := range preset.SetupSteps {
			if step.Name == "" || step.Command == "" {
				t.Errorf("preset %q has incomplete setup step: %#v", preset.ID, step)
			}
		}
	}
}

func TestPublicViewDoesNotExposeSetupCommands(t *testing.T) {
	data, err := json.Marshal(PublicView())
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(data)
	for _, forbidden := range []string{"apk add", "apt-get", "Command", "command"} {
		if contains(encoded, forbidden) {
			t.Fatalf("public catalog contains private setup command marker %q", forbidden)
		}
	}
}

func TestValidateName(t *testing.T) {
	for _, name := range []string{"nginx-01", "box_2", "Box3"} {
		if err := ValidateName(name); err != nil {
			t.Errorf("expected %q to be valid: %v", name, err)
		}
	}
	for _, name := range []string{"", "../bad", "bad/name", "has space"} {
		if err := ValidateName(name); err == nil {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}

func TestRequiresRootFSMatchesIDsAndContainerPaths(t *testing.T) {
	preset, ok := ByID("nginx-static")
	if !ok {
		t.Fatal("nginx-static preset missing")
	}
	for _, rootfs := range []string{"alpine-3.20", "/data/local/Droidspaces/rootfs/alpine-3.20"} {
		if !preset.RequiresRootFS(rootfs) {
			t.Errorf("expected rootfs %q to match", rootfs)
		}
	}
	for _, rootfs := range []string{"", "debian-12", "/tmp/custom-rootfs"} {
		if preset.RequiresRootFS(rootfs) {
			t.Errorf("unexpected rootfs match for %q", rootfs)
		}
	}
}

func TestRequiresRootFSReadsOSReleaseForClonedContainer(t *testing.T) {
	preset, _ := ByID("home-assistant")
	rootfs := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootfs, "etc"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootfs, "etc", "os-release"), []byte("ID=debian\nVERSION_ID=\"12\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !preset.RequiresRootFS(rootfs) {
		t.Fatal("expected Debian 12 os-release to match Home Assistant preset")
	}
	if preset.RequiresRootFS(filepath.Join(rootfs, "missing")) {
		t.Fatal("unexpected match for missing rootfs path")
	}
}

func contains(s, part string) bool {
	for i := 0; i+len(part) <= len(s); i++ {
		if s[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
