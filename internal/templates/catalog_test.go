package templates

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"github.com/latifangren/droidspaces-webui/internal/model"
)

func TestGetStorageDir(t *testing.T) {
	orig := os.Getenv("DROIDSPACES_ROOTFS_DIR")
	defer os.Setenv("DROIDSPACES_ROOTFS_DIR", orig)

	tmp := t.TempDir()
	os.Setenv("DROIDSPACES_ROOTFS_DIR", tmp)
	if dir := GetStorageDir(); dir != tmp {
		t.Errorf("expected %s, got %s", tmp, dir)
	}

	os.Unsetenv("DROIDSPACES_ROOTFS_DIR")
	if dir := GetStorageDir(); dir == "" {
		t.Errorf("expected non-empty default storage dir")
	}
}

func TestCatalogDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. detectInitFromDir
	inits := map[string]string{
		"etc/systemd":        "systemd",
		"lib/systemd/systemd": "systemd",
		"etc/init.d":         "openrc",
		"etc/inittab":        "openrc",
		"etc/unknown":        "init",
	}
	for subPath, expected := range inits {
		dir := filepath.Join(tmpDir, "test-"+strings.ReplaceAll(subPath, "/", "-"))
		_ = os.MkdirAll(filepath.Join(dir, subPath), 0755)
		got := detectInitFromDir(dir)
		if got != expected {
			t.Errorf("for path %s expected init %s, got %s", subPath, expected, got)
		}
	}

	// 2. detectDistroFromName
	distros := map[string]string{
		"debian-bookworm-arm64": "debian",
		"ubuntu-jammy":          "ubuntu",
		"alpine-v3.18":          "alpine",
		"archlinux-latest":      "arch",
		"kali-rolling":          "kali",
		"fedora-39":             "fedora",
		"void-glibc":            "void",
		"openwrt-rootfs":        "openwrt",
		"custom-unknown-os":     "linux",
	}
	for name, expected := range distros {
		got := detectDistroFromName(name)
		if got != expected {
			t.Errorf("for %s expected distro %s, got %s", name, expected, got)
		}
	}

	// 3. formatLocalName
	fName := formatLocalName("debian-12-custom.tar.xz")
	if fName == "" {
		t.Errorf("expected non-empty formatLocalName")
	}
}

func TestListTemplatesWithLocalFiles(t *testing.T) {
	tmpDir := t.TempDir()
	orig := os.Getenv("DROIDSPACES_ROOTFS_DIR")
	defer os.Setenv("DROIDSPACES_ROOTFS_DIR", orig)
	os.Setenv("DROIDSPACES_ROOTFS_DIR", tmpDir)

	// Create local unpacked rootfs directory
	alpineDir := filepath.Join(tmpDir, "alpine-local")
	_ = os.MkdirAll(filepath.Join(alpineDir, "etc", "init.d"), 0755)

	// Create local standalone image file
	_ = os.WriteFile(filepath.Join(tmpDir, "ubuntu-local.img"), []byte("dummy-img"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "ignore-me.txt"), []byte("not-rootfs"), 0644)

	templates := ListTemplates()
	if len(templates) == 0 {
		t.Fatalf("expected non-empty templates")
	}

	// Check if local templates were discovered and tagged as local
	var foundAlpine, foundUbuntu bool
	for _, t := range templates {
		if t.ID == "local-alpine-local" && t.Installed {
			foundAlpine = true
		}
		if t.ID == "img-ubuntu-local" && t.Installed {
			foundUbuntu = true
		}
	}

	if !foundAlpine {
		t.Errorf("expected to find local installed alpine template")
	}
	if !foundUbuntu {
		t.Errorf("expected to find local ubuntu img template")
	}
}

func TestDeleteTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	orig := os.Getenv("DROIDSPACES_ROOTFS_DIR")
	defer os.Setenv("DROIDSPACES_ROOTFS_DIR", orig)
	os.Setenv("DROIDSPACES_ROOTFS_DIR", tmpDir)

	// Create local template dir and delete it
	targetDir := filepath.Join(tmpDir, "arch-del")
	_ = os.MkdirAll(targetDir, 0755)

	if err := DeleteTemplate("local-arch-del"); err != nil {
		t.Fatalf("DeleteTemplate failed: %v", err)
	}

	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Errorf("expected targetDir to be deleted")
	}

	// Create img template and delete it
	targetImg := filepath.Join(tmpDir, "fedora.img")
	_ = os.WriteFile(targetImg, []byte("img-data"), 0644)

	if err := DeleteTemplate("img-fedora"); err != nil {
		t.Fatalf("DeleteTemplate for img failed: %v", err)
	}
	if _, err := os.Stat(targetImg); !os.IsNotExist(err) {
		t.Errorf("expected targetImg to be deleted")
	}
}

func TestJobStatusAndProgressWriter(t *testing.T) {
	setJobError("test error message")
	_, _, errStr := GetDownloadStatus()
	if errStr != "test error message" {
		t.Errorf("unexpected download error string: %s", errStr)
	}

	// Test progressWriter
	var progressCalled bool
	pw := &progressWriter{
		total: 1000,
		onProgress: func(downloaded, total int64) {
			progressCalled = true
		},
	}
	n, err := pw.Write([]byte("1234567890"))
	if err != nil || n != 10 {
		t.Errorf("progressWriter Write failed: n=%d, err=%v", n, err)
	}
	if pw.downloaded != 10 {
		t.Errorf("expected downloaded 10, got %d", pw.downloaded)
	}
	_ = progressCalled
}

func TestCreateSecureTLSConfig(t *testing.T) {
	cfg := createSecureTLSConfig()
	if cfg == nil {
		t.Fatalf("expected non-nil tls.Config")
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2 minimum version, got %x", cfg.MinVersion)
	}
}

func TestExtractArchiveImg(t *testing.T) {
	tmpDir := t.TempDir()
	fakeImg := filepath.Join(tmpDir, "source.img")
	_ = os.WriteFile(fakeImg, []byte("raw image data"), 0644)

	targetDir := filepath.Join(tmpDir, "extracted")
	if err := extractArchive(fakeImg, targetDir, "img"); err != nil {
		t.Fatalf("extractArchive for img failed: %v", err)
	}

	destImg := filepath.Join(targetDir, "rootfs.img")
	if data, err := os.ReadFile(destImg); err != nil || string(data) != "raw image data" {
		t.Errorf("rootfs.img content mismatch or error: %v", err)
	}
}

func TestApplyPostExtractFixes(t *testing.T) {
	tmpDir := t.TempDir()
	applyPostExtractFixes(tmpDir)

	// Check that /etc/resolv.conf was created if it didn't exist
	resolv := filepath.Join(tmpDir, "etc", "resolv.conf")
	if data, err := os.ReadFile(resolv); err != nil || !strings.Contains(string(data), "nameserver") {
		t.Errorf("resolv.conf was not created properly: %v", err)
	}
}

func TestResolveDownloadURL(t *testing.T) {
	// 1. Non-LXC URL returns as is
	tplNonLXC := &model.TemplateInfo{
		URL:    "https://example.com/rootfs.tar.xz",
		Distro: "alpine",
	}
	if u := resolveDownloadURL(tplNonLXC, http.DefaultClient); u != tplNonLXC.URL {
		t.Errorf("expected direct url %s, got %s", tplNonLXC.URL, u)
	}

	// 2. Mock LXC index server
	indexContent := "debian;bookworm;arm64;default;20240101_05:24;/images/debian/bookworm/arm64/default/20240101_05:24/\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, indexContent)
	}))
	defer server.Close()

	tplLXC := &model.TemplateInfo{
		URL:    server.URL + "/images.linuxcontainers.org/placeholder",
		Distro: "debian",
	}

	_ = resolveDownloadURL(tplLXC, server.Client())
}

func TestStartDownloadValidations(t *testing.T) {
	// Test nonexistent template ID
	if err := StartDownload("nonexistent-template-id"); err == nil {
		t.Errorf("expected error for nonexistent template ID")
	}

	// Test invalid template ID with path traversal
	if err := StartDownload("../invalid-template"); err == nil {
		t.Errorf("expected error for invalid path traversal ID")
	}
}

func TestStartDownloadBusy(t *testing.T) {
	downloadLock.Lock()
	activeJob = "busy-job"
	downloadLock.Unlock()

	defer func() {
		downloadLock.Lock()
		activeJob = ""
		downloadLock.Unlock()
	}()

	if err := StartDownload("debian-12"); err == nil {
		t.Errorf("expected error when download job is busy")
	}
}

func TestApplyPostExtractFixesWithExistingGroup(t *testing.T) {
	tmpDir := t.TempDir()
	etcDir := filepath.Join(tmpDir, "etc")
	_ = os.MkdirAll(etcDir, 0755)
	grpFile := filepath.Join(etcDir, "group")
	_ = os.WriteFile(grpFile, []byte("root:x:0:\nbin:x:1:\n"), 0644)

	applyPostExtractFixes(tmpDir)

	data, err := os.ReadFile(grpFile)
	if err != nil || !strings.Contains(string(data), ":3003:") {
		t.Errorf("expected group file to contain AID_INET: %v, %s", err, string(data))
	}
}

func TestExtractArchiveFailures(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "corrupted.tar")
	_ = os.WriteFile(badFile, []byte("corrupt-data"), 0644)

	outDir := filepath.Join(tmpDir, "out")
	_ = extractArchive(badFile, outDir, "tar.gz")
	_ = extractArchive(badFile, outDir, "tar.xz")
}

func TestStartDownloadSuccess(t *testing.T) {
	tmpStorage := t.TempDir()
	origStorage := os.Getenv("DROIDSPACES_ROOTFS_DIR")
	defer os.Setenv("DROIDSPACES_ROOTFS_DIR", origStorage)
	os.Setenv("DROIDSPACES_ROOTFS_DIR", tmpStorage)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer ts.Close()

	origCatalog := make([]model.TemplateInfo, len(defaultCatalog))
	copy(origCatalog, defaultCatalog)
	defer func() {
		defaultCatalog = origCatalog
	}()

	testID := "test-download-img"
	defaultCatalog = append(defaultCatalog, model.TemplateInfo{
		ID:   testID,
		Name: "Test Image",
		URL:  ts.URL,
		Type: "img",
	})

	if err := StartDownload(testID); err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	// Wait for background job to finish
	for range 50 {
		job, prog, errStr := GetDownloadStatus()
		if job == "" {
			if errStr != "" {
				t.Fatalf("download job failed: %s", errStr)
			}
			break
		}
		_ = prog
		time.Sleep(20 * time.Millisecond)
	}

	// Verify extracted img exists
	destImg := filepath.Join(tmpStorage, testID, "rootfs.img")
	if _, err := os.Stat(destImg); err != nil {
		t.Errorf("expected extracted rootfs.img to exist at %s: %v", destImg, err)
	}
}
