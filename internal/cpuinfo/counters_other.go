//go:build !windows

package cpuinfo

type counters struct{}

func openCounters() *counters { return nil }

func (c *counters) read() (perf, processes, threads float64) { return -1, -1, -1 }
