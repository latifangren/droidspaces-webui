package hardware

import (
	"bufio"
	"bytes"
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

type procNode struct {
	ppid  int
	ticks int64
}

var (
	samplerMu                sync.RWMutex
	prevSamples              = make(map[string]containerSample)
	prevPIDs                 = make(map[string]int)
	cpuPercents              = make(map[string]float64)
	samplerOnce              sync.Once
	sampleInterval           = 2 * time.Second
	containerRefreshInterval = 10 * time.Second
	cgroupBaseDir            = "/sys/fs/cgroup/droidspaces"
	procDir                  = "/proc"
)

// GetContainerCPU returns the latest calculated CPU % for a container.
func GetContainerCPU(name string, pid int) float64 {
	samplerMu.RLock()
	pct, ok := cpuPercents[name]
	sampledPID, hasPID := prevPIDs[name]
	samplerMu.RUnlock()
	if ok && hasPID && sampledPID == pid {
		return pct
	}
	return 0.0
}

// StartContainerCPUSampler periodically calculates real-time CPU percentage per running container.
func StartContainerCPUSampler(getRunningContainers func() map[string]int) {
	samplerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(sampleInterval)
			defer ticker.Stop()
			var containers map[string]int
			var lastRefresh time.Time
			for now := range ticker.C {
				if containers == nil || now.Sub(lastRefresh) >= containerRefreshInterval {
					containers = getRunningContainers()
					lastRefresh = now
				}
				procSnapshot := readProcSnapshot()
				nextTicks := make(map[string]int64, len(containers))
				for name, pid := range containers {
					if pid > 0 {
						nextTicks[name] = getContainerTotalTicksFromSnapshot(name, pid, procSnapshot)
					}
				}

				samplerMu.Lock()
				for name := range cpuPercents {
					if _, running := containers[name]; !running {
						delete(cpuPercents, name)
						delete(prevSamples, name)
						delete(prevPIDs, name)
					}
				}
				for name, currentTicks := range nextTicks {
					currentPID := containers[name]
					prev, hasPrev := prevSamples[name]
					lastPID, hadPID := prevPIDs[name]
					// Drop the stale baseline when the PID changes (container
					// restarted with the same name) so we never report the prior
					// process's CPU usage.
					if hadPID && lastPID != currentPID {
						hasPrev = false
						delete(prevSamples, name)
						delete(cpuPercents, name)
					}
					if hasPrev {
						dt := now.Sub(prev.sampleTime).Seconds()
						if dt >= 0.5 {
							dTicks := float64(currentTicks - prev.ticks)
							if dTicks < 0 {
								dTicks = 0
							}
							pct := dTicks / dt
							cpuPercents[name] = float64(int(pct*10)) / 10.0
						}
					}
					prevSamples[name] = containerSample{ticks: currentTicks, sampleTime: now}
					prevPIDs[name] = currentPID
				}
				samplerMu.Unlock()
			}
		}()
	})
}

func readProcSnapshot() map[int]procNode {
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return nil
	}
	snapshot := make(map[int]procNode, len(entries))
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
			snapshot[pid] = procNode{ppid: ppid, ticks: ticks}
		}
	}
	return snapshot
}

func getContainerTotalTicks(name string, rootPid int) int64 {
	if rootPid <= 0 {
		return 0
	}
	return getContainerTotalTicksFromSnapshot(name, rootPid, readProcSnapshot())
}

func getContainerTotalTicksFromSnapshot(name string, rootPid int, procSnapshot map[int]procNode) int64 {
	if rootPid <= 0 {
		return 0
	}
	var totalTicks int64
	seenPid := make(map[int]bool)
	cgroupBase := filepath.Join(cgroupBaseDir, name)
	_ = filepath.Walk(cgroupBase, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() && info.Name() == "cgroup.procs" {
			if data, err := os.ReadFile(path); err == nil {
				scanner := bufio.NewScanner(bytes.NewReader(data))
				for scanner.Scan() {
					if pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil && pid > 0 && !seenPid[pid] {
						seenPid[pid] = true
						node, ok := procSnapshot[pid]
						if ok {
							totalTicks += node.ticks
						} else {
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
		if node, ok := procSnapshot[rootPid]; ok {
			totalTicks += node.ticks
		} else {
			totalTicks += readPidTicks(rootPid)
		}
	}

	children := make(map[int][]int)
	for pid, node := range procSnapshot {
		children[node.ppid] = append(children[node.ppid], pid)
	}
	queue := []int{rootPid}
	for len(queue) > 0 {
		parent := queue[0]
		queue = queue[1:]
		for _, child := range children[parent] {
			if seenPid[child] {
				continue
			}
			seenPid[child] = true
			if node, ok := procSnapshot[child]; ok {
				totalTicks += node.ticks
			}
			queue = append(queue, child)
		}
	}
	return totalTicks
}

func readPidTicks(pid int) int64 {
	_, ticks := readPidStat(pid)
	return ticks
}

func readPidStat(pid int) (int, int64) {
	statPath := filepath.Join(procDir, strconv.Itoa(pid), "stat")
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
	if len(fields) >= 13 {
		ppid, _ := strconv.Atoi(fields[1])
		utime, _ := strconv.ParseInt(fields[11], 10, 64)
		stime, _ := strconv.ParseInt(fields[12], 10, 64)
		return ppid, utime + stime
	}
	return 0, 0
}
