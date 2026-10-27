package functions

import (
	"context"
	"math"
	"time"

	"myproject/internal/cpuinfo"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	recentHighLoadCount = 0

	// Task Manager's own "use" and the live speed, read from Windows' counters
	meter  cpuinfo.Meter
	primed bool
)

// GetCPUStats returns processor use, as Task Manager shows it, and the live speed
func GetCPUStats() (*CPUStats, error) {
	if !primed {
		// the counters compare with the previous reading, so take a first one
		meter.Read()
		time.Sleep(time.Second)
		primed = true
	}
	usage, currentMHz, ok := meter.Read()

	baseMHz := 0.0
	if info, err := cpu.Info(); err == nil && len(info) > 0 {
		baseMHz = info[0].Mhz
	}
	if !ok {
		// counters unavailable: share of time not idle, over a second
		percentages, err := cpu.Percent(time.Second, false)
		if err != nil {
			return nil, err
		}
		usage = percentages[0]
	}
	mhz := currentMHz
	if mhz <= 0 {
		mhz = baseMHz
	}
	freqGHz := math.Round(mhz/10) / 100

	// busy for several readings in a row
	if usage > 80.0 {
		recentHighLoadCount++
	} else if recentHighLoadCount > 0 {
		recentHighLoadCount--
	}
	sustained := recentHighLoadCount >= 3
	// slowed down: busy, yet running below its own base speed (heat or power saving)
	isThrottling := sustained && currentMHz > 0 && baseMHz > 0 && currentMHz < baseMHz*0.9

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
