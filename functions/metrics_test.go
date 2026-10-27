package functions

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"sysinfopro/internal/cpuinfo"
)

func fakeMonitor(usage float64, mhz float64) (*Monitor, *atomic.Int64) {
	var samples atomic.Int64
	m := NewMonitor()
	m.sample = func() cpuinfo.Detail {
		samples.Add(1)
		return cpuinfo.Detail{Usage: usage, CurrentMHz: mhz, BaseMHz: 1300, PerCore: []float64{usage, usage}}
	}
	m.memory = func() (*MemoryStats, error) { return &MemoryStats{UsagePercent: 50}, nil }
	m.baseMHz = func() float64 { return 1300 }
	return m, &samples
}

// many readers at once share the one sampler's results
func TestReadersShareOneSampler(t *testing.T) {
	m, samples := fakeMonitor(42, 2600)
	m.take()
	before := samples.Load()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c := m.CPU(); c.Usage != 42 || c.FrequencyGHz != 2.6 {
				t.Errorf("cpu = %+v", c)
			}
			if d := m.Detail(); len(d.PerCore) != 2 {
				t.Errorf("detail = %+v", d)
			}
			if mem, err := m.Memory(); err != nil || mem.UsagePercent != 50 {
				t.Errorf("memory = %+v %v", mem, err)
			}
		}()
	}
	wg.Wait()
	if samples.Load() != before {
		t.Errorf("readers took %d extra samples; only the sampler should", samples.Load()-before)
	}
}

func TestSustainedLoadAndThrottling(t *testing.T) {
	high := 0
	var s CPUStats
	for i := 0; i < 4; i++ {
		s = cpuStatsFrom(cpuinfo.Detail{Usage: 95, CurrentMHz: 900}, 1300, &high)
	}
	if s.SustainedLoad {
		t.Error("4 busy readings should not count as sustained yet")
	}
	s = cpuStatsFrom(cpuinfo.Detail{Usage: 95, CurrentMHz: 900}, 1300, &high)
	if !s.SustainedLoad || !s.IsThrottling {
		t.Errorf("5 busy readings below base speed: %+v", s)
	}
	// boosting above base is not throttling
	s = cpuStatsFrom(cpuinfo.Detail{Usage: 95, CurrentMHz: 3000}, 1300, &high)
	if s.IsThrottling {
		t.Error("running above base speed reported as throttling")
	}
	// unknown live speed: never claim throttling
	s = cpuStatsFrom(cpuinfo.Detail{Usage: 95}, 1300, &high)
	if s.IsThrottling || s.FrequencyGHz != 1.3 {
		t.Errorf("unknown speed: %+v", s)
	}
}

// while hidden, no updates are sent
func TestHiddenSendsNoUpdates(t *testing.T) {
	m, _ := fakeMonitor(10, 2000)
	var updates atomic.Int64
	m.OnUpdate = func(CPUStats, MemoryStats) { updates.Add(1) }
	m.SetVisible(false)
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	m.Run(ctx)
	if updates.Load() != 0 {
		t.Errorf("%d updates sent while hidden", updates.Load())
	}
}

func TestVisibleSendsUpdates(t *testing.T) {
	m, _ := fakeMonitor(10, 2000)
	var updates atomic.Int64
	m.OnUpdate = func(CPUStats, MemoryStats) { updates.Add(1) }
	ctx, cancel := context.WithTimeout(context.Background(), 4500*time.Millisecond)
	defer cancel()
	m.Run(ctx)
	if updates.Load() < 1 {
		t.Error("no updates while visible")
	}
}
