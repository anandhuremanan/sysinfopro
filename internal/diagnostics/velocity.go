package diagnostics

import (
	"math"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

type MemoryVelocitySample struct {
	Timestamp       string  `json:"timestamp"`
	UsedMB          float64 `json:"used_mb"`
	VelocityMBps    float64 `json:"velocity_mbps"`
	StatusText      string  `json:"status_text"`
	EstimatedOOMSec float64 `json:"estimated_oom_sec"`
}

type VelocityTracker struct {
	mu           sync.Mutex
	lastTime     time.Time
	lastUsedMB   float64
	lastVelocity float64
}

var globalTracker = &VelocityTracker{}

func GetMemoryVelocity() (*MemoryVelocitySample, error) {
	globalTracker.mu.Lock()
	defer globalTracker.mu.Unlock()

	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	usedMB := float64(v.Total-v.Free) / 1024 / 1024

	sample := &MemoryVelocitySample{
		Timestamp:       now.Format(time.RFC3339),
		UsedMB:          usedMB,
		VelocityMBps:    0,
		StatusText:      "Memory Stable",
		EstimatedOOMSec: -1,
	}

	if !globalTracker.lastTime.IsZero() {
		dt := now.Sub(globalTracker.lastTime).Seconds()
		if dt > 0.5 {
			dUsed := usedMB - globalTracker.lastUsedMB
			velocity := dUsed / dt

			// Exponential smoothing
			smoothedVelocity := 0.7*velocity + 0.3*globalTracker.lastVelocity
			sample.VelocityMBps = math.Round(smoothedVelocity*10) / 10
			globalTracker.lastVelocity = smoothedVelocity

			// no "runs out in N seconds" forecast: a few seconds of growth say
			// little about the next minute, so EstimatedOOMSec stays -1
			if smoothedVelocity > 150 {
				sample.StatusText = "Rapid Memory Spike"
			} else if smoothedVelocity > 50 {
				sample.StatusText = "Memory Climbing"
			} else if smoothedVelocity < -50 {
				sample.StatusText = "Memory Reclaiming"
			} else {
				sample.StatusText = "Memory Stable"
			}
		}
	}

	globalTracker.lastTime = now
	globalTracker.lastUsedMB = usedMB

	return sample, nil
}
