package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sysinfopro/functions"
	"sysinfopro/internal/alerts"
	"sysinfopro/internal/batteryreport"
	"sysinfopro/internal/cleaner"
	"sysinfopro/internal/cpuinfo"
	"sysinfopro/internal/devcache"
	"sysinfopro/internal/diagnostics"
	"sysinfopro/internal/dupes"
	"sysinfopro/internal/gitsweep"
	"sysinfopro/internal/handover"
	"sysinfopro/internal/hardware"
	"sysinfopro/internal/history"
	"sysinfopro/internal/hotkey"
	"sysinfopro/internal/locks"
	"sysinfopro/internal/logging"
	"sysinfopro/internal/netdoctor"
	"sysinfopro/internal/network"
	"sysinfopro/internal/notify"
	"sysinfopro/internal/process"
	"sysinfopro/internal/reminder"
	"sysinfopro/internal/security"
	"sysinfopro/internal/slownow"
	"sysinfopro/internal/storage"
	"sysinfopro/internal/system"
	"sysinfopro/internal/vdisk"
	"sysinfopro/internal/winget"
	"sysinfopro/internal/winwidget"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx              context.Context
	speedTestRunning atomic.Bool

	// network folders the user chose in a dialog; other network paths are refused
	picked pickedRoots

	// the one running instance of each long job; a new run cancels the old one
	diskScanJob, leftoverJob, devDeleteJob, cleanJob, gitJob, dupesJob cancelSlot

	// live processor and memory readings, sampled by one goroutine
	metrics *functions.Monitor
	// what the page last said about its own visibility; minimising is checked
	// separately, because Wails does not tell the page about it
	pageVisible atomic.Bool

	dupesMu   sync.Mutex // guards dupesLast
	dupesLast *dupes.Result

	alerts *alerts.Monitor

	hotkeyMu       sync.Mutex
	hotkeyListener *hotkey.Listener
	hotkeySettings hotkey.Settings

	// guards loading and saving reminder settings; never held during a scan
	reminderMu sync.Mutex
	// a reminder check is running; a second one is not started meanwhile
	reminderRunning atomic.Bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := logging.Init(""); err != nil {
		slog.Warn("log file unavailable; logging to standard error", "err", err)
	}
	slog.Info("SysInfo Pro starting", "version", appVersion())

	// one sampler for processor and memory; events go out from here, so the
	// domain packages stay free of the UI framework
	a.metrics = functions.NewMonitor()
	a.metrics.OnUpdate = func(cpu functions.CPUStats, mem functions.MemoryStats) {
		runtime.EventsEmit(ctx, "cpu-stats-update", cpu)
		runtime.EventsEmit(ctx, "ram-stats-update", mem)
	}
	logging.Go("metrics", func() { a.metrics.Run(ctx) })
	a.pageVisible.Store(true)
	logging.Go("window visibility", func() { a.watchVisibility(ctx) })

	a.alerts = alerts.NewMonitor(alerts.LoadSettings())
	logging.Go("drive alerts", func() { a.watchDrives(ctx) })
	a.applyHotkey(hotkey.LoadSettings())
	logging.Go("monthly reminder", func() { a.reminderLoop(ctx) })
}

// SetWindowVisible tells the app whether its window (or the widget) is on
// screen. While hidden, live readings are taken rarely and no updates are sent,
// so a minimised app costs next to nothing; drive alerts keep running.
func (a *App) SetWindowVisible(visible bool) {
	a.pageVisible.Store(visible)
	a.applyVisibility()
}

// applyVisibility slows the live readings while the window is minimised or the
// page is hidden; the widget is a normal visible window, so it keeps updating
func (a *App) applyVisibility() {
	if a.metrics == nil || a.ctx == nil {
		return
	}
	a.metrics.SetVisible(a.pageVisible.Load() && !runtime.WindowIsMinimised(a.ctx))
}

// watchVisibility notices minimise and restore, which the page is not told about
// for frameless windows
func (a *App) watchVisibility(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.applyVisibility()
		}
	}
}

// GetDiagnosticsText returns the app version, the Windows build and the most
// recent log lines, for the user to copy into a bug report
func (a *App) GetDiagnosticsText() string {
	return logging.Text(appVersion(), windowsVersion(), 200)
}

// reminderLoop runs the monthly cleanup check when it is due: a couple of
// minutes after start, so it never slows opening the app, then every few hours
func (a *App) reminderLoop(ctx context.Context) {
	wait := 2 * time.Minute
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		a.runReminderCheck(ctx, false)
		wait = 6 * time.Hour
	}
}

// runReminderCheck measures what could be freed and raises the reminder if it is
// worth it; force skips the once-a-month rule
func (a *App) runReminderCheck(ctx context.Context, force bool) {
	if !a.reminderRunning.CompareAndSwap(false, true) {
		return
	}
	defer a.reminderRunning.Store(false)

	// read the settings, then let go of the lock: the scan below takes minutes,
	// and turning the reminder on or off must not wait for it
	a.reminderMu.Lock()
	s := reminder.LoadSettings()
	a.reminderMu.Unlock()
	if !s.Enabled || (!force && !reminder.Due(s, time.Now())) {
		return
	}

	var f reminder.Findings
	for _, c := range devcache.ScanCaches(ctx) {
		if c.Deletable {
			f.CacheBytes += c.SizeBytes
		}
	}
	f.JunkBytes = cleaner.Scan(ctx, cleaner.DefaultLocations()).TotalBytes
	if s.ProjectsRoot != "" {
		cutoff := time.Now().Add(-reminder.StaleAfter).Format("2006-01-02")
		leftovers, _ := devcache.FindLeftovers(ctx, s.ProjectsRoot, nil)
		for _, l := range leftovers {
			if l.LastUsed != "" && l.LastUsed < cutoff {
				f.LeftoverBytes += l.SizeBytes
				f.LeftoverCount++
			}
		}
	}
	if ctx.Err() != nil {
		return
	}

	// record the check on the settings as they are now, keeping any change
	// made while the scan ran
	a.reminderMu.Lock()
	latest := reminder.LoadSettings()
	latest.LastCheck = time.Now().Unix()
	reminder.SaveSettings(latest)
	a.reminderMu.Unlock()
	if title, body, ok := reminder.Message(f); ok {
		a.raise(alerts.Alert{Kind: "reminder", Title: title, Body: body, At: time.Now().UnixMilli()})
	}
}

// applyHotkey (re)registers the widget shortcut; a press asks the frontend to
// switch between the widget and the full window
func (a *App) applyHotkey(s hotkey.Settings) hotkey.Settings {
	a.hotkeyMu.Lock()
	defer a.hotkeyMu.Unlock()
	a.hotkeyListener.Stop()
	a.hotkeyListener = nil
	s.Error = ""
	if s.Enabled {
		listener, err := hotkey.Listen(s.Combo, func() {
			runtime.EventsEmit(a.ctx, "widget-hotkey")
		})
		if err != nil {
			s.Error = err.Error()
		} else {
			a.hotkeyListener = listener
		}
	}
	a.hotkeySettings = s
	return s
}

// how often free space is checked for alerts
const driveCheckInterval = 30 * time.Second

// watchDrives checks every drive's free space while the app (or its widget) is
// open, and raises low-space and fast-drop alerts
func (a *App) watchDrives(ctx context.Context) {
	ticker := time.NewTicker(driveCheckInterval)
	defer ticker.Stop()
	for {
		if drives, err := storage.ListDrives(); err == nil {
			now := time.Now()
			for _, d := range drives {
				name := strings.TrimRight(d.Path, `\/`)
				for _, alert := range a.alerts.Observe(name, d.FreeBytes, d.TotalBytes, now) {
					a.raise(alert)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// raise shows an alert as a Windows notification and inside the app, which
// still works where an organisation has turned notifications off
func (a *App) raise(alert alerts.Alert) {
	// inside the app the banner has a button for the next step instead
	body := alert.Body
	switch alert.Kind {
	case "low", "dropping":
		body += " Open SysInfo Pro to see what is taking the space."
	case "reminder":
		body += " Open SysInfo Pro, then Dev caches, to review it."
	}
	notify.Send(alert.Title, body)
	runtime.EventsEmit(a.ctx, "drive-alert", alert)
}

// onSecondInstance brings the open window forward when the app is started again,
// for example from the Start menu or by clicking a notification
func (a *App) onSecondInstance(options.SecondInstanceData) {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
}

// GetSysInfo is the startup snapshot of the PC: system, processor, memory and disk
func (a *App) GetSysInfo() (*functions.SystemInfo, error) {
	sysInfo, err := functions.GetSystemInfo()
	if err != nil {
		return nil, fmt.Errorf("error gathering system info: %w", err)
	}
	return sysInfo, nil
}

// GetInterNetSpeed runs a speed test, sending "speedtest-progress" as it goes
func (a *App) GetInterNetSpeed() (*functions.InternetSpeedStat, error) {
	// Two tests at once would share the line and both read low
	if !a.speedTestRunning.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("a speed test is already running")
	}
	defer a.speedTestRunning.Store(false)

	internetSpeedStat, err := functions.GetInterNetSpeedWithProgress(func(p functions.SpeedProgress) {
		runtime.EventsEmit(a.ctx, "speedtest-progress", p)
	})
	if err != nil {
		return nil, fmt.Errorf("error gathering internet speed: %w", err)
	}
	return internetSpeedStat, nil
}

// GetCPUDetail reads the processor's live state: total and per-processor use,
// the real current speed, and how many processes and threads are running
func (a *App) GetCPUDetail() cpuinfo.Detail {
	return a.metrics.Detail()
}

// GetUptimeSeconds is how long Windows has been running since it last started
func (a *App) GetUptimeSeconds() uint64 {
	return uint64(time.Duration(windowsTickMs()) * time.Millisecond / time.Second)
}

// GetCPUStats is the latest processor reading, as Task Manager counts use
func (a *App) GetCPUStats() (*functions.CPUStats, error) {
	return a.metrics.CPU(), nil
}

// GetMemoryStats is the latest memory reading
func (a *App) GetMemoryStats() (*functions.MemoryStats, error) {
	return a.metrics.Memory()
}

func (a *App) GetMemoryVelocity() (*diagnostics.MemoryVelocitySample, error) {
	return diagnostics.GetMemoryVelocity()
}

func (a *App) CheckInternetConnection() bool {
	return functions.IsConnectedToInternet()
}

// GetBatteryDetails is the battery's live state (charge, power source, capacity);
// nil on a PC without a battery
func (a *App) GetBatteryDetails() (*functions.BatteryDetails, error) {
	batteryDetails, err := functions.GetBatteryDetails()
	if err != nil {
		return nil, fmt.Errorf("error gathering battery details: %w", err)
	}
	return batteryDetails, nil
}

// ScanJunk totals the temporary files and browser caches that CleanJunk would remove
func (a *App) ScanJunk() cleaner.ScanResult {
	return cleaner.Scan(a.ctx, cleaner.DefaultLocations())
}

// CleanJunk removes temporary files and browser caches, leaving anything changed
// in the last hour. Progress is sent as "cleaner-progress" events.
func (a *App) CleanJunk() cleaner.Summary {
	ctx, done := a.cleanJob.start(a.ctx)
	defer done()

	return cleaner.Clean(ctx, cleaner.DefaultLocations(), func(p cleaner.Progress) {
		runtime.EventsEmit(a.ctx, "cleaner-progress", p)
	})
}

// CancelCleanJunk stops a running CleanJunk after the file it is on
func (a *App) CancelCleanJunk() {
	a.cleanJob.stop()
}

// SaveSpecCardPNG saves the base64 PNG directly to Downloads without browser popups
func (a *App) SaveSpecCardPNG(base64Data string) (string, error) {
	idx := strings.Index(base64Data, ",")
	if idx != -1 {
		base64Data = base64Data[idx+1:]
	}

	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}
	if !isPNG(data) {
		return "", fmt.Errorf("the spec card is not a PNG image")
	}

	filename := fmt.Sprintf("SysInfo_SpecCard_%d.png", time.Now().Unix())
	targetPath := filepath.Join(downloadsDir(), filename)

	err = os.WriteFile(targetPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write spec card PNG file: %w", err)
	}

	return targetPath, nil
}

// GetActiveSockets returns all active listening network ports and owning processes
func (a *App) GetActiveSockets() ([]network.SocketItem, error) {
	return network.GetActiveSockets()
}

// RunThermalStressTest executes a 10-second multi-threaded CPU thermal stress burn
func (a *App) RunThermalStressTest() (*hardware.StressResult, error) {
	return hardware.RunThermalStressTest()
}

// GetStartupItems returns all auto-starting registry and folder startup programs
func (a *App) GetStartupItems() ([]system.StartupItem, error) {
	return system.GetStartupItems()
}

// GetEnvPathAnalysis debugs system PATH entries and environment variables
func (a *App) GetEnvPathAnalysis() (*system.EnvPathAnalysis, error) {
	return system.GetEnvPathAnalysis()
}

// GetSoftwareHealthDetails audits installed software versions against official releases
func (a *App) GetSoftwareHealthDetails() ([]system.SoftwareItem, error) {
	return system.GetSoftwareHealthDetails()
}

// --- Version 3.0 RPC Endpoints ---

// GetDrives lists mounted partitions with their capacity
func (a *App) GetDrives() ([]storage.DriveInfo, error) {
	return storage.ListDrives()
}

// AnalyzeDiskUsage sizes everything directly inside path; a new scan cancels the previous one
func (a *App) AnalyzeDiskUsage(path string) (*storage.FolderUsage, error) {
	if err := a.picked.check(path); err != nil {
		return nil, err
	}
	ctx, done := a.diskScanJob.start(a.ctx)
	defer done()

	return storage.AnalyzeFolder(ctx, path, func(p storage.ScanProgress) {
		if ctx.Err() == nil {
			runtime.EventsEmit(a.ctx, "disk-scan-progress", p)
		}
	})
}

// CancelDiskScan stops the running disk usage scan, if any
func (a *App) CancelDiskScan() {
	a.diskScanJob.stop()
}

// RevealInExplorer opens the file manager with the given file or folder selected
func (a *App) RevealInExplorer(path string) error {
	if err := a.picked.check(path); err != nil {
		return err
	}
	return storage.RevealInFileManager(path)
}

// GetHardwareSpecs returns the hardware inventory (system, BIOS, CPU, GPU, memory modules, disks)
func (a *App) GetHardwareSpecs() (*hardware.HardwareSpecs, error) {
	return hardware.GetHardwareSpecs(a.ctx)
}

// GetSecurityReport runs read-only checks of antivirus, firewall, encryption and update state
func (a *App) GetSecurityReport() (*security.Report, error) {
	return security.GetSecurityReport(a.ctx)
}

// ScanDevCaches sizes the caches of common package managers and build tools
func (a *App) ScanDevCaches() []devcache.Item {
	return devcache.ScanCaches(a.ctx)
}

// PickFolder opens the system folder picker and returns the chosen path ("" if cancelled)
func (a *App) PickFolder(title string) (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	a.picked.add(path)
	return path, err
}

// FindLeftovers searches a folder for project leftovers (node_modules, Rust target,
// Python virtual environments and similar); a new search cancels the previous one
func (a *App) FindLeftovers(root string) ([]devcache.Item, error) {
	if err := a.picked.check(root); err != nil {
		return nil, err
	}
	ctx, done := a.leftoverJob.start(a.ctx)
	defer done()

	// the monthly reminder looks for old leftovers in the folder searched last
	a.reminderMu.Lock()
	if s := reminder.LoadSettings(); s.ProjectsRoot != root {
		s.ProjectsRoot = root
		reminder.SaveSettings(s)
	}
	a.reminderMu.Unlock()

	return devcache.FindLeftovers(ctx, root, func(current string) {
		if ctx.Err() == nil {
			runtime.EventsEmit(a.ctx, "devcache-progress", current)
		}
	})
}

// CancelLeftoverSearch stops the running leftover search, if any
func (a *App) CancelLeftoverSearch() {
	a.leftoverJob.stop()
}

// DeleteDevCaches removes the given caches or project leftovers; any other path is refused.
// Progress is sent as "devcache-delete-progress" events.
func (a *App) DeleteDevCaches(paths []string) *devcache.DeleteResult {
	var refused []string
	allowed := paths[:0:0]
	for _, p := range paths {
		if a.picked.allowed(p) {
			allowed = append(allowed, p)
		} else {
			refused = append(refused, p+": "+errNetworkPath.Error())
		}
	}
	ctx, done := a.devDeleteJob.start(a.ctx)
	defer done()

	result := devcache.Delete(ctx, allowed, func(p devcache.DeleteProgress) {
		runtime.EventsEmit(a.ctx, "devcache-delete-progress", p)
	})
	if result != nil {
		result.Failed = append(result.Failed, refused...)
	}
	return result
}

// CancelDevCacheDelete stops a running DeleteDevCaches after the file it is on;
// whatever was already removed stays removed
func (a *App) CancelDevCacheDelete() {
	a.devDeleteJob.stop()
}

// SaveSystemReport writes an HTML report to Downloads and returns its path
func (a *App) SaveSystemReport(html string) (string, error) {
	filename := fmt.Sprintf("SysInfo_Report_%s.html", time.Now().Format("2006-01-02_150405"))
	targetPath := filepath.Join(downloadsDir(), filename)

	if err := os.WriteFile(targetPath, []byte(withCSP(html)), 0644); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}
	return targetPath, nil
}

// GetPrivateTerms lists words that identify this PC or its user (the PC's name
// and the Windows user name), so a shared report can leave them out
func (a *App) GetPrivateTerms() []string {
	terms := []string{}
	if h, err := os.Hostname(); err == nil && h != "" {
		terms = append(terms, h)
	}
	if u := os.Getenv("USERNAME"); u != "" {
		terms = append(terms, u)
	}
	if p := os.Getenv("USERPROFILE"); p != "" {
		if base := filepath.Base(p); base != "" && base != os.Getenv("USERNAME") {
			terms = append(terms, base)
		}
	}
	return terms
}

// PickFile opens the system file picker and returns the chosen path ("" if cancelled)
func (a *App) PickFile(title string) (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	a.picked.add(path)
	return path, err
}

// FindLockingProcesses lists the processes that hold a file, or files in a folder, open
func (a *App) FindLockingProcesses(path string) (*locks.Result, error) {
	if err := a.picked.check(path); err != nil {
		return nil, err
	}
	return locks.Find(path)
}

// WatchFolder is a suggested place to watch for disk growth
type WatchFolder struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

// GetWatchFolders suggests the folders where disk space usually disappears
func (a *App) GetWatchFolders() []WatchFolder {
	folders := []WatchFolder{}
	for _, f := range storage.WatchFolders() {
		folders = append(folders, WatchFolder{Label: f.Label, Path: f.Path})
	}
	return folders
}

// GetChanges compares the machine with the saved baseline. The first run saves a
// baseline and reports nothing to compare.
func (a *App) GetChanges() (*history.Changes, error) {
	store, err := history.DefaultStore()
	if err != nil {
		return nil, fmt.Errorf("cannot open snapshot storage: %w", err)
	}

	current := history.Take()
	baseline, err := store.Latest()
	if err != nil {
		return nil, err
	}
	if baseline == nil {
		if err := store.Save(current); err != nil {
			return nil, fmt.Errorf("cannot save the first snapshot: %w", err)
		}
		return &history.Changes{
			HasBaseline: false,
			Since:       current.TakenAt.Format(time.RFC3339),
			Categories:  []history.CategoryDiff{},
		}, nil
	}
	return history.Compare(*baseline, current), nil
}

// SaveChangesBaseline makes the current state the new baseline ("mark as reviewed")
func (a *App) SaveChangesBaseline() error {
	store, err := history.DefaultStore()
	if err != nil {
		return fmt.Errorf("cannot open snapshot storage: %w", err)
	}
	return store.Save(history.Take())
}

// SetWidgetMode keeps the window visible through "Show desktop" (Win+D) while it is a widget
func (a *App) SetWidgetMode(active bool) {
	// must match Title in main.go
	winwidget.SetActive("SysInfo Pro", active)
}

// --- Version 2.0 RPC Endpoints ---

// GetTopProcesses returns top RAM/CPU processes
func (a *App) GetTopProcesses(limit int) ([]process.ProcessItem, error) {
	if limit <= 0 {
		limit = 10
	}
	return process.GetTopProcesses(limit)
}

// TerminateProcess ends a process by PID; critical Windows processes and this
// app are refused
func (a *App) TerminateProcess(pid int32) error {
	return process.TerminateProcess(pid)
}

// TerminateProcessChecked ends a process only if it still has the given name
// and start time (Unix ms; 0 skips that check), so a reused PID is never ended
func (a *App) TerminateProcessChecked(pid int32, name string, startedMs int64) error {
	return process.TerminateProcessChecked(pid, name, startedMs)
}

// GetDriveHealth reports what each drive and Windows say about its condition
func (a *App) GetDriveHealth() ([]hardware.DriveHealth, error) {
	return hardware.GetDriveHealth(a.ctx)
}

// GetDiagnosticReport returns system health report
func (a *App) GetDiagnosticReport() (diagnostics.DiagnosticReport, error) {
	cpuStats := a.metrics.CPU()
	memStats, err := a.metrics.Memory()
	if err != nil {
		slog.Warn("diagnostic report: memory reading failed", "err", err)
	}
	batDetails, err := functions.GetBatteryDetails()
	if err != nil {
		slog.Info("diagnostic report: no battery reading", "err", err)
	}
	sysInfo, err := functions.GetSystemInfo()
	if err != nil {
		slog.Warn("diagnostic report: system information failed", "err", err)
	}

	input := diagnostics.EvaluationInput{
		NetworkConnected: functions.IsConnectedToInternet(),
		PingLatencyMs:    -1, // not measured here; never show a made-up number
	}

	if cpuStats != nil {
		input.CpuUsagePct = cpuStats.Usage
		input.IsThrottling = cpuStats.IsThrottling
	}
	if memStats != nil {
		input.RamUsagePct = memStats.UsagePercent
		input.SwapUsedMB = memStats.SwapUsedMB
		input.SwapTotalMB = memStats.SwapTotalMB
	}
	if sysInfo != nil {
		input.FreeDiskGB = sysInfo.FreeDiskGB
		input.TotalDiskGB = sysInfo.TotalDiskGB
	}
	if batDetails != nil {
		input.BatteryPct = batDetails.Percentage
		input.BatteryDegradation = batDetails.DegradationPct
	}

	return diagnostics.EvaluateSystemHealth(input), nil
}

// GetAlertSettings returns the low-space alert settings
func (a *App) GetAlertSettings() alerts.Settings {
	return a.alerts.Settings()
}

// SetAlertSettings saves and applies new low-space alert settings
func (a *App) SetAlertSettings(s alerts.Settings) error {
	a.alerts.SetSettings(s)
	return alerts.SaveSettings(s)
}

// GetNotificationStatus reports whether Windows will show the app's notifications
func (a *App) GetNotificationStatus() string {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	return notify.Status(ctx)
}

// SendTestAlert raises a sample alert so the user can see what one looks like
func (a *App) SendTestAlert() {
	a.raise(alerts.Alert{
		Drive: "C:", Kind: "test", At: time.Now().UnixMilli(),
		Title: "Alerts are on",
		Body:  "SysInfo Pro will tell you here when a drive runs low or fills up fast.",
	})
}

// RunNetworkDoctor checks each link of the connection in turn (adapter, router,
// DNS, internet, and the site given, if any), sending "netdoctor-step" as each finishes
func (a *App) RunNetworkDoctor(site string) netdoctor.Report {
	return netdoctor.Run(a.ctx, strings.TrimSpace(site), func(s netdoctor.Step) {
		runtime.EventsEmit(a.ctx, "netdoctor-step", s)
	})
}

// CheckDevTools reports, for Python, Node, Java and other developer tools, which
// copy runs in a new terminal and what points builds at a different one
func (a *App) CheckDevTools() []system.DevTool {
	return system.CheckDevTools()
}

// FindDuplicates looks for files with the same contents under root, sending
// "dupes-progress" while it works; a new search cancels the previous one
func (a *App) FindDuplicates(root string, minSize int64) (*dupes.Result, error) {
	if err := a.picked.check(root); err != nil {
		return nil, err
	}
	ctx, done := a.dupesJob.start(a.ctx)
	defer done()

	res, err := dupes.Find(ctx, root, minSize, func(p dupes.Progress) {
		if ctx.Err() == nil {
			runtime.EventsEmit(a.ctx, "dupes-progress", p)
		}
	})
	if err != nil {
		return nil, err
	}
	a.dupesMu.Lock()
	a.dupesLast = res
	a.dupesMu.Unlock()
	return res, nil
}

// GetDuplicateFolders suggests personal folders to search for duplicates
func (a *App) GetDuplicateFolders() []dupes.Place {
	return dupes.UserFolders()
}

// CancelDuplicateSearch stops the running duplicate search, if any
func (a *App) CancelDuplicateSearch() {
	a.dupesJob.stop()
}

type DuplicateRemoval struct {
	Removed int      `json:"removed"`
	Freed   int64    `json:"freed"`
	Failed  []string `json:"failed"`
}

// RemoveDuplicates moves extra copies to the Recycle Bin. Only files from the
// last search are accepted, and at least one copy of each is always kept.
func (a *App) RemoveDuplicates(paths []string) (DuplicateRemoval, error) {
	out := DuplicateRemoval{Failed: []string{}}
	a.dupesMu.Lock()
	last := a.dupesLast
	a.dupesMu.Unlock()
	if last == nil {
		return out, fmt.Errorf("search again before removing files")
	}
	if err := a.picked.check(paths...); err != nil {
		return out, err
	}
	if err := dupes.CheckKeepsOne(last.Groups, paths); err != nil {
		return out, err
	}
	out.Removed, out.Freed, out.Failed = dupes.Recycle(paths)
	return out, nil
}

// DiagnoseSlowness measures the PC for a few seconds and explains what is
// slowing it down right now
func (a *App) DiagnoseSlowness() slownow.Report {
	return slownow.Diagnose(a.ctx)
}

// CheckHandover lists what to deal with before the PC changes hands. Nothing is changed.
func (a *App) CheckHandover() handover.Report {
	return handover.Check(a.ctx)
}

// GetBatteryReport reads Windows' own battery report: real capacity, its
// history, and recent time on battery
func (a *App) GetBatteryReport() (*batteryreport.Report, error) {
	return batteryreport.Get(a.ctx)
}

// FindGitRepos finds git projects under root and what in them exists only on
// this PC, sending "gitsweep-progress"; a new search cancels the previous one
func (a *App) FindGitRepos(root string) (*gitsweep.Result, error) {
	if err := a.picked.check(root); err != nil {
		return nil, err
	}
	ctx, done := a.gitJob.start(a.ctx)
	defer done()

	return gitsweep.Find(ctx, root, func(p gitsweep.Progress) {
		if ctx.Err() == nil {
			runtime.EventsEmit(a.ctx, "gitsweep-progress", p)
		}
	})
}

// CancelGitSweep stops the running project search, if any
func (a *App) CancelGitSweep() {
	a.gitJob.stop()
}

// ListVirtualDisks returns the virtual disks of WSL distributions and Docker Desktop
func (a *App) ListVirtualDisks() []vdisk.Disk {
	return vdisk.ListDisks()
}

// GetDockerUsage reports how much space Docker uses and could give back
func (a *App) GetDockerUsage() vdisk.DockerUsage {
	return vdisk.GetDockerUsage(a.ctx)
}

// PruneDocker removes Docker's unused data (volumes are kept) and returns the
// space Docker reports as reclaimed
func (a *App) PruneDocker() (string, error) {
	return vdisk.PruneDocker(a.ctx)
}

// GetWingetUpgrades lists the installed apps that Windows Package Manager can update
func (a *App) GetWingetUpgrades() winget.Result {
	return winget.ListUpgrades(a.ctx)
}

// UpgradeWithWinget updates one app by its winget ID
func (a *App) UpgradeWithWinget(id string) error {
	return winget.Upgrade(a.ctx, id)
}

// RemoveUserPathEntries removes broken or duplicate entries from the user's own
// PATH; original is the stored PATH they were chosen from
func (a *App) RemoveUserPathEntries(original string, indices []int) (*system.EnvPathAnalysis, error) {
	return system.RemoveUserPathEntries(original, indices)
}

// UndoPathChange restores the user PATH as it was before the last removal
func (a *App) UndoPathChange() (*system.EnvPathAnalysis, error) {
	return system.UndoPathChange()
}

// GetHotkeySettings returns the widget shortcut setting, with an error if Windows refused it
func (a *App) GetHotkeySettings() hotkey.Settings {
	a.hotkeyMu.Lock()
	defer a.hotkeyMu.Unlock()
	return a.hotkeySettings
}

// GetHotkeyCombos lists the shortcuts the user can choose from
func (a *App) GetHotkeyCombos() []string {
	return hotkey.Combos
}

// SetHotkeySettings saves and applies the widget shortcut setting
func (a *App) SetHotkeySettings(enabled bool, combo string) (hotkey.Settings, error) {
	s := hotkey.Settings{Enabled: enabled, Combo: combo}
	if err := hotkey.SaveSettings(s); err != nil {
		return a.GetHotkeySettings(), err
	}
	return a.applyHotkey(s), nil
}

// GetReminderSettings returns the monthly cleanup reminder setting
func (a *App) GetReminderSettings() reminder.Settings {
	return reminder.LoadSettings()
}

// SetReminderEnabled turns the monthly cleanup reminder on or off. Turning it on
// runs the first check straight away.
func (a *App) SetReminderEnabled(enabled bool) (reminder.Settings, error) {
	a.reminderMu.Lock()
	s := reminder.LoadSettings()
	s.Enabled = enabled
	err := reminder.SaveSettings(s)
	a.reminderMu.Unlock()
	if err == nil && enabled {
		logging.Go("reminder check", func() { a.runReminderCheck(a.ctx, true) })
	}
	return s, err
}

// IsStoreApp reports whether this copy was installed from the Microsoft Store
func (a *App) IsStoreApp() bool {
	return functions.IsStorePackage()
}
