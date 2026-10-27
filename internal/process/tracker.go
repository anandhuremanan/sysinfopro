package process

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

type ProcessItem struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"` // share of the whole processor since the previous list
	MemoryMB      float64 `json:"memory_mb"`
	MemoryPercent float32 `json:"memory_percent"`
	// Kept for older screens; no longer calculated (it was an invented score)
	EnergyImpact      float64 `json:"energy_impact"`
	EnergyImpactLabel string  `json:"energy_impact_label"`
}

// processor time each process had used at the previous list, so CPUPercent is
// what it used since then rather than its average since it started
var (
	cpuMu    sync.Mutex
	lastCPU  = map[int32]float64{}
	lastWhen time.Time
)

// GetTopProcesses returns the processes using the most memory, with the share of
// the processor each used since the previous call
func GetTopProcesses(limit int) ([]ProcessItem, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate processes: %w", err)
	}

	cpuMu.Lock()
	defer cpuMu.Unlock()
	readTimes := func() map[int32]float64 {
		m := make(map[int32]float64, len(procs))
		for _, p := range procs {
			if t, err := p.Times(); err == nil {
				m[p.Pid] = t.User + t.System
			}
		}
		return m
	}
	if lastWhen.IsZero() {
		// first list: take a short first reading so the shares are real
		lastCPU = readTimes()
		lastWhen = time.Now()
		time.Sleep(500 * time.Millisecond)
	}
	nowCPU := readTimes()
	elapsed := time.Since(lastWhen).Seconds()
	cores := float64(runtime.NumCPU())

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

		cpuPct := 0.0
		if before, ok := lastCPU[p.Pid]; ok && elapsed > 0 {
			if used := nowCPU[p.Pid] - before; used > 0 {
				cpuPct = math.Min(100, used/(elapsed*cores)*100)
			}
		}

		items = append(items, ProcessItem{
			PID:           p.Pid,
			Name:          name,
			CPUPercent:    round1(cpuPct),
			MemoryMB:      round1(float64(memInfo.RSS) / 1024 / 1024),
			MemoryPercent: memPct,
		})
	}
	lastCPU = nowCPU
	lastWhen = time.Now()

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
