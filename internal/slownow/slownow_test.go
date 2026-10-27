package slownow

import (
	"strings"
	"testing"
)

// a quiet PC with plenty of everything
func calm() Sample {
	return Sample{
		CPUPct: 6, PerformancePct: 95, DiskBusyPct: 3,
		MemTotal: 16 * gb, MemAvailable: 9 * gb,
		SystemDrive: "C:", SystemFree: 200 * gb, SystemTotal: 500 * gb,
		UptimeHours: 30,
		Procs: []Proc{
			{PID: 10, Name: "chrome.exe", CPUPct: 2, MemoryBytes: 900 * mb, Count: 1},
			{PID: 11, Name: "chrome.exe", CPUPct: 1, MemoryBytes: 400 * mb, Count: 1},
			{PID: 12, Name: "Code.exe", CPUPct: 1, MemoryBytes: 600 * mb, Count: 1},
		},
	}
}

func find(r Report, id string) *Finding {
	for i := range r.Findings {
		if r.Findings[i].ID == id {
			return &r.Findings[i]
		}
	}
	return nil
}

func TestCalm(t *testing.T) {
	r := analyse(calm())
	if !r.Calm || len(r.Findings) != 0 || !strings.Contains(r.Summary, "Nothing") {
		t.Errorf("got %+v", r)
	}
	// browser processes are added together, and the biggest one is named
	if r.Memory.Top[0].Name != "chrome.exe" || r.Memory.Top[0].Count != 2 || r.Memory.Top[0].PID != 10 {
		t.Errorf("top memory = %+v", r.Memory.Top[0])
	}
	if r.Memory.UsedPct != 43.8 {
		t.Errorf("used = %v", r.Memory.UsedPct)
	}
}

func TestDefenderScan(t *testing.T) {
	s := calm()
	s.CPUPct, s.DiskBusyPct = 55, 95
	s.Procs = append(s.Procs, Proc{PID: 99, Name: "MsMpEng.exe", CPUPct: 30, IOBytesPerSec: 80 * mb, Count: 1})
	r := analyse(s)
	f := find(r, "defender")
	if f == nil || f.Level != "high" || f.PID != 99 || !strings.Contains(f.Detail, "80 MB/s") {
		t.Fatalf("defender = %+v", f)
	}
	// the processor and drive findings name it too, but do not tell the user to close Windows' antivirus
	if c := find(r, "cpu"); c == nil || strings.Contains(c.Fix, "close it") {
		t.Errorf("cpu = %+v", c)
	}
	if d := find(r, "disk"); d == nil || d.Level != "high" || d.Fix != "" {
		t.Errorf("disk = %+v", d)
	}
	if r.Calm || r.Findings[0].Level != "high" {
		t.Errorf("summary = %q", r.Summary)
	}
}

func TestWSLMemory(t *testing.T) {
	s := calm()
	s.MemAvailable = 2 * gb
	s.Procs = append(s.Procs, Proc{PID: 7, Name: "VmmemWSL", MemoryBytes: 8 * gb, Count: 1})
	r := analyse(s)
	w := find(r, "wsl-memory")
	if w == nil || w.Level != "high" || !strings.Contains(w.Title, "8.0 GB") || !strings.Contains(w.Fix, "wsl --shutdown") {
		t.Fatalf("wsl = %+v", w)
	}
	m := find(r, "memory")
	if m == nil || m.Level != "medium" || !strings.Contains(m.Detail, "VmmemWSL 8.0 GB") {
		t.Errorf("memory = %+v", m)
	}
	// an idle WSL holding memory on a PC with room to spare is only worth a note
	s = calm()
	s.Procs = append(s.Procs, Proc{PID: 7, Name: "vmmem", MemoryBytes: 3 * gb, Count: 1})
	if w := find(analyse(s), "wsl-memory"); w == nil || w.Level != "info" {
		t.Errorf("idle wsl = %+v", w)
	}
}

func TestThrottled(t *testing.T) {
	s := calm()
	s.CPUPct, s.PerformancePct = 70, 45
	r := analyse(s)
	f := find(r, "throttled")
	if f == nil || !strings.Contains(f.Detail, "too hot") {
		t.Fatalf("hot = %+v", f)
	}
	s.OnBattery, s.BatterySaver = true, true
	if f := find(analyse(s), "throttled"); f == nil || !strings.Contains(f.Detail, "Battery saver") {
		t.Errorf("saver = %+v", f)
	}
	// slow but idle is just power saving at rest, not a problem
	s = calm()
	s.PerformancePct = 40
	if f := find(analyse(s), "throttled"); f != nil {
		t.Errorf("idle = %+v", f)
	}
}

func TestOneAppHogging(t *testing.T) {
	s := calm()
	s.CPUPct = 35
	s.Procs = append(s.Procs, Proc{PID: 50, Name: "node.exe", CPUPct: 30, Count: 1})
	f := find(analyse(s), "cpu-app")
	if f == nil || f.Process != "node.exe" || f.PID != 50 {
		t.Errorf("got %+v", f)
	}
}

func TestHousekeeping(t *testing.T) {
	s := calm()
	s.RebootPending = true
	s.UptimeHours = 20 * 24
	s.SystemFree = 10 * gb // 2% of 500 GB
	s.PowerPlan = "Power saver"
	r := analyse(s)
	for _, id := range []string{"restart", "uptime", "drive-full", "power-plan"} {
		if find(r, id) == nil {
			t.Errorf("missing %s: %+v", id, r.Findings)
		}
	}
	if f := find(r, "drive-full"); f.Level != "high" {
		t.Errorf("drive = %+v", f)
	}
	if r.Findings[len(r.Findings)-1].ID != "uptime" {
		t.Error("notes come last")
	}
}

func TestUpdateGroupedOnce(t *testing.T) {
	s := calm()
	s.Procs = append(s.Procs,
		Proc{PID: 1, Name: "TiWorker.exe", CPUPct: 12, Count: 1},
		Proc{PID: 2, Name: "TrustedInstaller.exe", CPUPct: 8, Count: 1})
	r := analyse(s)
	n := 0
	for _, f := range r.Findings {
		if f.ID == "update" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("Windows Update explained %d times", n)
	}
}
