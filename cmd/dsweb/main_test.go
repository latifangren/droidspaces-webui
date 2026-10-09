package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePort(t *testing.T) {
	// 1. Explicit port flag != 84
	if p := resolvePort(8080); p != 8080 {
		t.Errorf("expected 8080, got %d", p)
	}

	// 2. DSWEB_PORT environment variable
	t.Setenv("DSWEB_PORT", "9090")
	if p := resolvePort(84); p != 9090 {
		t.Errorf("expected 9090 from env, got %d", p)
	}

	// 3. Port file candidate
	t.Setenv("DSWEB_PORT", "")
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "port.txt")
	_ = os.WriteFile(portFile, []byte("7070\n"), 0644)

	if p := resolvePort(84, portFile); p != 7070 {
		t.Errorf("expected 7070 from file candidate, got %d", p)
	}

	// 4. Fallback default
	if p := resolvePort(84, filepath.Join(tmpDir, "missing.txt")); p != 84 {
		t.Errorf("expected 84 default fallback, got %d", p)
	}
}

func TestBuildServer(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DROIDSPACES_CONTAINERS_DIR", filepath.Join(tmpDir, "Containers"))
	t.Setenv("DROIDSPACES_ROOTFS_DIR", filepath.Join(tmpDir, "rootfs"))
	t.Setenv("DROIDSPACES_LOGS_DIR", filepath.Join(tmpDir, "Logs"))
	t.Setenv("DROIDSPACES_AUTH_FILE", filepath.Join(tmpDir, ".webui_auth"))

	srv := buildServer(8088)
	if srv == nil || srv.Addr != ":8088" || srv.Handler == nil {
		t.Errorf("unexpected buildServer result: %+v", srv)
	}
}

func TestRun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DROIDSPACES_CONTAINERS_DIR", filepath.Join(tmpDir, "Containers"))
	t.Setenv("DROIDSPACES_ROOTFS_DIR", filepath.Join(tmpDir, "rootfs"))
	t.Setenv("DROIDSPACES_LOGS_DIR", filepath.Join(tmpDir, "Logs"))
	t.Setenv("DROIDSPACES_AUTH_FILE", filepath.Join(tmpDir, ".webui_auth"))

	origListen := listenAndServe
	listenAndServe = func(srv *http.Server) error {
		return nil
	}
	defer func() {
		listenAndServe = origListen
	}()

	// Valid args
	if err := run([]string{"-port", "8484"}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// Invalid flag
	if err := run([]string{"-invalidflag"}); err == nil {
		t.Errorf("expected error for invalid flag")
	}
}
