//go:build !windows

package cpuinfo

type counters struct{}

func openCounters(paths ...string) *counters { return nil }

func (c *counters) collect() {}

func (c *counters) value(path string) float64 { return -1 }
