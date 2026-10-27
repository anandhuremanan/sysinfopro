//go:build windows

package slownow

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const sampleFor = 2500 * time.Millisecond

var (
	kernel32                = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemTimes      = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatus  = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetTickCount64      = kernel32.NewProc("GetTickCount64")
	procGetProcessIoCounter = kernel32.NewProc("GetProcessIoCounters")
	procGetProcessMemInfo   = kernel32.NewProc("K32GetProcessMemoryInfo")
	procGetPowerStatus      = kernel32.NewProc("GetSystemPowerStatus")

	pdh           = windows.NewLazySystemDLL("pdh.dll")
	pdhOpenQuery  = pdh.NewProc("PdhOpenQueryW")
	pdhAddEnglish = pdh.NewProc("PdhAddEnglishCounterW")
	pdhCollect    = pdh.NewProc("PdhCollectQueryData")
	pdhFormatted  = pdh.NewProc("PdhGetFormattedCounterValue")
	pdhCloseQuery = pdh.NewProc("PdhCloseQuery")
)

func ft(f windows.Filetime) uint64 { return uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime) }

func systemTimes() (idle, kernel, user uint64, ok bool) {
	var i, k, u windows.Filetime
	r, _, _ := procGetSystemTimes.Call(uintptr(unsafe.Pointer(&i)), uintptr(unsafe.Pointer(&k)), uintptr(unsafe.Pointer(&u)))
	return ft(i), ft(k), ft(u), r != 0
}

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type ioCounters struct {
	ReadOps, WriteOps, OtherOps, ReadBytes, WriteBytes, OtherBytes uint64
}

type processMemoryCountersEx struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

// --- PDH counters (performance counters Task Manager also reads) ---

type pdhQuery struct {
	handle   uintptr
	counters map[string]uintptr
}

func openPDH(paths ...string) *pdhQuery {
	q := &pdhQuery{counters: map[string]uintptr{}}
	if pdhOpenQuery.Find() != nil {
		return nil
	}
	if r, _, _ := pdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&q.handle))); r != 0 {
		return nil
	}
	for _, p := range paths {
		path, _ := windows.UTF16PtrFromString(p)
		var counter uintptr
		if r, _, _ := pdhAddEnglish.Call(q.handle, uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&counter))); r == 0 {
			q.counters[p] = counter
		}
	}
	pdhCollect.Call(q.handle)
	return q
}

func (q *pdhQuery) collect() {
	if q != nil {
		pdhCollect.Call(q.handle)
	}
}

// value reads a counter as a number; -1 when it could not be read
func (q *pdhQuery) value(path string) float64 {
	if q == nil {
		return -1
	}
	counter, ok := q.counters[path]
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

func (q *pdhQuery) close() {
	if q != nil {
		pdhCloseQuery.Call(q.handle)
	}
}

// --- processes ---

type procReading struct {
	pid    uint32
	name   string
	handle windows.Handle
	cpu    uint64 // kernel + user, 100 ns units
	io     uint64
}

func readCPUAndIO(h windows.Handle) (cpu, io uint64, ok bool) {
	var c, e, k, u windows.Filetime
	if windows.GetProcessTimes(h, &c, &e, &k, &u) != nil {
		return 0, 0, false
	}
	var counters ioCounters
	if r, _, _ := procGetProcessIoCounter.Call(uintptr(h), uintptr(unsafe.Pointer(&counters))); r != 0 {
		io = counters.ReadBytes + counters.WriteBytes
	}
	return ft(k) + ft(u), io, true
}

func privateBytes(h windows.Handle) uint64 {
	var m processMemoryCountersEx
	m.CB = uint32(unsafe.Sizeof(m))
	if r, _, _ := procGetProcessMemInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&m)), uintptr(m.CB)); r == 0 {
		return 0
	}
	// memory the program has to itself; working set undercounts programs whose memory was moved to the drive
	if m.PrivateUsage > m.WorkingSetSize {
		return uint64(m.PrivateUsage)
	}
	return uint64(m.WorkingSetSize)
}

// openProcesses lists running processes and keeps a handle to each, so a
// process that ends and whose number is reused is never mixed up
func openProcesses() []procReading {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snap)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	var out []procReading
	for err = windows.Process32First(snap, &entry); err == nil; err = windows.Process32Next(snap, &entry) {
		if entry.ProcessID == 0 {
			continue
		}
		h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
		if err != nil {
			continue // protected system processes; nothing the user could change anyway
		}
		r := procReading{pid: entry.ProcessID, name: windows.UTF16ToString(entry.ExeFile[:]), handle: h}
		var ok bool
		if r.cpu, r.io, ok = readCPUAndIO(h); !ok {
			windows.CloseHandle(h)
			continue
		}
		out = append(out, r)
	}
	return out
}

var planName = regexp.MustCompile(`\(([^)]+)\)\s*$`)

func activePowerPlan(ctx context.Context) string {
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, "powercfg", "/getactivescheme")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	if m := planName.FindStringSubmatch(strings.TrimSpace(string(out))); m != nil {
		return m[1]
	}
	return ""
}

func rebootPending() bool {
	for _, path := range []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending`,
	} {
		if k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE); err == nil {
			k.Close()
			return true
		}
	}
	return false
}

const (
	cpuPerfCounter = `\Processor Information(_Total)\% Processor Performance`
	// what Task Manager shows as processor use (Windows 11 22H2 and later)
	cpuUtility = `\Processor Information(_Total)\% Processor Utility`
	diskIdle   = `\PhysicalDisk(_Total)\% Idle Time`
)

func collect(ctx context.Context) Sample {
	began := time.Now()
	s := Sample{PerformancePct: -1, DiskBusyPct: -1}

	plan := make(chan string, 1)
	go func() { plan <- activePowerPlan(ctx) }()

	q := openPDH(cpuPerfCounter, cpuUtility, diskIdle)
	defer q.close()
	idle1, kernel1, user1, sysOK := systemTimes()
	procs := openProcesses()
	defer func() {
		for _, p := range procs {
			windows.CloseHandle(p.handle)
		}
	}()
	start := time.Now()

	select {
	case <-ctx.Done():
	case <-time.After(sampleFor):
	}

	elapsed := time.Since(start)
	q.collect()
	idle2, kernel2, user2, sysOK2 := systemTimes()
	if sysOK && sysOK2 {
		// kernel time includes idle time
		total := (kernel2 - kernel1) + (user2 - user1)
		if total > 0 {
			s.CPUPct = 100 * float64(total-(idle2-idle1)) / float64(total)
		}
	}
	cores := float64(runtime.NumCPU())
	ticks := float64(elapsed.Nanoseconds()) / 100 // 100 ns units
	for _, p := range procs {
		cpu, io, ok := readCPUAndIO(p.handle)
		if !ok {
			continue
		}
		pr := Proc{PID: p.pid, Name: p.name, Count: 1, MemoryBytes: privateBytes(p.handle)}
		if cpu >= p.cpu && ticks > 0 {
			pr.CPUPct = 100 * float64(cpu-p.cpu) / (ticks * cores)
		}
		if io >= p.io && elapsed > 0 {
			pr.IOBytesPerSec = float64(io-p.io) / elapsed.Seconds()
		}
		s.Procs = append(s.Procs, pr)
	}
	if v := q.value(cpuPerfCounter); v >= 0 {
		s.PerformancePct = v
	}
	// report use as Task Manager does, and scale each program's share the same
	// way, so the numbers here match what people see there
	if u := q.value(cpuUtility); u >= 0 {
		s.CPUTimePct = s.CPUPct
		if s.CPUPct > 0.5 {
			ratio := u / s.CPUPct
			for i := range s.Procs {
				s.Procs[i].CPUPct = min(100, s.Procs[i].CPUPct*ratio)
			}
		}
		s.CPUPct = max(0, min(100, u))
	}
	if v := q.value(diskIdle); v >= 0 {
		s.DiskBusyPct = max(0, min(100, 100-v))
	}

	var m memoryStatusEx
	m.Length = uint32(unsafe.Sizeof(m))
	if r, _, _ := procGlobalMemoryStatus.Call(uintptr(unsafe.Pointer(&m))); r != 0 {
		s.MemTotal, s.MemAvailable = m.TotalPhys, m.AvailPhys
	}

	var ps systemPowerStatus
	if r, _, _ := procGetPowerStatus.Call(uintptr(unsafe.Pointer(&ps))); r != 0 {
		s.OnBattery = ps.ACLineStatus == 0
		s.BatterySaver = ps.SystemStatusFlag == 1
	}

	ms, _, _ := procGetTickCount64.Call()
	s.UptimeHours = float64(ms) / 3_600_000
	s.RebootPending = rebootPending()

	s.SystemDrive = os.Getenv("SystemDrive")
	if s.SystemDrive == "" {
		s.SystemDrive = "C:"
	}
	root, _ := windows.UTF16PtrFromString(s.SystemDrive + `\`)
	var avail, total, free uint64
	if windows.GetDiskFreeSpaceEx(root, &avail, &total, &free) == nil {
		s.SystemFree, s.SystemTotal = avail, total
	}

	select {
	case s.PowerPlan = <-plan:
	case <-time.After(time.Second):
	}
	s.SampledMs = time.Since(began).Milliseconds()
	return s
}
