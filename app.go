package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"myproject/functions"
	"myproject/internal/alerts"
	"myproject/internal/batteryreport"
	"myproject/internal/cleaner"
	"myproject/internal/devcache"
	"myproject/internal/diagnostics"
	"myproject/internal/dupes"
	"myproject/internal/hardware"
	"myproject/internal/gitsweep"
	"myproject/internal/handover"
	"myproject/internal/history"
	"myproject/internal/hotkey"
	"myproject/internal/locks"
	"myproject/internal/netdoctor"
	"myproject/internal/network"
	"myproject/internal/notify"
	"myproject/internal/process"
	"myproject/internal/reminder"
	"myproject/internal/security"
	"myproject/internal/slownow"
	"myproject/internal/storage"
	"myproject/internal/system"
	"myproject/internal/vdisk"
	"myproject/internal/winget"
	"myproject/internal/winwidget"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx              context.Context
	speedTestRunning atomic.Bool

	diskScanMu     sync.Mutex
	diskScanCancel context.CancelFunc

	devScanMu       sync.Mutex
	devScanCancel   context.CancelFunc
	devDeleteCancel context.CancelFunc

	cleanMu     sync.Mutex
	cleanCancel context.CancelFunc

	gitMu     sync.Mutex
	gitCancel context.CancelFunc

	dupesMu     sync.Mutex
	dupesCancel context.CancelFunc
	dupesLast   *dupes.Result

	alerts *alerts.Monitor

	hotkeyMu       sync.Mutex
	hotkeyListener *hotkey.Listener
	hotkeySettings hotkey.Settings

	reminderMu sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	functions.StartCPUMonitoring(ctx)
	functions.StartMemoryMonitoring(ctx)
	a.alerts = alerts.NewMonitor(alerts.LoadSettings())
	go a.watchDrives(ctx)
	a.applyHotkey(hotkey.LoadSettings())
	go a.reminderLoop(ctx)
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
	a.reminderMu.Lock()
	defer a.reminderMu.Unlock()

	s := reminder.LoadSettings()
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

	s.LastCheck = time.Now().Unix()
	reminder.SaveSettings(s)
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

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) GetSysInfo() (string, error) {
	sysInfo, err := functions.GetSystemInfo()
	if err != nil {
		return "Error Occured", fmt.Errorf("error gathering system info: %w", err)
	}

	jsonBytes, err := json.MarshalIndent(sysInfo, "", "  ")
	if err != nil {
		return "Error Occured", fmt.Errorf("error marshalling JSON: %w", err)
	}

	return string(jsonBytes), nil
}

func (a *App) GetInterNetSpeed() (string, error) {
	// Two tests at once would share the line and both read low
	if !a.speedTestRunning.CompareAndSwap(false, true) {
		return "Error Occured", fmt.Errorf("a speed test is already running")
	}
	defer a.speedTestRunning.Store(false)

	internetSpeedStat, err := functions.GetInterNetSpeedWithProgress(func(p functions.SpeedProgress) {
		runtime.EventsEmit(a.ctx, "speedtest-progress", p)
	})
	if err != nil {
		return "Error Occured", fmt.Errorf("error gathering internet speed: %w", err)
	}

	jsonBytes, err := json.MarshalIndent(internetSpeedStat, "", "  ")

	if err != nil {
		return "Error Occured", fmt.Errorf("error marshalling JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// GetUptimeSeconds is how long Windows has been running since it last started
func (a *App) GetUptimeSeconds() uint64 {
	return uint64(time.Duration(windowsTickMs()) * time.Millisecond / time.Second)
}

func (a *App) GetCPUStats() (*functions.CPUStats, error) {
	return functions.GetCPUStats()
}

func (a *App) GetMemoryStats() (*functions.MemoryStats, error) {
	return functions.GetMemoryStats()
}

func (a *App) GetMemoryVelocity() (*diagnostics.MemoryVelocitySample, error) {
	return diagnostics.GetMemoryVelocity()
}

func (a *App) CheckInternetConnection() bool {
	return functions.IsConnectedToInternet()
}

func (a *App) GetBatteryDetails() (string, error) {
	batteryDetails, err := functions.GetBatteryDetails()
	if err != nil {
		return "Error Occured", fmt.Errorf("error gathering battery details: %w", err)
	}

	jsonBytes, err := json.MarshalIndent(batteryDetails, "", "  ")
	if err != nil {
		return "Error Occured", fmt.Errorf("error marshalling JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// ScanJunk totals the temporary files and browser caches that CleanJunk would remove
func (a *App) ScanJunk() cleaner.ScanResult {
	return cleaner.Scan(a.ctx, cleaner.DefaultLocations())
}

// CleanJunk removes temporary files and browser caches, leaving anything changed
// in the last hour. Progress is sent as "cleaner-progress" events.
func (a *App) CleanJunk() cleaner.Summary {
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	a.cleanMu.Lock()
	if a.cleanCancel != nil {
		a.cleanCancel()
	}
	a.cleanCancel = cancel
	a.cleanMu.Unlock()

	return cleaner.Clean(ctx, cleaner.DefaultLocations(), func(p cleaner.Progress) {
		runtime.EventsEmit(a.ctx, "cleaner-progress", p)
	})
}

// CancelCleanJunk stops a running CleanJunk after the file it is on
func (a *App) CancelCleanJunk() {
	a.cleanMu.Lock()
	defer a.cleanMu.Unlock()
	if a.cleanCancel != nil {
		a.cleanCancel()
		a.cleanCancel = nil
	}
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	downloadsDir := filepath.Join(homeDir, "Downloads")
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		downloadsDir = homeDir
	}

	filename := fmt.Sprintf("SysInfo_SpecCard_%d.png", time.Now().Unix())
	targetPath := filepath.Join(downloadsDir, filename)

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
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	a.diskScanMu.Lock()
	if a.diskScanCancel != nil {
		a.diskScanCancel()
	}
	a.diskScanCancel = cancel
	a.diskScanMu.Unlock()

	return storage.AnalyzeFolder(ctx, path, func(p storage.ScanProgress) {
		if ctx.Err() == nil {
			runtime.EventsEmit(a.ctx, "disk-scan-progress", p)
		}
	})
}

// CancelDiskScan stops the running disk usage scan, if any
func (a *App) CancelDiskScan() {
	a.diskScanMu.Lock()
	defer a.diskScanMu.Unlock()
	if a.diskScanCancel != nil {
		a.diskScanCancel()
		a.diskScanCancel = nil
	}
}

// RevealInExplorer opens the file manager with the given file or folder selected
func (a *App) RevealInExplorer(path string) error {
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
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
}

// FindLeftovers searches a folder for project leftovers (node_modules, Rust target,
// Python virtual environments and similar); a new search cancels the previous one
func (a *App) FindLeftovers(root string) ([]devcache.Item, error) {
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	a.devScanMu.Lock()
	if a.devScanCancel != nil {
		a.devScanCancel()
	}
	a.devScanCancel = cancel
	a.devScanMu.Unlock()

	// the monthly reminder looks for old leftovers in the folder searched last
	if s := reminder.LoadSettings(); s.ProjectsRoot != root {
		s.ProjectsRoot = root
		reminder.SaveSettings(s)
	}

	return devcache.FindLeftovers(ctx, root, func(current string) {
		if ctx.Err() == nil {
			runtime.EventsEmit(a.ctx, "devcache-progress", current)
		}
	})
}

// CancelLeftoverSearch stops the running leftover search, if any
func (a *App) CancelLeftoverSearch() {
	a.devScanMu.Lock()
	defer a.devScanMu.Unlock()
	if a.devScanCancel != nil {
		a.devScanCancel()
		a.devScanCancel = nil
	}
}

// DeleteDevCaches removes the given caches or project leftovers; any other path is refused.
// Progress is sent as "devcache-delete-progress" events.
func (a *App) DeleteDevCaches(paths []string) *devcache.DeleteResult {
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	a.devScanMu.Lock()
	if a.devDeleteCancel != nil {
		a.devDeleteCancel()
	}
	a.devDeleteCancel = cancel
	a.devScanMu.Unlock()

	return devcache.Delete(ctx, paths, func(p devcache.DeleteProgress) {
		runtime.EventsEmit(a.ctx, "devcache-delete-progress", p)
	})
}

// CancelDevCacheDelete stops a running DeleteDevCaches after the file it is on;
// whatever was already removed stays removed
func (a *App) CancelDevCacheDelete() {
	a.devScanMu.Lock()
	defer a.devScanMu.Unlock()
	if a.devDeleteCancel != nil {
		a.devDeleteCancel()
		a.devDeleteCancel = nil
	}
}

// SaveSystemReport writes an HTML report to Downloads and returns its path
func (a *App) SaveSystemReport(html string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	downloadsDir := filepath.Join(homeDir, "Downloads")
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		downloadsDir = homeDir
	}

	filename := fmt.Sprintf("SysInfo_Report_%s.html", time.Now().Format("2006-01-02_150405"))
	targetPath := filepath.Join(downloadsDir, filename)

	if err := os.WriteFile(targetPath, []byte(html), 0644); err != nil {
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
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
}

// FindLockingProcesses lists the processes that hold a file, or files in a folder, open
func (a *App) FindLockingProcesses(path string) (*locks.Result, error) {
	return locks.Find(path)
}

// WatchFolder is a suggested place to watch for disk growth
type WatchFolder struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

// GetWatchFolders suggests the folders where disk space usually disappears
func (a *App) GetWatchFolders() []WatchFolder {
	home, _ := os.UserHomeDir()
	candidates := []WatchFolder{
		{"Local app data", os.Getenv("LOCALAPPDATA")},
		{"Roaming app data", os.Getenv("APPDATA")},
		{"Temp", os.TempDir()},
		{"Downloads", filepath.Join(home, "Downloads")},
		{"User folder", home},
	}

	folders := []WatchFolder{}
	for _, c := range candidates {
		if info, err := os.Stat(c.Path); c.Path != "" && err == nil && info.IsDir() {
			folders = append(folders, c)
		}
	}
	return folders
}

// currentSnapshot records startup programs, installed software, listening ports and PATH
func currentSnapshot() history.Snapshot {
	snapshot := history.Snapshot{
		TakenAt: time.Now(),
		Startup: []string{},
		Ports:   []string{},
		Path:    []string{},
		Apps:    system.InstalledApps(),
	}

	if items, err := system.GetStartupItems(); err == nil {
		for _, item := range items {
			snapshot.Startup = append(snapshot.Startup, fmt.Sprintf("%s (%s)", item.Name, item.Location))
		}
	}
	if ports, err := network.ListeningPorts(); err == nil {
		snapshot.Ports = ports
	}
	if analysis, err := system.GetEnvPathAnalysis(); err == nil && analysis != nil {
		for _, entry := range analysis.PathEntries {
			snapshot.Path = append(snapshot.Path, entry.Path)
		}
	}
	return snapshot
}

// GetChanges compares the machine with the saved baseline. The first run saves a
// baseline and reports nothing to compare.
func (a *App) GetChanges() (*history.Changes, error) {
	store, err := history.DefaultStore()
	if err != nil {
		return nil, fmt.Errorf("cannot open snapshot storage: %w", err)
	}

	current := currentSnapshot()
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
	return store.Save(currentSnapshot())
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

// TerminateProcess kills a process by PID
func (a *App) TerminateProcess(pid int32) error {
	return process.TerminateProcess(pid)
}

// GetDriveHealth reports what each drive and Windows say about its condition
func (a *App) GetDriveHealth() ([]hardware.DriveHealth, error) {
	return hardware.GetDriveHealth(a.ctx)
}

// GetDiagnosticReport returns system health report
func (a *App) GetDiagnosticReport() (diagnostics.DiagnosticReport, error) {
	cpuStats, _ := functions.GetCPUStats()
	memStats, _ := functions.GetMemoryStats()
	batDetails, _ := functions.GetBatteryDetails()
	sysInfo, _ := functions.GetSystemInfo()

	input := diagnostics.EvaluationInput{
		NetworkConnected: functions.IsConnectedToInternet(),
		PingLatencyMs:    24.0,
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
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.dupesMu.Lock()
	if a.dupesCancel != nil {
		a.dupesCancel()
	}
	a.dupesCancel = cancel
	a.dupesMu.Unlock()

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
	a.dupesMu.Lock()
	defer a.dupesMu.Unlock()
	if a.dupesCancel != nil {
		a.dupesCancel()
		a.dupesCancel = nil
	}
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
	if err := dupes.CheckKeepsOne(last.Groups, paths); err != nil {
		return out, err
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			out.Failed = append(out.Failed, p)
			continue
		}
		if err := dupes.ToRecycleBin(p); err != nil {
			out.Failed = append(out.Failed, p)
			continue
		}
		out.Removed++
		out.Freed += info.Size()
	}
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
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.gitMu.Lock()
	if a.gitCancel != nil {
		a.gitCancel()
	}
	a.gitCancel = cancel
	a.gitMu.Unlock()

	return gitsweep.Find(ctx, root, func(p gitsweep.Progress) {
		if ctx.Err() == nil {
			runtime.EventsEmit(a.ctx, "gitsweep-progress", p)
		}
	})
}

// CancelGitSweep stops the running project search, if any
func (a *App) CancelGitSweep() {
	a.gitMu.Lock()
	defer a.gitMu.Unlock()
	if a.gitCancel != nil {
		a.gitCancel()
		a.gitCancel = nil
	}
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
		go a.runReminderCheck(a.ctx, true)
	}
	return s, err
}

// IsStoreApp reports whether this copy was installed from the Microsoft Store
func (a *App) IsStoreApp() bool {
	return functions.IsStorePackage()
}
