package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/latifangren/droidspaces-webui/internal/config"
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

	mockCode := "package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\t\"strings\"\n)\n\nfunc main() {\n\tallArgs := strings.Join(os.Args[1:], \" \")\n\n\tif strings.Contains(allArgs, \"fail-box\") || strings.Contains(allArgs, \"--fail\") {\n\t\tfmt.Fprintln(os.Stderr, \"command failed: simulated error\")\n\t\tos.Exit(1)\n\t}\n\n\tswitch {\n\tcase strings.HasPrefix(allArgs, \"show\"):\n\t\tfmt.Println(`{\"running\":[{\"name\":\"box-alpha\",\"pid\":100,\"ip\":\"172.28.1.5\"}],\"stopped\":[{\"name\":\"box-beta\"}]}`)\n\tcase strings.HasPrefix(allArgs, \"info\"):\n\t\tif strings.Contains(allArgs, \"bad-json\") {\n\t\t\tfmt.Println(`{invalid json`)\n\t\t} else {\n\t\t\tfmt.Println(`{\"name\":\"box-alpha\",\"status\":\"running\",\"pid\":100,\"ip\":\"172.28.1.5\"}`)\n\t\t}\n\tcase strings.HasPrefix(allArgs, \"check\"):\n\t\tfmt.Println(\"Environment check: OK\")\n\tcase strings.HasPrefix(allArgs, \"scan\"):\n\t\tfmt.Println(\"Containers found: box-alpha, box-beta\")\n\tcase strings.Contains(allArgs, \"run\"):\n\t\tswitch {\n\t\tcase strings.Contains(allArgs, \"box-psaux\"):\n\t\t\tfmt.Println(\"USER PID %CPU %MEM VSZ RSS TTY STAT START TIME COMMAND\")\n\t\t\tfmt.Println(\"root 200 0.5 0.2 1234 567 ? S 00:00 0:01 /usr/bin/daemon\")\n\t\tcase strings.Contains(allArgs, \"ps\"):\n\t\t\tfmt.Println(\"PID USER %CPU %MEM COMMAND\")\n\t\t\tfmt.Println(\"1 root 0.0 0.1 /sbin/init\")\n\t\tcase strings.Contains(allArgs, \"/etc/passwd\"):\n\t\t\tfmt.Println(\"root\")\n\t\t\tfmt.Println(\"app\")\n\t\tcase strings.Contains(allArgs, \"systemctl list-unit-files\"):\n\t\t\tfmt.Println(\"ssh.service enabled\")\n\t\t\tfmt.Println(\"cron.service disabled\")\n\t\tcase strings.Contains(allArgs, \"systemctl list-units\"):\n\t\t\tfmt.Println(\"ssh.service loaded active running OpenSSH server\")\n\t\t\tfmt.Println(\"cron.service loaded inactive dead Cron daemon\")\n\t\tcase strings.Contains(allArgs, \"rc-update\"):\n\t\t\tfmt.Println(\"ssh | default\")\n\t\tcase strings.Contains(allArgs, \"rc-status\"):\n\t\t\tfmt.Println(\"Runlevel: default\")\n\t\t\tfmt.Println(\" ssh [ started ]\")\n\t\t\tfmt.Println(\" cron [ stopped ]\")\n\t\tcase strings.Contains(allArgs, \"find /etc/init.d\"):\n\t\t\tfmt.Println(\"/etc/init.d/ssh\")\n\t\t\tfmt.Println(\"/etc/init.d/cron\")\n\t\tcase strings.Contains(allArgs, \"journalctl\"):\n\t\t\tfmt.Println(\"Jan 01 00:00:00 host systemd[1]: Starting service...\")\n\t\tcase strings.Contains(allArgs, \"service --status-all\"):\n\t\t\tfmt.Println(\" [ + ] ssh\")\n\t\t\tfmt.Println(\" [ - ] cron\")\n\t\tcase strings.Contains(allArgs, \"cat /etc/os-release\"):\n\t\t\tfmt.Println(\"ID=debian\")\n\t\tdefault:\n\t\t\tfmt.Println(\"mock exec output ok\")\n\t\t}\n\tdefault:\n\t\tfmt.Println(\"success\")\n\t}\n}\n"

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

	stopBox := filepath.Join(tmpDir, "stopped-box")
	_ = os.MkdirAll(stopBox, 0755)
	_ = os.WriteFile(filepath.Join(stopBox, "container.config"), []byte("name=stopped-box\nrun_at_boot=1\nrun_at_boot_priority=15\nport_forwards=8080:80\nrootfs_path="+stopBox+"\n"), 0644)

	alphaBox := filepath.Join(tmpDir, "box-alpha")
	_ = os.MkdirAll(alphaBox, 0755)
	_ = os.WriteFile(filepath.Join(alphaBox, "container.config"), []byte("name=box-alpha\nrootfs_path="+alphaBox+"\n"), 0644)

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
	if len(res.Stopped) != 2 {
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
	if _, err := c.Info("fail-box"); err == nil {
		t.Errorf("expected error from Info for fail-box")
	}
	if _, err := c.Info("bad-json"); err == nil {
		t.Errorf("expected error from Info for bad-json")
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

	// 1. Systemd container
	c.setInitSysCache("sysd-box", "systemd")
	svcsSysd, sysdName, err := c.ListServices("sysd-box")
	if err != nil || len(svcsSysd) == 0 {
		t.Errorf("ListServices for systemd failed: %v, svcs=%+v", err, svcsSysd)
	}
	_ = sysdName
	_ = c.ManageService("sysd-box", "start", "ssh")
	_, _ = c.GetServiceJournal("sysd-box", "ssh", 50)

	// 2. OpenRC container
	c.setInitSysCache("oprc-box", "openrc")
	svcsOprc, _, err := c.ListServices("oprc-box")
	if err != nil || len(svcsOprc) == 0 {
		t.Errorf("ListServices for openrc failed: %v, svcs=%+v", err, svcsOprc)
	}
	_ = c.ManageService("oprc-box", "stop", "ssh")
	_, _ = c.GetServiceJournal("oprc-box", "ssh", 50)

	// 3. SysVinit container
	c.setInitSysCache("sysv-box", "sysvinit")
	svcsSysv, _, err := c.ListServices("sysv-box")
	if err != nil {
		t.Errorf("ListServices for sysvinit failed: %v", err)
	}
	_ = svcsSysv
	_, _ = c.GetServiceJournal("sysv-box", "ssh", 50)
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
	t.Setenv("DROIDSPACES_CONTAINERS_DIR", tmpDir)
	tmpPids := filepath.Join(tmpDir, "Pids")
	_ = os.MkdirAll(tmpPids, 0755)
	_ = os.WriteFile(filepath.Join(tmpPids, "stale-box.pid"), []byte("9999999\n"), 0644)
	origPids := pidsDirs
	pidsDirs = []string{tmpPids}
	defer func() {
		pidsDirs = origPids
	}()

	c.pruneStalePID("stale-box")
	c.pruneStalePID("nonexistent-box")
	_ = os.WriteFile(filepath.Join(tmpPids, "bad-pid.pid"), []byte("notanumber\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpPids, "neg-pid.pid"), []byte("-5\n"), 0644)
	c.pruneStalePID("bad-pid")
	c.pruneStalePID("neg-pid")

	// Test static inspection of container rootfs in DetectInitSystem
	openrcBox := filepath.Join(tmpDir, "openrc-box")
	_ = os.MkdirAll(filepath.Join(openrcBox, "rootfs", "etc", "init.d"), 0755)
	_ = os.WriteFile(filepath.Join(openrcBox, "container.config"), []byte("name=openrc-box\nrootfs_path="+filepath.Join(openrcBox, "rootfs")+"\n"), 0644)
	cfg, err := c.ReadContainerConfig("openrc-box")
	dirs := config.GetContainersDirs()
	t.Logf("cfg: %+v, err: %v, dirs: %v", cfg, err, dirs)
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

func TestRestoreEndToEndAndSparseClone(t *testing.T) {
	c := newTestClient()
	tmpDir := t.TempDir()
	t.Setenv("DROIDSPACES_CONTAINERS_DIR", filepath.Join(tmpDir, "Containers"))

	origBackups := backupsDir
	tmpBackups := filepath.Join(tmpDir, "Backups")
	_ = os.MkdirAll(tmpBackups, 0755)
	backupsDir = tmpBackups
	defer func() {
		backupsDir = origBackups
	}()

	// 1. Create a container folder with image to clone
	containersDir := filepath.Join(tmpDir, "Containers")
	imgBoxDir := filepath.Join(containersDir, "img-box")
	_ = os.MkdirAll(imgBoxDir, 0755)
	imgPath := filepath.Join(imgBoxDir, "rootfs.img")
	_ = os.WriteFile(imgPath, []byte("fake-raw-img-content"), 0644)
	_ = os.WriteFile(filepath.Join(imgBoxDir, "container.config"), []byte("name=img-box\nrootfs_path="+imgPath+"\n"), 0644)

	// Test Clone with sparse image
	if err := c.Clone("img-box", "cloned-img-box", false); err != nil {
		t.Fatalf("Clone with img rootfs failed: %v", err)
	}

	// 2. Prepare mock archive for Restore
	stageDir := filepath.Join(tmpDir, "stage")
	_ = os.MkdirAll(filepath.Join(stageDir, "rootfs", "etc"), 0755)
	_ = os.WriteFile(filepath.Join(stageDir, "container.config"), []byte("name=stage\nstatic_nat_ip=172.28.1.50\n"), 0644)

	backupFile := filepath.Join(tmpBackups, "box-alpha_20240101.tar.gz")
	cmdTar := exec.Command("tar", "-czf", backupFile, "-C", stageDir, ".")
	if err := cmdTar.Run(); err == nil {
		// Test ListBackups & ListAllBackups
		list, err := c.ListBackups("box-alpha")
		if err != nil || len(list) == 0 {
			t.Errorf("expected ListBackups to find backup: %v, list=%+v", err, list)
		}
		allList, err := c.ListAllBackups()
		if err != nil || len(allList) == 0 {
			t.Errorf("expected ListAllBackups to find backup: %v", err)
		}

		// Test Restore
		req := model.RestoreRequest{
			TargetName: "restored-alpha",
			Filename:   "box-alpha_20240101.tar.gz",
			Overwrite:  true,
			AutoStart:  false,
		}
		if err := c.Restore(req); err != nil {
			t.Fatalf("Restore failed: %v", err)
		}

		// Test DeleteBackup
		if err := c.DeleteBackup("box-alpha_20240101.tar.gz"); err != nil {
			t.Errorf("DeleteBackup failed: %v", err)
		}
	}
}

func TestStartComprehensiveAndDiskSize(t *testing.T) {
	c := newTestClient()
	tmpDir := t.TempDir()

	// 1. Start with full flags
	reqFull := model.StartRequest{
		Name:              "super-box",
		RootFS:            "/tmp/rootfs",
		Net:               "gateway",
		Gateway:           "192.168.1.1",
		GatewayNet:        "192.168.1.0/24",
		GatewayIface:      "wlan0",
		GatewayBridge:     "br-gw",
		DNS:               "1.1.1.1",
		Memory:            "1G",
		CPUs:              "4",
		PIDsLimit:         "200",
		Privileged:        "true",
		DisableIPv6:       true,
		AndroidStorage:    true,
		HWAccess:          true,
		GPU:               true,
		TermuxX11:         true,
		VirGL:             true,
		PulseAudio:        true,
		SELinuxPermissive: true,
		Volatile:          true,
		ForceCgroupV1:     true,
		AllowSandboxing:   true,
		EnvVars:           []string{"FOO=BAR", "BAZ=QUX"},
		Binds:             []string{"/sdcard:/sdcard"},
		Port:              []string{"80:80/tcp", "53:53/udp"},
		CustomInit:        "/sbin/init",
	}
	if err := c.Start(reqFull); err != nil {
		t.Fatalf("Start full options failed: %v", err)
	}

	// 2. GetContainerDiskSize
	fPath := filepath.Join(tmpDir, "disk.img")
	_ = os.WriteFile(fPath, make([]byte, 1024), 0644)
	strSize, numBytes := GetContainerDiskSize(fPath)
	if numBytes != 1024 || strSize == "" {
		t.Errorf("unexpected disk size for file: %d, %s", numBytes, strSize)
	}

	// Directory disk size
	dPath := filepath.Join(tmpDir, "dir-disk")
	_ = os.MkdirAll(dPath, 0755)
	_ = os.WriteFile(filepath.Join(dPath, "f1"), make([]byte, 512), 0644)
	strDir, numDir := GetContainerDiskSize(dPath)
	if numDir < 512 || strDir == "" {
		t.Errorf("unexpected disk size for dir: %d, %s", numDir, strDir)
	}

	// Non-existent path
	sEmpty, nZero := GetContainerDiskSize(filepath.Join(tmpDir, "nonexistent"))
	if nZero != 0 || sEmpty != "0 B" {
		t.Errorf("expected 0 B for nonexistent path, got %d, %s", nZero, sEmpty)
	}

	// 3. ManageService actions
	for _, action := range []string{"stop", "restart", "enable", "disable", "reload"} {
		_ = c.ManageService("super-box", action, "ssh")
	}
}


func TestRestoreUncompressedAndAutoStart(t *testing.T) {
	c := newTestClient()
	tmpDir := t.TempDir()
	t.Setenv("DROIDSPACES_CONTAINERS_DIR", filepath.Join(tmpDir, "Containers"))

	origBackups := backupsDir
	tmpBackups := filepath.Join(tmpDir, "Backups")
	_ = os.MkdirAll(tmpBackups, 0755)
	backupsDir = tmpBackups
	defer func() {
		backupsDir = origBackups
	}()

	stageDir := filepath.Join(tmpDir, "stage2")
	_ = os.MkdirAll(filepath.Join(stageDir, "rootfs", "etc"), 0755)
	_ = os.WriteFile(filepath.Join(stageDir, "container.config"), []byte("name=stage2\n"), 0644)

	// Uncompressed tar
	backupFile := filepath.Join(tmpBackups, "uncomp.tar")
	cmdTar := exec.Command("tar", "-cf", backupFile, "-C", stageDir, ".")
	if err := cmdTar.Run(); err == nil {
		// Restore with Overwrite: false and AutoStart: true
		req := model.RestoreRequest{
			TargetName: "target-fresh",
			Filename:   "uncomp.tar",
			Overwrite:  false,
			AutoStart:  true,
		}
		if err := c.Restore(req); err != nil {
			t.Fatalf("Restore fresh failed: %v", err)
		}

		// Restoring to target-fresh again without Overwrite must fail
		if err := c.Restore(req); err == nil {
			t.Errorf("expected error when restoring existing container without overwrite")
		}
	}

	// Test Clone with autoStart
	srcDir := filepath.Join(tmpDir, "Containers", "target-fresh")
	_ = c.Clone("target-fresh", "cloned-autostart", true)
	_ = srcDir
}

func TestProcessesEdgeCases(t *testing.T) {
	c := newTestClient()

	// KillProcess invalid pid
	if err := c.KillProcess("box-alpha", 0, 9); err == nil {
		t.Errorf("expected error for invalid PID 0")
	}

	// KillProcess default signal (signal <= 0 -> 9)
	if err := c.KillProcess("box-alpha", 123, 0); err != nil {
		t.Errorf("expected default signal kill to succeed: %v", err)
	}

	// ListUsers failure fallback to root
	users, err := c.ListUsers("fail-box")
	if err != nil || len(users) != 1 || users[0] != "root" {
		t.Errorf("expected fallback to root on list users failure: %v, %v", err, users)
	}

	// ListProcesses ps aux format
	procsAux, err := c.ListProcesses("box-psaux")
	if err != nil || len(procsAux) == 0 || procsAux[0].PID != 200 {
		t.Errorf("expected ps aux parse to succeed: %v, procs=%+v", err, procsAux)
	}
}