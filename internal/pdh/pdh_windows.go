//go:build windows

package pdh

import (
	"log/slog"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	dll           = windows.NewLazySystemDLL("pdh.dll")
	pdhOpenQuery  = dll.NewProc("PdhOpenQueryW")
	pdhAddEnglish = dll.NewProc("PdhAddEnglishCounterW")
	pdhCollect    = dll.NewProc("PdhCollectQueryData")
	pdhFormatted  = dll.NewProc("PdhGetFormattedCounterValue")
	pdhCloseQuery = dll.NewProc("PdhCloseQuery")
)

const (
	fmtDouble   = 0x00000200
	fmtNoCap100 = 0x00008000 // some counters pass 100% (turbo boost); callers cap if they need to
)

// Open starts reading the counters at paths and takes a first reading. Counters
// this Windows does not have are left out (and read as -1). It returns nil when
// performance counters are unavailable altogether.
func Open(paths ...string) *Query {
	if pdhOpenQuery.Find() != nil {
		return nil
	}
	q := &Query{counters: map[string]uintptr{}}
	if r, _, _ := pdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&q.handle))); r != 0 {
		slog.Debug("performance counters unavailable", "status", r)
		return nil
	}
	for _, p := range paths {
		path, err := windows.UTF16PtrFromString(p)
		if err != nil {
			continue
		}
		var counter uintptr
		if r, _, _ := pdhAddEnglish.Call(q.handle, uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&counter))); r == 0 {
			q.counters[p] = counter
		} else {
			slog.Debug("performance counter not available", "counter", p, "status", r)
		}
	}
	pdhCollect.Call(q.handle)
	return q
}

// Collect takes a new reading of every counter
func (q *Query) Collect() {
	if q != nil {
		pdhCollect.Call(q.handle)
	}
}

// Value is a counter's latest reading, not capped at 100; -1 when unknown
func (q *Query) Value(path string) float64 {
	if q == nil {
		return -1
	}
	counter, ok := q.counters[path]
	if !ok {
		return -1
	}
	var v struct {
		Status uint32
		_      uint32
		Value  float64
	}
	if r, _, _ := pdhFormatted.Call(counter, fmtDouble|fmtNoCap100, 0, uintptr(unsafe.Pointer(&v))); r != 0 || v.Status > 1 {
		return -1
	}
	return v.Value
}

// Close releases the query
func (q *Query) Close() {
	if q != nil && q.handle != 0 {
		pdhCloseQuery.Call(q.handle)
		q.handle = 0
		q.counters = nil
	}
}
