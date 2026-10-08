package config

import (
	"os"
	"sort"
	"strconv"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

// GetBootPriorities scans container configurations and computes the boot sequence.
func GetBootPriorities(runningNames map[string]bool) ([]model.BootPriorityItem, error) {
	var items []model.BootPriorityItem
	seen := make(map[string]bool)

	for _, cdir := range GetContainersDirs() {
		entries, err := os.ReadDir(cdir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			name := ent.Name()
			if seen[name] {
				continue
			}
			seen[name] = true

			cfg, err := ReadContainerConfig(name)
			if err != nil {
				continue
			}

			runAtBoot := cfg["run_at_boot"] == "1" || cfg["run_at_boot"] == "true"
			prio := 0
			if p, err := strconv.Atoi(cfg["run_at_boot_priority"]); err == nil {
				prio = p
			}

			status := "stopped"
			if runningNames[name] {
				status = "running"
			}

			items = append(items, model.BootPriorityItem{
				Name:              name,
				RunAtBoot:         runAtBoot,
				RunAtBootPriority: prio,
				Status:            status,
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].RunAtBoot != items[j].RunAtBoot {
			return items[i].RunAtBoot
		}
		if items[i].RunAtBootPriority != items[j].RunAtBootPriority {
			if items[i].RunAtBootPriority == 0 {
				return false
			}
			if items[j].RunAtBootPriority == 0 {
				return true
			}
			return items[i].RunAtBootPriority < items[j].RunAtBootPriority
		}
		return items[i].Name < items[j].Name
	})

	return items, nil
}

// SetBootPriorities updates run_at_boot and run_at_boot_priority for given containers.
func SetBootPriorities(items []model.BootPriorityItem) error {
	for _, it := range items {
		updates := map[string]string{
			"run_at_boot":          "0",
			"run_at_boot_priority": "0",
		}
		if it.RunAtBoot {
			updates["run_at_boot"] = "1"
			updates["run_at_boot_priority"] = strconv.Itoa(it.RunAtBootPriority)
		}
		if err := WriteContainerConfigKeys(it.Name, updates); err != nil {
			return err
		}
	}
	return nil
}
