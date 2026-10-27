// Package cpuinfo reads the processor's live state for the Processor screen:
// total and per-processor use, the real current speed, and how many processes
// and threads are running. Call Sample about once a second; each call compares
// with the previous one, so the first reading may be rough.
//
// "Use" is Windows' "% Processor Utility", the figure Task Manager shows since
// Windows 11 22H2: the work done compared with what the processor does at its
// base speed. The older "% Processor Time" (share of time not idle) reads much
// lower on processors that boost far above a low base speed, so the two would
// never agree with Task Manager. Where utility is not available, time is used.
package cpuinfo

import (
	"fmt"
	"math"
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
)

type Detail struct {
	Usage   float64   `json:"usage"`    // whole processor, 0-100
	PerCore []float64 `json:"per_core"` // each logical processor, 0-100
	// the speed Windows calls the processor's base speed, and the speed it runs at now
	BaseMHz    float64 `json:"base_mhz"`
	CurrentMHz float64 `json:"current_mhz"` // 0 when Windows cannot say
	Processes  int     `json:"processes"`   // -1 when unknown
	Threads    int     `json:"threads"`     // -1 when unknown
}

const (
	utilityPath   = `\Processor Information(_Total)\% Processor Utility`
	perfPath      = `\Processor Information(_Total)\% Processor Performance`
	processesPath = `\System\Processes`
	threadsPath   = `\System\Threads`
)

// corePath is one logical processor's utility; on a one-socket PC its
// instance is "0,<number>"
func corePath(i int) string {
	return fmt.Sprintf(`\Processor Information(0,%d)\%% Processor Utility`, i)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func clamp(v float64) float64  { return math.Max(0, math.Min(100, v)) }

var (
	baseOnce sync.Once
	baseMHz  float64
)

func base() float64 {
	baseOnce.Do(func() {
		if info, err := cpu.Info(); err == nil && len(info) > 0 {
			baseMHz = info[0].Mhz
		}
	})
	return baseMHz
}

type Sampler struct {
	mu      sync.Mutex
	started bool
	logical int
	perf    *counters
}

// Sample reads the processor now
func (s *Sampler) Sample() Detail {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		s.started = true
		s.logical, _ = cpu.Counts(true)
		paths := []string{utilityPath, perfPath, processesPath, threadsPath}
		for i := 0; i < s.logical; i++ {
			paths = append(paths, corePath(i))
		}
		s.perf = openCounters(paths...)
		cpu.Percent(0, true) // prime the fallback's "since last call" reading
	}
	s.perf.collect()

	d := Detail{BaseMHz: base(), PerCore: []float64{}, Processes: -1, Threads: -1}
	timeShare, _ := cpu.Percent(0, true) // the fallback: share of time not idle
	for i := 0; i < s.logical; i++ {
		v := s.perf.value(corePath(i))
		if v < 0 && i < len(timeShare) {
			v = timeShare[i]
		}
		d.PerCore = append(d.PerCore, round1(clamp(v)))
	}
	if u := s.perf.value(utilityPath); u >= 0 {
		d.Usage = round1(clamp(u))
	} else if len(d.PerCore) > 0 {
		total := 0.0
		for _, v := range d.PerCore {
			total += v
		}
		d.Usage = round1(total / float64(len(d.PerCore)))
	}
	// "% Processor Performance" is the current speed as a share of base speed;
	// above 100 means turbo boost
	if p := s.perf.value(perfPath); p > 0 && d.BaseMHz > 0 {
		d.CurrentMHz = math.Round(d.BaseMHz * p / 100)
	}
	if v := s.perf.value(processesPath); v >= 0 {
		d.Processes = int(v)
	}
	if v := s.perf.value(threadsPath); v >= 0 {
		d.Threads = int(v)
	}
	return d
}

// Meter is the lightweight reading behind the home screen's CPU card: total use
// (as Task Manager shows it) and the current speed, since the previous Read
type Meter struct {
	mu   sync.Mutex
	open bool
	c    *counters
}

// Read returns use (0-100) and the current speed in MHz; ok is false when
// Windows' counters are not available, so the caller can fall back
func (m *Meter) Read() (usage, currentMHz float64, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.open {
		m.open = true
		m.c = openCounters(utilityPath, perfPath)
	}
	m.c.collect()
	u := m.c.value(utilityPath)
	if u < 0 {
		return 0, 0, false
	}
	if p := m.c.value(perfPath); p > 0 && base() > 0 {
		currentMHz = math.Round(base() * p / 100)
	}
	return round1(clamp(u)), currentMHz, true
}
