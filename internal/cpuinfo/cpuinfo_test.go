//go:build integration

package cpuinfo

import (
	"testing"
	"time"
)

func TestSample(t *testing.T) {
	var s Sampler
	s.Sample()
	time.Sleep(1100 * time.Millisecond)
	d := s.Sample()
	if len(d.PerCore) == 0 || d.Usage < 0 || d.Usage > 100 {
		t.Fatalf("got %+v", d)
	}
	for _, p := range d.PerCore {
		if p < 0 || p > 100 {
			t.Errorf("core reading out of range: %v", p)
		}
	}
	t.Logf("usage %.1f%%, %d logical processors, base %.0f MHz, now %.0f MHz, %d processes, %d threads",
		d.Usage, len(d.PerCore), d.BaseMHz, d.CurrentMHz, d.Processes, d.Threads)
}
