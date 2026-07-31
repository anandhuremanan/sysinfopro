package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"myproject/functions"
	"myproject/internal/diagnostics"
	"myproject/internal/hardware"
	"myproject/internal/network"
	"myproject/internal/process"
	"myproject/internal/system"
)

// App struct
type App struct {
	ctx context.Context
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
	internetSpeedStat, err := functions.GetInterNetSpeed()
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
