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
	"myproject/internal/devcache"
	"myproject/internal/diagnostics"
	"myproject/internal/hardware"
	"myproject/internal/history"
	"myproject/internal/locks"
	"myproject/internal/network"
	"myproject/internal/process"
	"myproject/internal/security"
	"myproject/internal/storage"
	"myproject/internal/system"
	"myproject/internal/winwidget"

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

func (a *App) ScanCleanableFiles() (functions.CleanerResult, error) {
	result := functions.GetCleanableFiles()
	return result, nil
}

// ScanSafeCleanableFiles scans only user directories (no admin required)
func (a *App) ScanSafeCleanableFiles() (functions.CleanerResult, error) {
	result := functions.SafeClean()
	return result, nil
}

// CleanSelectedFiles cleans the selected files and returns results
func (a *App) CleanSelectedFiles(files []functions.FileInfo) map[string]interface{} {
	count, size, failures := functions.CleanFiles(files)

	return map[string]interface{}{
		"cleanedCount":  count,
		"cleanedSize":   size,
		"formattedSize": functions.GetFormattedSize(size),
		"failures":      failures,
	}
}

// PurgeJunkFiles purges all scanned temporary files, logs, and caches
func (a *App) PurgeJunkFiles() map[string]interface{} {
	count, size, failures := functions.PurgeJunk()

	return map[string]interface{}{
		"cleanedCount":  count,
		"cleanedSize":   size,
		"formattedSize": functions.GetFormattedSize(size),
		"failures":      failures,
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

// CheckCleanerPermissions checks if we have elevated permissions
func (a *App) CheckCleanerPermissions() functions.PermissionStatus {
	return functions.CheckPermissions()
}

// GetFormattedSize returns a human-readable size
func (a *App) GetFormattedSize(size int64) string {
	return functions.GetFormattedSize(size)
}

// CheckUpdate checks for updates
func (a *App) CheckUpdate() (string, error) {
	updateStatus, err := functions.CheckUpdate()

	if err != nil {
		return "Error Occured", fmt.Errorf("error checking for update: %w", err)
	}

	jsonBytes, err := json.MarshalIndent(updateStatus, "", "  ")
	if err != nil {
		return "Error Occured", fmt.Errorf("error marshalling JSON: %w", err)
	}

	return string(jsonBytes), nil
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

// FindNodeModules searches a folder for node_modules directories; a new search cancels the previous one
func (a *App) FindNodeModules(root string) ([]devcache.Item, error) {
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	a.devScanMu.Lock()
	if a.devScanCancel != nil {
		a.devScanCancel()
	}
	a.devScanCancel = cancel
	a.devScanMu.Unlock()

	return devcache.FindNodeModules(ctx, root, func(current string) {
		if ctx.Err() == nil {
			runtime.EventsEmit(a.ctx, "devcache-progress", current)
		}
	})
}

// CancelNodeModulesSearch stops the running node_modules search, if any
func (a *App) CancelNodeModulesSearch() {
	a.devScanMu.Lock()
	defer a.devScanMu.Unlock()
	if a.devScanCancel != nil {
		a.devScanCancel()
		a.devScanCancel = nil
	}
}

// DeleteDevCaches removes the given cache or node_modules folders; any other path is refused.
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

// GetDriveHealthDetails returns storage health inspection data
func (a *App) GetDriveHealthDetails() ([]hardware.DriveHealthInfo, error) {
	return hardware.GetDriveHealthDetails()
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
