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

// StressResult is what a stress test measured. It makes no claims about
// temperature: nothing here reads a thermal sensor.
type StressResult struct {
	DurationSec     float64 `json:"duration_sec"`
	OpsCompleted    int64   `json:"ops_completed"`
	BaselineLoadPct float64 `json:"baseline_load_pct"`
	// how busy the processor was while the test ran; -1 when it could not be read
	PeakLoadPct float64 `json:"peak_load_pct"`
}

func RunThermalStressTest() (*StressResult, error) {
	numCPU := runtime.NumCPU()
	duration := 10 * time.Second

	// 1. How busy the processor was before the test
	beforePercent, _ := cpu.Percent(500*time.Millisecond, false)
	baselineLoad := 0.0
	if len(beforePercent) > 0 {
		baselineLoad = math.Round(beforePercent[0]*10) / 10
	}

	var opsCounter int64
	doneChan := make(chan struct{})
	var wg sync.WaitGroup

	startTime := time.Now()

	// 2. One worker per logical processor
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
					_ = math.Sin(rand.Float64()) * math.Cos(rand.Float64())
					localOps++
				}
			}
		}()
	}

	// 3. How busy the processor is while the workers run (measured mid-test,
	//    not after they stop)
	peakLoad := -1.0
	peakDone := make(chan struct{})
	go func() {
		defer close(peakDone)
		time.Sleep(3 * time.Second)
		if during, err := cpu.Percent(2*time.Second, false); err == nil && len(during) > 0 {
			peakLoad = math.Round(during[0]*10) / 10
		}
	}()

	time.Sleep(duration)
	close(doneChan)
	wg.Wait()
	<-peakDone

	elapsed := time.Since(startTime).Seconds()

	return &StressResult{
		DurationSec:     math.Round(elapsed*10) / 10,
		OpsCompleted:    opsCounter,
		BaselineLoadPct: baselineLoad,
		PeakLoadPct:     peakLoad,
	}, nil
}
