// Package cpuinfo reads the processor's live state for the Processor screen:
// total and per-processor use, the real current speed, and how many processes
// and threads are running. Call Sample about once a second; each call compares
// with the previous one, so the first reading may be rough.
package cpuinfo

import (
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

type Sampler struct {
	mu      sync.Mutex
	baseMHz float64
	started bool
	perf    *counters
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// Sample reads the processor now
func (s *Sampler) Sample() Detail {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		s.started = true
		if info, err := cpu.Info(); err == nil && len(info) > 0 {
			s.baseMHz = info[0].Mhz
		}
		s.perf = openCounters()
		// prime the "since last call" readings
		cpu.Percent(0, true)
	}

	d := Detail{BaseMHz: s.baseMHz, PerCore: []float64{}, Processes: -1, Threads: -1}
	if per, err := cpu.Percent(0, true); err == nil && len(per) > 0 {
		total := 0.0
		for _, p := range per {
			p = math.Max(0, math.Min(100, p))
			d.PerCore = append(d.PerCore, round1(p))
			total += p
		}
		d.Usage = round1(total / float64(len(per)))
	}
	perfPct, processes, threads := s.perf.read()
	// "% Processor Performance" is the current speed as a share of base speed;
	// above 100 means turbo boost
	if perfPct > 0 && s.baseMHz > 0 {
		d.CurrentMHz = math.Round(s.baseMHz * perfPct / 100)
	}
	if processes >= 0 {
		d.Processes = int(processes)
	}
	if threads >= 0 {
		d.Threads = int(threads)
	}
	return d
}
