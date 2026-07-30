package functions

import (
	"context"
	"math"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	recentHighLoadCount = 0
)

// GetCPUStats returns the current CPU utilization percentage and frequency metrics
func GetCPUStats() (*CPUStats, error) {
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, err
	}

	usage := percentages[0]

	// Sample CPU clock frequency
	freqGHz := 2.5
	info, err := cpu.Info()
	if err == nil && len(info) > 0 && info[0].Mhz > 0 {
		freqGHz = math.Round((info[0].Mhz/1000.0)*100) / 100
	}

	// Detect sustained load (e.g. usage > 80% for multiple samples)
	if usage > 80.0 {
		recentHighLoadCount++
	} else {
		if recentHighLoadCount > 0 {
			recentHighLoadCount--
		}
	}

	sustained := recentHighLoadCount >= 3
	isThrottling := sustained && freqGHz < 2.0

	return &CPUStats{
		Usage:         usage,
		Timestamp:     time.Now().Unix(),
		FrequencyGHz:  freqGHz,
		IsThrottling:  isThrottling,
		SustainedLoad: sustained,
	}, nil
}

// StartCPUMonitoring begins periodic monitoring of CPU usage and emits events
func StartCPUMonitoring(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)

	go func() {
		defer ticker.Stop()

		for range ticker.C {
			stats, err := GetCPUStats()
			if err == nil {
				runtime.EventsEmit(ctx, "cpu-stats-update", stats)
			}
		}
	}()
}
