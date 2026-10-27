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

const (
	perfPath      = `\Processor Information(_Total)\% Processor Performance`
	processesPath = `\System\Processes`
	threadsPath   = `\System\Threads`
)

type counters struct {
	query    uintptr
	counters map[string]uintptr
}

func openCounters() *counters {
	if pdhOpenQuery.Find() != nil {
		return nil
	}
	c := &counters{counters: map[string]uintptr{}}
	if r, _, _ := pdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&c.query))); r != 0 {
		return nil
	}
	for _, p := range []string{perfPath, processesPath, threadsPath} {
		path, _ := windows.UTF16PtrFromString(p)
		var counter uintptr
		if r, _, _ := pdhAddEnglish.Call(c.query, uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&counter))); r == 0 {
			c.counters[p] = counter
		}
	}
	pdhCollect.Call(c.query)
	return c
}

func (c *counters) value(path string) float64 {
	counter, ok := c.counters[path]
	if !ok {
		return -1
	}
	const pdhFmtDouble = 0x00000200
	var v struct {
		Status uint32
		_      uint32
		Value  float64
	}
	if r, _, _ := pdhFormatted.Call(counter, pdhFmtDouble, 0, uintptr(unsafe.Pointer(&v))); r != 0 || v.Status > 1 {
		return -1
	}
	return v.Value
}

// read collects a new sample: performance %, processes, threads (-1 when unknown)
func (c *counters) read() (perf, processes, threads float64) {
	if c == nil {
		return -1, -1, -1
	}
	pdhCollect.Call(c.query)
	return c.value(perfPath), c.value(processesPath), c.value(threadsPath)
}
