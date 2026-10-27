// Package slownow answers "why is this PC slow right now?". It samples the
// processor, memory and drive for a couple of seconds, then explains in plain
// words what is using them and what to try.
package slownow

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Proc is one program; processes with the same exe name are added together
// (a browser runs dozens)
type Proc struct {
	PID           uint32  `json:"pid"` // the biggest process of the group
	Name          string  `json:"name"`
	CPUPct        float64 `json:"cpu_pct"` // share of the whole processor, 0-100
	MemoryBytes   uint64  `json:"memory_bytes"`
	IOBytesPerSec float64 `json:"io_bytes_per_sec"` // reads and writes
	Count         int     `json:"count"`
}

type Finding struct {
	ID      string `json:"id"`
	Level   string `json:"level"` // high | medium | info
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Fix     string `json:"fix"`
	PID     uint32 `json:"pid"`     // 0 when not about one program
	Process string `json:"process"` // exe name, or ""
}

type CPUInfo struct {
	Percent float64 `json:"percent"`
	// speed compared with the processor's normal speed; -1 when unknown
	PerformancePct float64 `json:"performance_pct"`
	Top            []Proc  `json:"top"`
}

type MemoryInfo struct {
	UsedPct        float64 `json:"used_pct"`
	TotalBytes     uint64  `json:"total_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	Top            []Proc  `json:"top"`
}

type DiskInfo struct {
	BusyPct float64 `json:"busy_pct"` // -1 when unknown
	Top     []Proc  `json:"top"`
}

type Report struct {
	Findings  []Finding  `json:"findings"`
	CPU       CPUInfo    `json:"cpu"`
	Memory    MemoryInfo `json:"memory"`
	Disk      DiskInfo   `json:"disk"`
	Summary   string     `json:"summary"`
	Calm      bool       `json:"calm"`
	SampledMs int64      `json:"sampled_ms"`
}

// Sample is what was measured; analyse turns it into a Report
type Sample struct {
	CPUPct float64 // as Task Manager shows it (% Processor Utility)
	// share of time the processor was not idle; 0 when only CPUPct is known.
	// Utility shrinks when the processor slows down, so "is it busy?" for the
	// slowed-down check is answered from this instead.
	CPUTimePct     float64
	PerformancePct float64 // -1 unknown
	Procs          []Proc  // one per process, Count 1
	MemTotal       uint64
	MemAvailable   uint64
	DiskBusyPct    float64 // -1 unknown
	OnBattery      bool
	BatterySaver   bool
	PowerPlan      string // "" unknown
	RebootPending  bool
	UptimeHours    float64
	SystemDrive    string
	SystemFree     uint64
	SystemTotal    uint64
	SampledMs      int64
}

// Diagnose samples the PC for about 2.5 seconds
func Diagnose(ctx context.Context) Report {
	return analyse(collect(ctx))
}

const (
	gb = 1 << 30
	mb = 1 << 20
)

// programs that are often behind a slow PC, and what they are doing
type culprit struct {
	id, title, detail, fix string
}

var culprits = map[string]culprit{
	"msmpeng.exe": {"defender", "Microsoft Defender is scanning",
		"Windows' antivirus is checking files. While it scans, the drive and processor are busy.",
		"It stops on its own. Scans run when the PC is idle, so leaving it on and plugged in now and then keeps them out of your way."},
	"tiworker.exe":           updateCulprit,
	"trustedinstaller.exe":   updateCulprit,
	"mousocoreworker.exe":    updateCulprit,
	"wuauclt.exe":            updateCulprit,
	"usoclient.exe":          updateCulprit,
	"searchindexer.exe":      indexCulprit,
	"searchprotocolhost.exe": indexCulprit,
	"searchfilterhost.exe":   indexCulprit,
	"onedrive.exe": {"onedrive", "OneDrive is syncing",
		"Files are being uploaded or downloaded, which keeps the drive and network busy.",
		"Pause syncing for 2 hours from the OneDrive icon in the taskbar."},
	"compattelrunner.exe": {"telemetry", "Windows is checking app compatibility",
		"A Windows task that collects compatibility information is running.",
		"It runs now and then and finishes on its own."},
	"wmiprvse.exe": {"wmi", "A program keeps asking Windows for system details",
		"The Windows service that answers questions about the hardware is busy, usually because some program asks it over and over.",
		"If it stays busy, close monitoring or vendor utilities one at a time to find which one."},
	// vmmemWSL is WSL 2; plain vmmem is any Hyper-V virtual machine (Windows
	// Sandbox, Hyper-V, Docker on Hyper-V), so it is not called WSL
	"vmmem":    vmCulprit,
	"vmmemwsl": wslCulprit,
	"com.docker.backend.exe": {"docker", "Docker is busy",
		"Docker Desktop's background service is using the processor or drive.",
		"Stop containers you are not using, or quit Docker Desktop when you do not need it."},
}

var (
	updateCulprit = culprit{"update", "Windows Update is installing",
		"Updates are being prepared or installed, which uses the drive and processor heavily.",
		"Let it finish, then restart when Windows asks."}
	indexCulprit = culprit{"indexing", "Windows is indexing files for search",
		"Search is reading files so it can find them quickly later.",
		"It calms down once the index is built. To index less: Settings > Privacy & security > Searching Windows."}
	vmCulprit = culprit{"vm", "A virtual machine is busy",
		"A virtual machine (Windows Sandbox, Hyper-V, or Docker on Hyper-V) is using the processor.",
		"Close virtual machines you are not using."}
	wslCulprit = culprit{"wsl", "WSL is busy",
		"The Linux virtual machine of WSL (or Docker, which runs inside it) is using the processor.",
		"Run wsl --shutdown in a terminal when you are not using Linux tools."}
)

func culpritFor(name string) (culprit, bool) {
	n := strings.ToLower(name)
	if c, ok := culprits[n]; ok {
		return c, true
	}
	c, ok := culprits[strings.TrimSuffix(n, ".exe")]
	return c, ok
}

// processes that are not programs the user can do anything about
var ignored = map[string]bool{"": true, "system idle process": true, "[system process]": true, "idle": true}

// group adds processes with the same name together
func group(procs []Proc) []Proc {
	by := map[string]*Proc{}
	biggest := map[string]float64{}
	var order []string
	for _, p := range procs {
		key := strings.ToLower(p.Name)
		if ignored[key] {
			continue
		}
		g, ok := by[key]
		if !ok {
			g = &Proc{Name: p.Name}
			by[key] = g
			order = append(order, key)
			biggest[key] = -1
		}
		g.CPUPct += p.CPUPct
		g.MemoryBytes += p.MemoryBytes
		g.IOBytesPerSec += p.IOBytesPerSec
		g.Count++
		if weight := p.CPUPct*float64(gb) + float64(p.MemoryBytes); weight > biggest[key] {
			biggest[key] = weight
			g.PID = p.PID
		}
	}
	out := make([]Proc, 0, len(order))
	for _, k := range order {
		out = append(out, *by[k])
	}
	return out
}

func top(procs []Proc, n int, value func(Proc) float64) []Proc {
	sorted := append([]Proc(nil), procs...)
	sort.SliceStable(sorted, func(i, j int) bool { return value(sorted[i]) > value(sorted[j]) })
	out := []Proc{}
	for _, p := range sorted {
		if len(out) == n || value(p) <= 0 {
			break
		}
		out = append(out, p)
	}
	return out
}

func gbText(b uint64) string {
	if b >= gb {
		return fmt.Sprintf("%.1f GB", float64(b)/gb)
	}
	return fmt.Sprintf("%d MB", b/mb)
}

func rateText(bps float64) string {
	if bps >= gb {
		return fmt.Sprintf("%.1f GB/s", bps/gb)
	}
	return fmt.Sprintf("%.0f MB/s", bps/mb)
}

// friendly turns "chrome.exe" x24 into "chrome.exe (24 processes)"
func friendly(p Proc) string {
	if p.Count > 1 {
		return fmt.Sprintf("%s (%d processes)", p.Name, p.Count)
	}
	return p.Name
}

func analyse(s Sample) Report {
	groups := group(s.Procs)
	r := Report{
		CPU: CPUInfo{Percent: round1(s.CPUPct), PerformancePct: round1(s.PerformancePct),
			Top: top(groups, 5, func(p Proc) float64 { return p.CPUPct })},
		Memory: MemoryInfo{TotalBytes: s.MemTotal, AvailableBytes: s.MemAvailable,
			Top: top(groups, 5, func(p Proc) float64 { return float64(p.MemoryBytes) })},
		Disk: DiskInfo{BusyPct: round1(s.DiskBusyPct),
			Top: top(groups, 5, func(p Proc) float64 { return p.IOBytesPerSec })},
		Findings:  []Finding{},
		SampledMs: s.SampledMs,
	}
	for _, list := range [][]Proc{r.CPU.Top, r.Memory.Top, r.Disk.Top} {
		for i := range list {
			list[i].CPUPct = round1(list[i].CPUPct)
			list[i].IOBytesPerSec = float64(int64(list[i].IOBytesPerSec))
		}
	}
	if s.MemTotal > 0 {
		r.Memory.UsedPct = round1(100 * float64(s.MemTotal-s.MemAvailable) / float64(s.MemTotal))
	}
	add := func(f Finding) { r.Findings = append(r.Findings, f) }
	reported := map[string]bool{} // culprit ids already explained

	// known background work
	for _, g := range groups {
		c, ok := culpritFor(g.Name)
		if !ok || reported[c.id] {
			continue
		}
		busy := g.CPUPct >= 5 || g.IOBytesPerSec >= 2*mb
		if !busy {
			continue
		}
		level := "medium"
		if g.CPUPct >= 25 || g.IOBytesPerSec >= 20*mb {
			level = "high"
		}
		detail := c.detail + " " + usage(g)
		add(Finding{ID: c.id, Level: level, Title: c.title, Detail: detail, Fix: c.fix, PID: g.PID, Process: g.Name})
		reported[c.id] = true
	}

	// WSL or another virtual machine holding memory, busy or not
	for _, g := range groups {
		c, ok := culpritFor(g.Name)
		if !ok || (c.id != "wsl" && c.id != "vm") || g.MemoryBytes < 2*gb {
			continue
		}
		level := "info"
		if r.Memory.UsedPct >= 80 {
			level = "high"
		} else if r.Memory.UsedPct >= 65 {
			level = "medium"
		}
		if c.id == "vm" {
			add(Finding{ID: "vm-memory", Level: level, Title: fmt.Sprintf("A virtual machine is holding %s of memory", gbText(g.MemoryBytes)),
				Detail: "A Hyper-V virtual machine (Windows Sandbox, Hyper-V, or Docker on Hyper-V) has this much memory set aside.",
				Fix:    "Shut down virtual machines you are not using.",
				PID:    g.PID, Process: g.Name})
			continue
		}
		add(Finding{ID: "wsl-memory", Level: level, Title: fmt.Sprintf("WSL is holding %s of memory", gbText(g.MemoryBytes)),
			Detail: "The Linux virtual machine of WSL (and Docker, which runs inside it) keeps memory it once used, even when idle.",
			Fix:    "Run wsl --shutdown in a terminal to give it back, or set a memory limit in .wslconfig in your user folder.",
			PID:    g.PID, Process: g.Name})
	}

	// the processor
	if s.CPUPct >= 50 {
		level := "medium"
		if s.CPUPct >= 80 {
			level = "high"
		}
		f := Finding{ID: "cpu", Level: level, Title: fmt.Sprintf("The processor is %.0f%% busy", s.CPUPct)}
		if len(r.CPU.Top) > 0 {
			t := r.CPU.Top[0]
			f.Detail = fmt.Sprintf("%s is using the most: %.0f%% of the processor.", friendly(t), t.CPUPct)
			f.PID, f.Process = t.PID, t.Name
			if _, known := culpritFor(t.Name); known {
				f.Fix = "That is Windows' own work (see above); it settles on its own."
			} else {
				f.Fix = fmt.Sprintf("If you are not using %s right now, close it.", t.Name)
			}
		}
		add(f)
	} else {
		// one program hogging the processor while the rest is quiet
		for _, t := range r.CPU.Top {
			if t.CPUPct < 25 {
				break
			}
			if _, known := culpritFor(t.Name); known {
				continue
			}
			add(Finding{ID: "cpu-app", Level: "medium", Title: fmt.Sprintf("%s is using %.0f%% of the processor", t.Name, t.CPUPct),
				Detail: fmt.Sprintf("%s is keeping the processor busy.", friendly(t)),
				Fix:    fmt.Sprintf("If you are not using %s right now, close it.", t.Name), PID: t.PID, Process: t.Name})
			break
		}
	}

	// memory
	if r.Memory.UsedPct >= 80 {
		level := "medium"
		if r.Memory.UsedPct >= 90 {
			level = "high"
		}
		names := []string{}
		for _, t := range r.Memory.Top[:min(3, len(r.Memory.Top))] {
			names = append(names, fmt.Sprintf("%s %s", friendly(t), gbText(t.MemoryBytes)))
		}
		state := "getting full"
		if level == "high" {
			state = "almost full"
		}
		f := Finding{ID: "memory", Level: level,
			Title: fmt.Sprintf("Memory is %s: %s of %s in use", state, gbText(s.MemTotal-s.MemAvailable), gbText(s.MemTotal)),
			Detail: "When memory runs out, Windows moves things to the drive, which is much slower. Using the most: " +
				strings.Join(names, ", ") + ".",
			Fix: "Close programs and browser tabs you are not using."}
		if len(r.Memory.Top) > 0 {
			f.PID, f.Process = r.Memory.Top[0].PID, r.Memory.Top[0].Name
		}
		add(f)
	}

	// the drive
	if s.DiskBusyPct >= 60 {
		level := "medium"
		if s.DiskBusyPct >= 90 {
			level = "high"
		}
		f := Finding{ID: "disk", Level: level, Title: fmt.Sprintf("The drive is busy %.0f%% of the time", s.DiskBusyPct),
			Detail: "Programs have to wait for the drive, so opening files and apps is slow."}
		if len(r.Disk.Top) > 0 {
			t := r.Disk.Top[0]
			f.Detail += fmt.Sprintf(" %s is reading and writing the most (%s).", friendly(t), rateText(t.IOBytesPerSec))
			f.PID, f.Process = t.PID, t.Name
			if _, known := culpritFor(t.Name); !known {
				f.Fix = fmt.Sprintf("If %s is not doing something you asked for, close it.", t.Name)
			}
		}
		add(f)
	}

	// the processor held back: busy, yet running well below its base speed
	busy := s.CPUTimePct
	if busy <= 0 {
		busy = s.CPUPct
	}
	switch {
	case s.PerformancePct >= 0 && s.PerformancePct < 70 && busy >= 40:
		f := Finding{ID: "throttled", Level: "high",
			Title: fmt.Sprintf("The processor is running at %.0f%% of its base speed", s.PerformancePct)}
		switch {
		case s.BatterySaver:
			f.Detail, f.Fix = "Battery saver is on, which holds the processor back.", "Plug in, or turn off battery saver in Settings > System > Power & battery."
		case s.OnBattery:
			f.Detail, f.Fix = "The PC is on battery, and Windows slows the processor to save it.", "Plug in, or set Power mode to Best performance in Settings > System > Power & battery."
		default:
			f.Detail, f.Fix = "It is busy but running below its base speed, which often means it is too hot.", "Make sure the vents are not blocked (not on a bed or lap), and clean dust out of them."
		}
		add(f)
	case s.BatterySaver:
		add(Finding{ID: "battery-saver", Level: "medium", Title: "Battery saver is on",
			Detail: "Windows holds the processor back and pauses some background work to save battery.",
			Fix:    "Turn it off in Settings > System > Power & battery when you need speed."})
	}
	if strings.Contains(strings.ToLower(s.PowerPlan), "saver") {
		add(Finding{ID: "power-plan", Level: "medium", Title: "The power plan is set to save power",
			Detail: fmt.Sprintf("The %s plan keeps the processor slow even when plugged in.", s.PowerPlan),
			Fix:    "Choose Balanced in Control Panel > Power Options."})
	}

	// the system drive
	if s.SystemTotal > 0 {
		freePct := 100 * float64(s.SystemFree) / float64(s.SystemTotal)
		if freePct < 10 {
			level := "medium"
			if freePct < 5 {
				level = "high"
			}
			add(Finding{ID: "drive-full", Level: level, Title: fmt.Sprintf("%s is nearly full: %s free", s.SystemDrive, gbText(s.SystemFree)),
				Detail: "Windows needs free space for updates, temporary files and memory it moves to the drive. A full drive slows everything down.",
				Fix:    "Free some space with Disk & health or Disk usage."})
		}
	}

	if s.RebootPending {
		add(Finding{ID: "restart", Level: "medium", Title: "Windows is waiting for a restart",
			Detail: "Updates were installed but cannot finish until the PC restarts. Until then some of the work keeps running.",
			Fix:    "Save your work and restart."})
	}
	if s.UptimeHours >= 14*24 {
		add(Finding{ID: "uptime", Level: "info", Title: fmt.Sprintf("Not restarted for %.0f days", s.UptimeHours/24),
			Detail: "Programs that leak a little memory, and half-finished updates, add up over weeks.",
			Fix:    "Restart now and then (Shut down with fast startup does not count as a restart)."})
	}

	rank := map[string]int{"high": 0, "medium": 1, "info": 2}
	sort.SliceStable(r.Findings, func(i, j int) bool { return rank[r.Findings[i].Level] < rank[r.Findings[j].Level] })

	r.Calm = true
	for _, f := range r.Findings {
		if f.Level != "info" {
			r.Calm = false
		}
	}
	if r.Calm {
		r.Summary = "Nothing is slowing this PC down right now."
	} else {
		r.Summary = r.Findings[0].Title + "."
	}
	return r
}

func usage(g Proc) string {
	parts := []string{}
	if g.CPUPct >= 1 {
		parts = append(parts, fmt.Sprintf("%.0f%% of the processor", g.CPUPct))
	}
	if g.IOBytesPerSec >= mb {
		parts = append(parts, rateText(g.IOBytesPerSec)+" of drive traffic")
	}
	if len(parts) == 0 {
		return ""
	}
	return "Right now: " + strings.Join(parts, ", ") + "."
}

func round1(v float64) float64 {
	if v < 0 {
		return -1
	}
	return float64(int(v*10+0.5)) / 10
}
