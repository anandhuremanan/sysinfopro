package functions

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"sysinfopro/internal/cpuinfo"
)

// Monitor is the one owner of live processor and memory readings. A single
// goroutine (Run) samples on a fixed beat and keeps the latest results; every
// reader (the home screen events, the Processor screen, the diagnostic report)
// reads those, so extra callers never shorten the measuring window or skew the
// "busy for a while" count. While the window is hidden it samples rarely and
// sends no events, so a minimised app costs next to nothing.
type Monitor struct {
	// OnUpdate is called from the sampling goroutine with fresh readings, about
	// every 2 seconds while the window is visible, never while it is hidden
	OnUpdate func(cpu CPUStats, mem MemoryStats)

	sample  func() cpuinfo.Detail // the processor reading; replaced in tests
	memory  func() (*MemoryStats, error)
	baseMHz func() float64

	mu        sync.Mutex
	detail    cpuinfo.Detail
	cpu       *CPUStats
	mem       *MemoryStats
	highCount int // consecutive busy readings
	visible   bool

	ready     chan struct{} // closed after the first reading
	readyOnce sync.Once
	wake      chan struct{} // nudges the loop when visibility changes
}

const (
	visibleEvery = time.Second      // sampling beat while the window is shown
	hiddenEvery  = 15 * time.Second // while hidden: enough to stay roughly current
	eventEvery   = 2                // send events every this many visible beats
)

// NewMonitor prepares a monitor; call Run to start it
func NewMonitor() *Monitor {
	var sampler cpuinfo.Sampler
	return &Monitor{
		sample:  sampler.Sample,
		memory:  GetMemoryStats,
		baseMHz: cpuinfo.BaseMHz,
		visible: true,
		ready:   make(chan struct{}),
		wake:    make(chan struct{}, 1),
	}
}

// SetVisible slows sampling and stops events while the window is hidden
func (m *Monitor) SetVisible(visible bool) {
	m.mu.Lock()
	changed := m.visible != visible
	m.visible = visible
	m.mu.Unlock()
	if changed {
		select {
		case m.wake <- struct{}{}:
		default:
		}
	}
}

// Run samples until ctx ends
func (m *Monitor) Run(ctx context.Context) {
	// rate counters compare two readings: take a first one, then wait a beat
	m.sample()
	beat := 0
	for {
		m.mu.Lock()
		visible := m.visible
		m.mu.Unlock()
		every := hiddenEvery
		if visible {
			every = visibleEvery
		}
		select {
		case <-ctx.Done():
			return
		case <-m.wake:
			continue
		case <-time.After(every):
		}

		cpu, mem := m.take()
		beat++
		if visible && beat%eventEvery == 0 && m.OnUpdate != nil && cpu != nil && mem != nil {
			m.OnUpdate(*cpu, *mem)
		}
	}
}

// take reads both and stores the results
func (m *Monitor) take() (*CPUStats, *MemoryStats) {
	d := m.sample()
	mem, err := m.memory()
	if err != nil {
		slog.Warn("memory reading failed", "err", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.detail = d
	cpu := cpuStatsFrom(d, m.baseMHz(), &m.highCount)
	m.cpu = &cpu
	if mem != nil {
		m.mem = mem
	}
	m.readyOnce.Do(func() { close(m.ready) })
	return m.cpu, m.mem
}

// waitReady waits briefly for the first reading, right after the app starts
func (m *Monitor) waitReady() {
	select {
	case <-m.ready:
	case <-time.After(3 * time.Second):
	}
}

// CPU is the latest processor reading, as Task Manager counts use
func (m *Monitor) CPU() *CPUStats {
	m.waitReady()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cpu == nil {
		return &CPUStats{Timestamp: time.Now().Unix()}
	}
	c := *m.cpu
	return &c
}

// Detail is the latest full processor reading for the Processor screen
func (m *Monitor) Detail() cpuinfo.Detail {
	m.waitReady()
	m.mu.Lock()
	defer m.mu.Unlock()
	d := m.detail
	d.PerCore = append([]float64(nil), d.PerCore...)
	if d.PerCore == nil {
		d.PerCore = []float64{}
	}
	return d
}

// Memory is the latest memory reading; read directly when there is none yet
func (m *Monitor) Memory() (*MemoryStats, error) {
	m.mu.Lock()
	mem := m.mem
	m.mu.Unlock()
	if mem == nil {
		return m.memory()
	}
	c := *mem
	return &c, nil
}

// cpuStatsFrom turns a processor reading into the home screen's summary.
// highCount carries the run of busy readings between calls.
func cpuStatsFrom(d cpuinfo.Detail, baseMHz float64, highCount *int) CPUStats {
	mhz := d.CurrentMHz
	if mhz <= 0 {
		mhz = baseMHz
	}
	// busy for several readings in a row
	if d.Usage > 80 {
		*highCount++
	} else if *highCount > 0 {
		*highCount--
	}
	sustained := *highCount >= 5 // about 5 seconds while the window is shown
	return CPUStats{
		Usage:         d.Usage,
		Timestamp:     time.Now().Unix(),
		FrequencyGHz:  math.Round(mhz/10) / 100,
		SustainedLoad: sustained,
		// slowed down: busy, yet below its own base speed (heat or power saving)
		IsThrottling: sustained && d.CurrentMHz > 0 && baseMHz > 0 && d.CurrentMHz < baseMHz*0.9,
	}
}
