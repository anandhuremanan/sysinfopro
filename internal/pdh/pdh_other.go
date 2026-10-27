//go:build !windows

package pdh

// Open returns nil: performance counters are a Windows feature
func Open(paths ...string) *Query { return nil }

func (q *Query) Collect() {}

func (q *Query) Value(path string) float64 { return -1 }

func (q *Query) Close() {}
