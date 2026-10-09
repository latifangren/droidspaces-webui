package hardware

import (
	"fmt"
	"os"
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
