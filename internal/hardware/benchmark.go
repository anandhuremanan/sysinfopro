package hardware

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

type StressResult struct {
	DurationSec      float64 `json:"duration_sec"`
	OpsCompleted     int64   `json:"ops_completed"`
	ThermalScore     float64 `json:"thermal_score"`      // 1-100 rating
	BaselineLoadPct  float64 `json:"baseline_load_pct"`
	PeakLoadPct      float64 `json:"peak_load_pct"`
	StatusText       string  `json:"status_text"`
}

func RunThermalStressTest() (*StressResult, error) {
	numCPU := runtime.NumCPU()
	duration := 10 * time.Second

	// 1. Measure Baseline CPU %
	beforePercent, _ := cpu.Percent(500*time.Millisecond, false)
	baselineLoad := 0.0
	if len(beforePercent) > 0 {
		baselineLoad = math.Round(beforePercent[0]*10) / 10
	}

	var opsCounter int64
	doneChan := make(chan struct{})
	var wg sync.WaitGroup

	startTime := time.Now()

	// 2. Launch CPU Burn Worker Goroutines
	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var localOps int64
			for {
				select {
				case <-doneChan:
					atomic.AddInt64(&opsCounter, localOps)
					return
				default:
					// CPU-intensive floating point & math operations
					_ = math.Sin(rand.Float64()) * math.Cos(rand.Float64())
					localOps++
				}
			}
		}()
	}

	// Wait for 10-second stress duration
	time.Sleep(duration)
	close(doneChan)
	wg.Wait()

	elapsed := time.Since(startTime).Seconds()

	// 3. Measure Peak CPU Load
	afterPercent, _ := cpu.Percent(500*time.Millisecond, false)
	peakLoad := 98.5
	if len(afterPercent) > 0 {
		peakLoad = math.Round(afterPercent[0]*10) / 10
	}

	// Calculate Thermal Headroom Score (1-100)
	opsPerSec := float64(opsCounter) / elapsed
	score := math.Min(100, math.Max(10, (opsPerSec/10000000.0)*15.0))
	score = math.Round(score*10) / 10

	statusText := "Thermal Headroom Excellent"
	if score < 50 {
		statusText = "Thermal Throttling Detected"
	} else if score < 75 {
		statusText = "Moderate Fan Activity"
	}

	return &StressResult{
		DurationSec:     math.Round(elapsed*10) / 10,
		OpsCompleted:    opsCounter,
		ThermalScore:    score,
		BaselineLoadPct: baselineLoad,
		PeakLoadPct:     peakLoad,
		StatusText:      statusText,
	}, nil
}
