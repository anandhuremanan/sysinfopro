// Package pdh reads Windows performance counters (the ones Task Manager and
// Performance Monitor use) by their English names, so it works on any Windows
// language. Every method is safe on a nil *Query, which reads as "unknown".
package pdh

// Query reads a fixed set of counters. Rate counters (percentages) compare two
// readings, so call Collect, wait, then Collect again before reading them.
type Query struct {
	handle   uintptr
	counters map[string]uintptr
}

// Has reports whether Windows knows the counter at path
func (q *Query) Has(path string) bool {
	if q == nil {
		return false
	}
	_, ok := q.counters[path]
	return ok
}
