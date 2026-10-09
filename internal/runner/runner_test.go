package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

var mockDroidspacesBin string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "runner-mock-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	mockSrc := filepath.Join(tmpDir, "mock_droidspaces.go")
	mockExe := filepath.Join(tmpDir, "mock_droidspaces")
	if runtime.GOOS == "windows" {
		mockExe += ".exe"
	}

	mockCode := `package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	allArgs := strings.Join(os.Args[1:], " ")

	if strings.Contains(allArgs, "fail-box") || strings.Contains(allArgs, "--fail") {
		fmt.Fprintln(os.Stderr, "command failed: simulated error")
		os.Exit(1)
	}

	switch {
	case strings.HasPrefix(allArgs, "show"):
		fmt.Println(` + "`" + `{"running":[{"name":"box-alpha","pid":100,"ip":"172.28.1.5"}],"stopped":[{"name":"box-beta"}]}` + "`" + `)
	case strings.HasPrefix(allArgs, "info"):
		fmt.Println(` + "`" + `{"name":"box-alpha","status":"running","pid":100,"ip":"172.28.1.5"}` + "`" + `)
	case strings.HasPrefix(allArgs, "check"):
		fmt.Println("Environment check: OK")
	case strings.HasPrefix(allArgs, "scan"):
		fmt.Println("Containers found: box-alpha, box-beta")
	case strings.Contains(allArgs, "run"):
		switch {
		case strings.Contains(allArgs, "ps"):
			fmt.Println("PID USER %CPU %MEM COMMAND")
			fmt.Println("1 root 0.0 0.1 /sbin/init")
		case strings.Contains(allArgs, "/etc/passwd"):
			fmt.Println("root")
			fmt.Println("app")
		case strings.Contains(allArgs, "systemctl list-units"):
			fmt.Println("ssh.service loaded active running OpenSSH server")
			fmt.Println("cron.service loaded inactive dead Cron daemon")
		case strings.Contains(allArgs, "rc-status"):
			fmt.Println("Runlevel: default")
			fmt.Println(" ssh [ started ]")
			fmt.Println(" cron [ stopped ]")
		case strings.Contains(allArgs, "journalctl"):
			fmt.Println("Jan 01 00:00:00 host systemd[1]: Starting service...")
		case strings.Contains(allArgs, "service --status-all"):
			fmt.Println(" [ + ] ssh")
			fmt.Println(" [ - ] cron")
		case strings.Contains(allArgs, "cat /etc/os-release"):
			fmt.Println("ID=debian")
		default:
			fmt.Println("mock exec output ok")
		}
	default:
		fmt.Println("success")
	}
}
`
	if err := os.WriteFile(mockSrc, []byte(mockCode), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write mock code: %v\n", err)
		os.Exit(1)
	}

	buildCmd := exec.Command("go", "build", "-o", mockExe, mockSrc)
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to compile mock binary: %v\n", err)
		os.Exit(1)
	}

	mockDroidspacesBin = mockExe
	os.Exit(m.Run())
}

func newTestClient() *Client {
	return &Client{binPath: mockDroidspacesBin}
}

func TestClientLifecycleAndOps(t *testing.T) {
	tmpDir := t.TempDir()
	origEnv := os.Getenv("DROIDSPACES_CONTAINERS_DIR")
	defer os.Setenv("DROIDSPACES_CONTAINERS_DIR", origEnv)
	os.Setenv("DROIDSPACES_CONTAINERS_DIR", tmpDir)

	c := newTestClient()

	if c.BinaryPath() != mockDroidspacesBin {
		t.Errorf("expected BinaryPath %s, got %s", mockDroidspacesBin, c.BinaryPath())
	}

	// 1. Show
	res, err := c.Show()
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}
	if len(res.Running) != 1 || res.Running[0].Name != "box-alpha" {
		t.Errorf("unexpected Show running result: %+v", res.Running)
	}
	if len(res.Stopped) != 1 || res.Stopped[0].Name != "box-beta" {
		t.Errorf("unexpected Show stopped result: %+v", res.Stopped)
	}

	// 2. Info
	info, err := c.Info("box-alpha")
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if info["name"] != "box-alpha" {
		t.Errorf("expected box-alpha, got %v", info["name"])
	}

	// 3. Start
	req := model.StartRequest{
		Name:       "box-alpha",
		RootFS:     "/tmp/rootfs",
		Net:        "nat",
		NATIP:      "172.28.1.5",
		Gateway:    "172.28.1.1",
		Port:       []string{"8080:80"},
		Binds:      []string{"/sdcard:/sdcard"},
		CustomInit: "/sbin/init",
		Memory:     "512M",
		CPUs:       "2",
	}
	if err := c.Start(req); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Start with host network
	reqHost := req
	reqHost.Net = "host"
	if err := c.Start(reqHost); err != nil {
		t.Fatalf("Start host-net failed: %v", err)
	}

	// 4. Stop, Restart, Delete
	if err := c.Stop("box-alpha"); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if err := c.Restart("box-alpha"); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}
	if err := c.Delete("box-alpha"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Test failure with fail-box
	if err := c.Stop("fail-box"); err == nil {
		t.Errorf("expected error stopping fail-box")
	}
	if err := c.Restart("fail-box"); err == nil {
		t.Errorf("expected error restarting fail-box")
	}

	// 5. Check, Scan
	chk, err := c.Check()
	if err != nil || !strings.Contains(chk, "OK") {
		t.Errorf("Check failed: err=%v, out=%s", err, chk)
	}
	scn, err := c.Scan()
	if err != nil || !strings.Contains(scn, "Containers") {
		t.Errorf("Scan failed: err=%v, out=%s", err, scn)
	}

	// 6. Exec & ExecHost
	outExec, err := c.Exec("box-alpha", "ls -l", "root")
	if err != nil || !strings.Contains(outExec, "mock exec output ok") {
		t.Errorf("Exec failed: err=%v, out=%s", err, outExec)
	}
	_, _ = c.ExecHost("echo test")
}

func TestProcessesAndUsers(t *testing.T) {
	c := newTestClient()

	// ListProcesses
	procs, err := c.ListProcesses("box-alpha")
	if err != nil {
		t.Fatalf("ListProcesses failed: %v", err)
	}
	if len(procs) == 0 {
		t.Fatalf("expected non-empty processes")
	}
	if procs[0].PID != 1 || procs[0].Command != "/sbin/init" {
		t.Errorf("unexpected first process: %+v", procs[0])
	}

	// KillProcess
	if err := c.KillProcess("box-alpha", 10, 15); err != nil {
		t.Fatalf("KillProcess failed: %v", err)
	}

	// ListUsers
	users, err := c.ListUsers("box-alpha")
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) < 2 || users[0] != "root" || users[1] != "app" {
		t.Errorf("unexpected users list: %v", users)
	}
}

func TestServicesManagement(t *testing.T) {
	c := newTestClient()

	// DetectInitSystem
	initSys := c.DetectInitSystem("box-alpha")
	if initSys == "" {
		t.Errorf("DetectInitSystem returned empty")
	}

	// ListServices
	svcs, sysName, err := c.ListServices("box-alpha")
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	_ = svcs
	_ = sysName

	// ManageService
	if err := c.ManageService("box-alpha", "start", "ssh"); err != nil {
		t.Fatalf("ManageService start failed: %v", err)
	}

	// GetServiceJournal
	logs, err := c.GetServiceJournal("box-alpha", "ssh", 50)
	if err != nil {
		t.Fatalf("GetServiceJournal failed: %v", err)
	}
	_ = logs
}

func TestBackupsAndExport(t *testing.T) {
	c := newTestClient()
	tmpDir := t.TempDir()
	origEnv := os.Getenv("DROIDSPACES_CONTAINERS_DIR")
	defer os.Setenv("DROIDSPACES_CONTAINERS_DIR", origEnv)
	os.Setenv("DROIDSPACES_CONTAINERS_DIR", tmpDir)

	// Create container with config and rootfs
	boxDir := filepath.Join(tmpDir, "box-alpha")
	_ = os.MkdirAll(boxDir, 0755)
	_ = os.WriteFile(filepath.Join(boxDir, "container.config"), []byte("name=box-alpha\nrootfs_path="+boxDir+"\n"), 0644)

	// ListBackups & ListAllBackups
	_ = os.WriteFile(filepath.Join(tmpDir, "droidspaces_backup_box-alpha_20240101.tar.gz"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "droidspaces_backup_box-beta_20240101.tar.gz"), []byte("data"), 0644)

	backups, err := c.ListBackups("box-alpha")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	_ = backups

	allBackups, _ := c.ListAllBackups()
	_ = allBackups
	// Test Export
	_ = c.Export("box-alpha", filepath.Join(tmpDir, "out.tar.gz"))
	_ = c.Export("missing-box", filepath.Join(tmpDir, "out.tar.gz"))

	// Test DeleteBackup for nonexistent
	_ = c.DeleteBackup("nonexistent.tar.gz")

	// Test NewClient and generateUUID
	realC := NewClient()
	if realC.BinaryPath() == "" {
		t.Errorf("expected non-empty binary path")
	}
	if uuid := generateUUID(); len(uuid) == 0 {
		t.Errorf("expected non-empty UUID")
	}
}

func TestCleanupAndInitVariants(t *testing.T) {
	c := newTestClient()
	tmpDir := t.TempDir()
	origEnv := os.Getenv("DROIDSPACES_CONTAINERS_DIR")
	defer os.Setenv("DROIDSPACES_CONTAINERS_DIR", origEnv)
	os.Setenv("DROIDSPACES_CONTAINERS_DIR", tmpDir)

	c.cleanupRogueConfigs()
	c.pruneStalePID("nonexistent-box")

	// Test static inspection of container rootfs in DetectInitSystem
	openrcBox := filepath.Join(tmpDir, "openrc-box")
	_ = os.MkdirAll(filepath.Join(openrcBox, "rootfs", "etc", "init.d"), 0755)
	_ = os.WriteFile(filepath.Join(openrcBox, "container.config"), []byte("name=openrc-box\nrootfs_path="+filepath.Join(openrcBox, "rootfs")+"\n"), 0644)

	initSys := c.DetectInitSystem("openrc-box")
	if initSys != "openrc" {
		t.Errorf("expected openrc, got %s", initSys)
	}

	// Test ListServices and GetServiceJournal with openrc
	svcs, _, err := c.ListServices("openrc-box")
	if err != nil {
		t.Fatalf("ListServices for openrc failed: %v", err)
	}
	_ = svcs

	logs, err := c.GetServiceJournal("openrc-box", "ssh", 20)
	if err != nil {
		t.Fatalf("GetServiceJournal for openrc failed: %v", err)
	}
	_ = logs
}

func TestCloneValidationsAndExecution(t *testing.T) {
	c := newTestClient()
	tmpDir := t.TempDir()
	origEnv := os.Getenv("DROIDSPACES_CONTAINERS_DIR")
	defer os.Setenv("DROIDSPACES_CONTAINERS_DIR", origEnv)
	os.Setenv("DROIDSPACES_CONTAINERS_DIR", tmpDir)

	// Invalid names
	if err := c.Clone("../bad", "valid-target", false); err == nil {
		t.Errorf("expected error for invalid source name")
	}
	if err := c.Clone("valid-src", "../bad", false); err == nil {
		t.Errorf("expected error for invalid target name")
	}
	if err := c.Clone("same-name", "same-name", false); err == nil {
		t.Errorf("expected error for identical source and target names")
	}
	if err := c.Clone("nonexistent-src", "valid-target", false); err == nil {
		t.Errorf("expected error for nonexistent source container")
	}

	// Setup source container
	srcDir := filepath.Join(tmpDir, "src-box")
	_ = os.MkdirAll(srcDir, 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "container.config"), []byte("name=src-box\nstatic_nat_ip=172.28.1.10\nrootfs_path="+srcDir+"\n"), 0644)
	// Successful clone
	if err := c.Clone("src-box", "cloned-box", false); err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Verify target config exists and has new name and new IP
	targetCfg, err := c.ReadContainerConfig("cloned-box")
	if err != nil || targetCfg["name"] != "cloned-box" {
		t.Fatalf("failed to read cloned config: %v, cfg=%v", err, targetCfg)
	}

	// Cloning again to same target should fail because target already exists
	if err := c.Clone("src-box", "cloned-box", false); err == nil {
		t.Errorf("expected error when target container already exists")
	}
}

func TestRestoreValidations(t *testing.T) {
	c := newTestClient()

	// Invalid target name
	if err := c.Restore(model.RestoreRequest{TargetName: "../bad", Filename: "b.tar.gz"}); err == nil {
		t.Errorf("expected error for invalid target name")
	}

	// Invalid filename extension
	if err := c.Restore(model.RestoreRequest{TargetName: "target", Filename: "b.zip"}); err == nil {
		t.Errorf("expected error for invalid archive format")
	}

	// Nonexistent backup file
	if err := c.Restore(model.RestoreRequest{TargetName: "target", Filename: "missing.tar.gz"}); err == nil {
		t.Errorf("expected error for missing backup file")
	}
}

func TestExtractJSONEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"no json here", "no json here"},
		{"\x1b[32m{\"foo\":\"bar\"}\x1b[0m", "{\"foo\":\"bar\"}"},
		{"prefix { \"a\": 1 } suffix", "{ \"a\": 1 }"},
		{"{nested: {deep: true}}", "{nested: {deep: true}}"},
	}

	for _, tc := range tests {
		got := ExtractJSON(tc.input)
		if got != tc.expected {
			t.Errorf("ExtractJSON(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestClientConfigAndNetworkDelegation(t *testing.T) {
	c := newTestClient()
	tmpDir := t.TempDir()
	origEnv := os.Getenv("DROIDSPACES_CONTAINERS_DIR")
	defer os.Setenv("DROIDSPACES_CONTAINERS_DIR", origEnv)
	os.Setenv("DROIDSPACES_CONTAINERS_DIR", tmpDir)

	boxDir := filepath.Join(tmpDir, "box-cfg")
	_ = os.MkdirAll(boxDir, 0755)
	_ = os.WriteFile(filepath.Join(boxDir, "container.config"), []byte("key1=val1\n"), 0644)

	// ReadContainerConfig
	cfg, err := c.ReadContainerConfig("box-cfg")
	if err != nil || cfg["key1"] != "val1" {
		t.Fatalf("ReadContainerConfig failed: %v, cfg=%v", err, cfg)
	}

	// WriteContainerConfigKeys
	if err := c.WriteContainerConfigKeys("box-cfg", map[string]string{"key2": "val2"}); err != nil {
		t.Fatalf("WriteContainerConfigKeys failed: %v", err)
	}

	// GetBootPriorities & SetBootPriorities
	bp, err := c.GetBootPriorities()
	if err != nil {
		t.Fatalf("GetBootPriorities failed: %v", err)
	}
	if err := c.SetBootPriorities(bp); err != nil {
		t.Fatalf("SetBootPriorities failed: %v", err)
	}

	// ListHostNetworkInterfaces
	ifaces, err := c.ListHostNetworkInterfaces()
	if err != nil {
		t.Fatalf("ListHostNetworkInterfaces failed: %v", err)
	}
	_ = ifaces
}
