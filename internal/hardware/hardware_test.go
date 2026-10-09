package hardware

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetStatsDefault(t *testing.T) {
	// Should not panic on any platform (graceful fallback when sysfs not present)
	stats := GetStats()
	if stats.RAMTotalMB < 0 || stats.RAMUsedMB < 0 {
		t.Errorf("invalid RAM values: %+v", stats)
	}

	used, total := getStorageStats(os.TempDir())
	if total < 0 || used < 0 {
		t.Errorf("storage stats returned negative: used=%f, total=%f", used, total)
	}
}

func TestGetStatsMocked(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup mock battery
	bTemp := filepath.Join(tmpDir, "battery_temp")
	bCap := filepath.Join(tmpDir, "battery_cap")
	bStat := filepath.Join(tmpDir, "battery_stat")
	_ = os.WriteFile(bTemp, []byte("350\n"), 0644) // 350 > 200 -> 35.0 C
	_ = os.WriteFile(bCap, []byte("85\n"), 0644)
	_ = os.WriteFile(bStat, []byte("Charging\n"), 0644)

	// Setup mock proc
	pLoad := filepath.Join(tmpDir, "loadavg")
	pMem := filepath.Join(tmpDir, "meminfo")
	_ = os.WriteFile(pLoad, []byte("1.25 0.95 0.80 2/450 1234\n"), 0644)
	_ = os.WriteFile(pMem, []byte("MemTotal:        8000000 kB\nMemFree:         2000000 kB\nMemAvailable:    4000000 kB\n"), 0644)

	// Setup mock thermal
	tType := filepath.Join(tmpDir, "thermal_type_%d")
	tTemp := filepath.Join(tmpDir, "thermal_temp_%d")
	thermalZone0Type := fmt.Sprintf(tType, 0)
	thermalZone0Temp := fmt.Sprintf(tTemp, 0)
	_ = os.WriteFile(thermalZone0Type, []byte("cpu-thermal-zone\n"), 0644)
	_ = os.WriteFile(thermalZone0Temp, []byte("45000\n"), 0644) // 45000 > 1000 -> 45.0 C

	// Point hardware package paths to mock files
	origBTemp, origBCap, origBStat := batteryTempPath, batteryCapPath, batteryStatusPath
	origPLoad, origPMem := procLoadavgPath, procMeminfoPath
	origTTypeFmt, origTTempFmt := thermalZoneTypeFmt, thermalZoneTempFmt
	origStorage := storagePath

	defer func() {
		batteryTempPath, batteryCapPath, batteryStatusPath = origBTemp, origBCap, origBStat
		procLoadavgPath, procMeminfoPath = origPLoad, origPMem
		thermalZoneTypeFmt, thermalZoneTempFmt = origTTypeFmt, origTTempFmt
		storagePath = origStorage
	}()

	batteryTempPath = bTemp
	batteryCapPath = bCap
	batteryStatusPath = bStat
	procLoadavgPath = pLoad
	procMeminfoPath = pMem
	thermalZoneTypeFmt = tType
	thermalZoneTempFmt = tTemp
	storagePath = tmpDir

	// Invalidate thermal cached zone for testing
	cpuThermalZonePath = ""
	foundZone := discoverCPUThermalZone()
	if foundZone != thermalZone0Temp {
		t.Errorf("expected foundZone %s, got %s", thermalZone0Temp, foundZone)
	}
	cpuThermalZonePath = foundZone

	stats := GetStats()
	if stats.BatteryTempC != 35.0 {
		t.Errorf("expected battery temp 35.0, got %f", stats.BatteryTempC)
	}
	if stats.BatteryLevelPct != 85 {
		t.Errorf("expected battery cap 85, got %d", stats.BatteryLevelPct)
	}
	if stats.BatteryStatus != "Charging" {
		t.Errorf("expected battery status Charging, got %s", stats.BatteryStatus)
	}
	if stats.CPUTempC != 45.0 {
		t.Errorf("expected cpu temp 45.0, got %f", stats.CPUTempC)
	}
	if stats.CPULoad1m != 1.25 {
		t.Errorf("expected cpu load 1.25, got %f", stats.CPULoad1m)
	}
	if stats.RAMTotalMB != 8000000/1024 {
		t.Errorf("expected RAM total %d, got %d", 8000000/1024, stats.RAMTotalMB)
	}
	if stats.RAMUsedMB != (8000000-4000000)/1024 {
		t.Errorf("expected RAM used %d, got %d", (8000000-4000000)/1024, stats.RAMUsedMB)
	}

	// Test battery temp <= 200 and cpu temp <= 1000
	_ = os.WriteFile(bTemp, []byte("38.5\n"), 0644)
	_ = os.WriteFile(thermalZone0Temp, []byte("50\n"), 0644)
	stats2 := GetStats()
	if stats2.BatteryTempC != 38.5 {
		t.Errorf("expected battery temp 38.5, got %f", stats2.BatteryTempC)
	}
	if stats2.CPUTempC != 50.0 {
		t.Errorf("expected cpu temp 50.0, got %f", stats2.CPUTempC)
	}
}

func TestCPUSamplerFull(t *testing.T) {
	tmpDir := t.TempDir()

	origCgroup := cgroupBaseDir
	origProc := procDir
	defer func() {
		cgroupBaseDir = origCgroup
		procDir = origProc
	}()

	mockCgroup := filepath.Join(tmpDir, "cgroup")
	mockProc := filepath.Join(tmpDir, "proc")
	cgroupBaseDir = mockCgroup
	procDir = mockProc

	// Setup container cgroup with procs
	contCgroupDir := filepath.Join(mockCgroup, "test-box", "subgroup")
	_ = os.MkdirAll(contCgroupDir, 0755)
	cgroupProcs := filepath.Join(contCgroupDir, "cgroup.procs")
	_ = os.WriteFile(cgroupProcs, []byte("101\n102\n"), 0644)

	// Setup proc dirs for pid 100 (root), 101, 102, 103 (child of 100)
	// Linux /proc/[pid]/stat format:
	// pid (comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime ...
	// fields after ')' are 1-indexed in code:
	// fields[1] = ppid
	// fields[11] = utime
	// fields[12] = stime
	makeStat := func(pid, ppid, utime, stime int) string {
		return fmt.Sprintf("%d (app) S %d 0 0 0 0 0 0 0 0 0 %d %d 0 0\n", pid, ppid, utime, stime)
	}

	for _, p := range []struct {
		pid, ppid, utime, stime int
	}{
		{100, 1, 10, 5},   // rootPid
		{101, 100, 20, 5}, // cgroup pid 1
		{102, 100, 30, 5}, // cgroup pid 2
		{103, 100, 15, 5}, // child of 100 in proc tree
		{200, 1, 99, 99},  // unrelated process
	} {
		pDir := filepath.Join(mockProc, fmt.Sprintf("%d", p.pid))
		_ = os.MkdirAll(pDir, 0755)
		_ = os.WriteFile(filepath.Join(pDir, "stat"), []byte(makeStat(p.pid, p.ppid, p.utime, p.stime)), 0644)
	}

	// Test readPidStat
	ppid, ticks := readPidStat(100)
	if ppid != 1 || ticks != 15 {
		t.Errorf("readPidStat(100) expected ppid=1, ticks=15, got %d, %d", ppid, ticks)
	}

	// Test readPidStat bad formatting
	badPidDir := filepath.Join(mockProc, "999")
	_ = os.MkdirAll(badPidDir, 0755)
	_ = os.WriteFile(filepath.Join(badPidDir, "stat"), []byte("invalid stat"), 0644)
	pBad, tBad := readPidStat(999)
	if pBad != 0 || tBad != 0 {
		t.Errorf("expected 0,0 for invalid stat, got %d, %d", pBad, tBad)
	}

	// Test getContainerTotalTicks with rootPid 100
	totalTicks := getContainerTotalTicks("test-box", 100)
	// Expected ticks:
	// 101: 25 ticks
	// 102: 35 ticks
	// 100: 15 ticks
	// 103 (child of 100): 20 ticks
	// Total = 25 + 35 + 15 + 20 = 95
	if totalTicks != 95 {
		t.Errorf("expected totalTicks 95, got %d", totalTicks)
	}

	// Test getContainerTotalTicks with rootPid <= 0
	if zeroTicks := getContainerTotalTicks("test-box", 0); zeroTicks != 0 {
		t.Errorf("expected 0 ticks for rootPid 0, got %d", zeroTicks)
	}

	sampleInterval = 10 * time.Millisecond
	pass := 0
	mockFn := func() map[string]int {
		pass++
		if pass > 2 {
			return map[string]int{}
		}
		return map[string]int{
			"test-box": 100,
			"stopped":  -1,
		}
	}
	StartContainerCPUSampler(mockFn)
	time.Sleep(150 * time.Millisecond)

	_ = GetContainerCPU("test-box", 100)
	_ = GetContainerCPU("unknown", 0)
}
