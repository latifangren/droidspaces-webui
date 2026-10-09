package hardware

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type containerSample struct {
	ticks      int64
	sampleTime time.Time
}

var (
	samplerMu   sync.RWMutex
	prevSamples = make(map[string]containerSample)
	cpuPercents = make(map[string]float64)
	samplerOnce sync.Once
)

// GetContainerCPU returns the latest calculated CPU % for a container.
func GetContainerCPU(name string, pid int) float64 {
	samplerMu.RLock()
	pct, ok := cpuPercents[name]
	samplerMu.RUnlock()
	if ok {
		return pct
	}
	return 0.0
}

// StartContainerCPUSampler periodically calculates real-time CPU percentage per running container.
func StartContainerCPUSampler(getRunningContainers func() map[string]int) {
	samplerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			for range ticker.C {
				containers := getRunningContainers()
				now := time.Now()

				samplerMu.Lock()
				// Evict stopped containers
				for name := range cpuPercents {
					if _, running := containers[name]; !running {
						delete(cpuPercents, name)
						delete(prevSamples, name)
					}
				}

				for name, pid := range containers {
					if pid <= 0 {
						continue
					}
					currentTicks := getContainerTotalTicks(name, pid)
					prev, hasPrev := prevSamples[name]
					if hasPrev {
						dt := now.Sub(prev.sampleTime).Seconds()
						if dt >= 0.5 {
							dTicks := float64(currentTicks - prev.ticks)
							if dTicks < 0 {
								dTicks = 0
							}
							// Standard Linux USER_HZ = 100 ticks/sec.
							// CPU% = (dTicks / (dt * 100)) * 100 = dTicks / dt
							pct := dTicks / dt
							if pct < 0 {
								pct = 0
							}
							cpuPercents[name] = float64(int(pct*10)) / 10.0
						}
					}
					prevSamples[name] = containerSample{
						ticks:      currentTicks,
						sampleTime: now,
					}
				}
				samplerMu.Unlock()
			}
		}()
	})
}

func getContainerTotalTicks(name string, rootPid int) int64 {
	if rootPid <= 0 {
		return 0
	}

	var totalTicks int64
	seenPid := make(map[int]bool)

	// 1. Check cgroup.procs in container's cgroup hierarchy
	cgroupBase := filepath.Join("/sys/fs/cgroup/droidspaces", name)
	_ = filepath.Walk(cgroupBase, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Name() == "cgroup.procs" {
			if data, err := os.ReadFile(path); err == nil {
				scanner := bufio.NewScanner(bytes.NewReader(data))
				for scanner.Scan() {
					if pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil && pid > 0 {
						if !seenPid[pid] {
							seenPid[pid] = true
							totalTicks += readPidTicks(pid)
						}
					}
				}
			}
		}
		return nil
	})

	if !seenPid[rootPid] {
		seenPid[rootPid] = true
		totalTicks += readPidTicks(rootPid)
	}

	// 2. Scan /proc for descendant processes belonging to rootPid
	entries, err := os.ReadDir("/proc")
	if err == nil {
		type procNode struct {
			ppid  int
			ticks int64
		}
		procMap := make(map[int]procNode)

		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			pid, err := strconv.Atoi(ent.Name())
			if err != nil || pid <= 0 {
				continue
			}
			ppid, ticks := readPidStat(pid)
			if ppid > 0 {
				procMap[pid] = procNode{ppid: ppid, ticks: ticks}
			}
		}

		descendants := map[int]bool{rootPid: true}
		for {
			added := false
			for p, node := range procMap {
				if descendants[node.ppid] && !descendants[p] {
					descendants[p] = true
					added = true
				}
			}
			if !added {
				break
			}
		}

		for p := range descendants {
			if !seenPid[p] {
				seenPid[p] = true
				totalTicks += procMap[p].ticks
			}
		}
	}

	return totalTicks
}

func readPidTicks(pid int) int64 {
	_, ticks := readPidStat(pid)
	return ticks
}

func readPidStat(pid int) (int, int64) {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0, 0
	}

	str := string(data)
	idx := strings.LastIndex(str, ")")
	if idx == -1 || idx+2 >= len(str) {
		return 0, 0
	}

	fields := strings.Fields(str[idx+2:])
	// fields[1] = ppid, fields[11] = utime, fields[12] = stime
	if len(fields) >= 13 {
		ppid, _ := strconv.Atoi(fields[1])
		utime, _ := strconv.ParseInt(fields[11], 10, 64)
		stime, _ := strconv.ParseInt(fields[12], 10, 64)
		return ppid, utime + stime
	}
	return 0, 0
}
