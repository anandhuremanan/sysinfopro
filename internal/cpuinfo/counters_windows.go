//go:build windows

package cpuinfo

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows performance counters, the same ones Task Manager reads
var (
	pdh           = windows.NewLazySystemDLL("pdh.dll")
	pdhOpenQuery  = pdh.NewProc("PdhOpenQueryW")
	pdhAddEnglish = pdh.NewProc("PdhAddEnglishCounterW")
	pdhCollect    = pdh.NewProc("PdhCollectQueryData")
	pdhFormatted  = pdh.NewProc("PdhGetFormattedCounterValue")
)

type counters struct {
	query    uintptr
	counters map[string]uintptr
}

// openCounters starts reading the given counters; counters Windows does not
// have are skipped (and read as -1)
func openCounters(paths ...string) *counters {
	if pdhOpenQuery.Find() != nil {
		return nil
	}
	c := &counters{counters: map[string]uintptr{}}
	if r, _, _ := pdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&c.query))); r != 0 {
		return nil
	}
	for _, p := range paths {
		path, _ := windows.UTF16PtrFromString(p)
		var counter uintptr
		if r, _, _ := pdhAddEnglish.Call(c.query, uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&counter))); r == 0 {
			c.counters[p] = counter
		}
	}
	pdhCollect.Call(c.query)
	return c
}

// collect takes a new reading of every counter
func (c *counters) collect() {
	if c != nil {
		pdhCollect.Call(c.query)
	}
}

// value is a counter's latest reading; -1 when unknown
func (c *counters) value(path string) float64 {
	if c == nil {
		return -1
	}
	counter, ok := c.counters[path]
	if !ok {
		return -1
	}
	const pdhFmtDouble = 0x00000200
	const pdhFmtNoCap100 = 0x00008000 // utility passes 100% while boosting; capped later
	var v struct {
		Status uint32
		_      uint32
		Value  float64
	}
	if r, _, _ := pdhFormatted.Call(counter, pdhFmtDouble|pdhFmtNoCap100, 0, uintptr(unsafe.Pointer(&v))); r != 0 || v.Status > 1 {
		return -1
	}
	return v.Value
}
