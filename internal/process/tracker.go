package process

import (
	"fmt"
	"math"
	"sort"

	"github.com/shirou/gopsutil/v4/process"
)

type ProcessItem struct {
	PID               int32   `json:"pid"`
	Name              string  `json:"name"`
	CPUPercent        float64 `json:"cpu_percent"`
	MemoryMB          float64 `json:"memory_mb"`
	MemoryPercent     float32 `json:"memory_percent"`
	EnergyImpact      float64 `json:"energy_impact"`
	EnergyImpactLabel string  `json:"energy_impact_label"`
}

// GetTopProcesses returns the top resource consuming processes sorted by Memory & CPU usage
func GetTopProcesses(limit int) ([]ProcessItem, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate processes: %w", err)
	}

	items := []ProcessItem{}
	for _, p := range procs {
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}

		memInfo, err := p.MemoryInfo()
		if err != nil || memInfo == nil {
			continue
		}

		memPct, _ := p.MemoryPercent()
		cpuPct, _ := p.CPUPercent()

		memMB := float64(memInfo.RSS) / 1024 / 1024

		// Calculate Battery & Energy Impact Score (1-100)
		energyScore := math.Min(100, (cpuPct*2.2)+(float64(memPct)*1.5))
		energyScore = math.Round(energyScore*10) / 10

		label := "Low"
		if energyScore > 50 {
			label = "High"
		} else if energyScore > 20 {
			label = "Moderate"
		}

		items = append(items, ProcessItem{
			PID:               p.Pid,
			Name:              name,
			CPUPercent:        round1(cpuPct),
			MemoryMB:          round1(memMB),
			MemoryPercent:     memPct,
			EnergyImpact:      energyScore,
			EnergyImpactLabel: label,
		})
	}

	// Sort primarily by Memory footprint, secondarily by CPU %
	sort.Slice(items, func(i, j int) bool {
		if items[i].MemoryMB == items[j].MemoryMB {
			return items[i].CPUPercent > items[j].CPUPercent
		}
		return items[i].MemoryMB > items[j].MemoryMB
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// TerminateProcess safely terminates a process by PID
func TerminateProcess(pid int32) error {
	// Security check to avoid terminating critical system processes
	if pid <= 4 {
		return fmt.Errorf("refusing to terminate system process PID %d", pid)
	}

	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("process with PID %d not found: %w", pid, err)
	}

	return p.Kill()
}

func round1(val float64) float64 {
	return float64(int(val*10)) / 10
}
