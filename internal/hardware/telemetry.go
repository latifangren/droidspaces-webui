package hardware

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	cpuThermalZonePath string
	cpuThermalOnce     sync.Once

	thermalZoneTypeFmt = "/sys/class/thermal/thermal_zone%d/type"
	thermalZoneTempFmt = "/sys/class/thermal/thermal_zone%d/temp"
	batteryTempPath    = "/sys/class/power_supply/battery/temp"
	batteryCapPath     = "/sys/class/power_supply/battery/capacity"
	batteryStatusPath  = "/sys/class/power_supply/battery/status"
	procLoadavgPath    = "/proc/loadavg"
	procMeminfoPath    = "/proc/meminfo"
	storagePath        = "/data"
)

func discoverCPUThermalZone() string {
	for i := range 40 {
		typePath := fmt.Sprintf(thermalZoneTypeFmt, i)
		tempPath := fmt.Sprintf(thermalZoneTempFmt, i)
		if zType, err := os.ReadFile(typePath); err == nil {
			tStr := strings.ToLower(string(zType))
			if strings.Contains(tStr, "cpu") || strings.Contains(tStr, "soc") || strings.Contains(tStr, "tsens") {
				if _, err := os.ReadFile(tempPath); err == nil {
					return tempPath
				}
			}
		}
	}
	return ""
}

type HardwareStats struct {
	BatteryLevelPct int     `json:"battery_level_pct"`
	BatteryTempC    float64 `json:"battery_temp_c"`
	BatteryStatus   string  `json:"battery_status"`
	CPUTempC        float64 `json:"cpu_temp_c"`
	CPULoad1m       float64 `json:"cpu_load_1m"`
	RAMUsedMB       int64   `json:"ram_used_mb"`
	RAMTotalMB      int64   `json:"ram_total_mb"`
	StorageFreeGB   float64 `json:"storage_free_gb"`
	StorageTotalGB  float64 `json:"storage_total_gb"`
	StorageUsedGB   float64 `json:"storage_used_gb"`
	StorageUsedPct  float64 `json:"storage_used_pct"`
}

type HostSystemInfo struct {
	DeviceModel string `json:"device_model"`
	Kernel      string `json:"kernel"`
	Arch        string `json:"arch"`
	RootType    string `json:"root_type"`
	SELinux     string `json:"selinux"`
	Uptime      string `json:"uptime"`
}

func GetHostSystemInfo() HostSystemInfo {
	info := HostSystemInfo{
		Arch: runtime.GOARCH,
	}

	// 1. Device Model from Android getprop or build.prop
	if out, err := exec.Command("getprop", "ro.product.model").Output(); err == nil && len(out) > 0 {
		info.DeviceModel = strings.TrimSpace(string(out))
	} else if data, err := os.ReadFile("/system/build.prop"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "ro.product.model=") {
				info.DeviceModel = strings.TrimSpace(strings.TrimPrefix(line, "ro.product.model="))
				break
			}
		}
	}
	if info.DeviceModel == "" {
		info.DeviceModel = runtime.GOOS + " (" + runtime.GOARCH + ")"
	}

	// 2. Kernel release
	if data, err := os.ReadFile("/proc/version"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			info.Kernel = parts[2]
		}
	}
	if info.Kernel == "" {
		if out, err := exec.Command("uname", "-r").Output(); err == nil {
			info.Kernel = strings.TrimSpace(string(out))
		}
	}
	if info.Kernel == "" {
		info.Kernel = "Unknown"
	}

	// 3. Root Subsystem
	if _, err := os.Stat("/data/adb/ksu"); err == nil {
		info.RootType = "KernelSU"
	} else if _, err := os.Stat("/data/adb/ap"); err == nil {
		info.RootType = "APatch"
	} else if _, err := os.Stat("/data/adb/magisk"); err == nil {
		info.RootType = "Magisk"
	} else if os.Getuid() == 0 {
		info.RootType = "Root (uid=0)"
	} else {
		info.RootType = "Non-root"
	}

	// 4. SELinux Mode
	if out, err := exec.Command("getenforce").Output(); err == nil && len(out) > 0 {
		info.SELinux = strings.TrimSpace(string(out))
	} else if data, err := os.ReadFile("/sys/fs/selinux/enforce"); err == nil {
		if strings.TrimSpace(string(data)) == "1" {
			info.SELinux = "Enforcing"
		} else {
			info.SELinux = "Permissive"
		}
	} else {
		info.SELinux = "Disabled"
	}

	// 5. Uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if sec, err := strconv.ParseFloat(fields[0], 64); err == nil {
				d := int(sec) / 86400
				h := (int(sec) % 86400) / 3600
				m := (int(sec) % 3600) / 60
				if d > 0 {
					info.Uptime = fmt.Sprintf("%dd %dh %dm", d, h, m)
				} else if h > 0 {
					info.Uptime = fmt.Sprintf("%dh %dm", h, m)
				} else {
					info.Uptime = fmt.Sprintf("%dm", m)
				}
			}
		}
	}
	if info.Uptime == "" {
		info.Uptime = "Active"
	}

	return info
}

func GetStats() HardwareStats {
	var s HardwareStats

	// 1. Battery Temp & Level
	if data, err := os.ReadFile(batteryTempPath); err == nil {
		if t, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			if t > 200 {
				s.BatteryTempC = t / 10.0
			} else {
				s.BatteryTempC = t
			}
		}
	}

	if data, err := os.ReadFile(batteryCapPath); err == nil {
		if cap, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			s.BatteryLevelPct = cap
		}
	}

	if data, err := os.ReadFile(batteryStatusPath); err == nil {
		s.BatteryStatus = strings.TrimSpace(string(data))
	}
	// 2. CPU Temperature from cached thermal zone
	cpuThermalOnce.Do(func() {
		cpuThermalZonePath = discoverCPUThermalZone()
	})

	if cpuThermalZonePath != "" {
		if zTemp, err := os.ReadFile(cpuThermalZonePath); err == nil {
			if t, err := strconv.ParseFloat(strings.TrimSpace(string(zTemp)), 64); err == nil {
				if t > 1000 {
					s.CPUTempC = t / 1000.0
				} else {
					s.CPUTempC = t
				}
			}
		}
	}

	// 3. Load average
	if data, err := os.ReadFile(procLoadavgPath); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) > 0 {
			if l, err := strconv.ParseFloat(parts[0], 64); err == nil {
				s.CPULoad1m = l
			}
		}
	}

	// 4. Meminfo
	if data, err := os.ReadFile(procMeminfoPath); err == nil {
		var totalKB, availKB int64
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					totalKB, _ = strconv.ParseInt(fields[1], 10, 64)
				}
			} else if strings.HasPrefix(line, "MemAvailable:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					availKB, _ = strconv.ParseInt(fields[1], 10, 64)
				}
			}
		}
		if totalKB > 0 {
			s.RAMTotalMB = totalKB / 1024
			s.RAMUsedMB = (totalKB - availKB) / 1024
		}
	}

	// 5. Storage /data
	s.StorageTotalGB, s.StorageFreeGB = getStorageStats(storagePath)
	if s.StorageTotalGB > 0 {
		s.StorageUsedGB = s.StorageTotalGB - s.StorageFreeGB
		if s.StorageUsedGB < 0 {
			s.StorageUsedGB = 0
		}
		s.StorageUsedPct = (s.StorageUsedGB / s.StorageTotalGB) * 100.0
	}
	return s
}
