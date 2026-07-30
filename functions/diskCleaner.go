package functions

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// PermissionStatus represents the system permissions status
type PermissionStatus struct {
	IsElevated          bool
	RequiresElevation   bool
	UnaccessiblePaths   []string
	ElevationCommand    string
	CanCleanUserFiles   bool
	CanCleanSystemFiles bool
}

// FileInfo represents information about a file/folder that can be cleaned
type FileInfo struct {
	Path           string
	Size           int64
	Name           string
	Location       string
	NeedsElevation bool
}

// CleanerResult represents the result of scanning the system
type CleanerResult struct {
	Files       map[string][]FileInfo
	TotalSize   int64
	Permissions PermissionStatus
}

// GetCleanableFiles scans the system for files that can be cleaned
func GetCleanableFiles() CleanerResult {
	result := CleanerResult{
		Files:       make(map[string][]FileInfo),
		TotalSize:   0,
		Permissions: CheckPermissions(),
	}

	// Get standard temp directories
	tempDirs := getSystemTempDirs()

	// Add browser cache directories
	browserDirs := getBrowserCacheDirs()

	// Combine all directories to scan
	allDirs := append(tempDirs, browserDirs...)

	for _, dirInfo := range allDirs {
		dirPath := dirInfo.Path
		dirCategory := dirInfo.Location

		if _, exists := result.Files[dirCategory]; !exists {
			result.Files[dirCategory] = []FileInfo{}
		}

		needsElevation := needsElevatedPermissions(dirPath)

		// Skip directories that need elevation if we don't have it
		if needsElevation && !result.Permissions.IsElevated {
			result.Permissions.UnaccessiblePaths = append(result.Permissions.UnaccessiblePaths, dirPath)
			result.Permissions.RequiresElevation = true
			continue
		}

		// Only scan if we have permission or can try
		if hasReadPermission(dirPath) {
			files, size := scanDirectory(dirPath, dirCategory, needsElevation)
			result.Files[dirCategory] = append(result.Files[dirCategory], files...)
			result.TotalSize += size
		} else {
			result.Permissions.UnaccessiblePaths = append(result.Permissions.UnaccessiblePaths, dirPath)
		}
	}

	return result
}

// CleanFiles removes the specified files
func CleanFiles(files []FileInfo) (int, int64, []string) {
	cleanedFiles := 0
	cleanedSize := int64(0)
	failures := []string{}

	for _, file := range files {
		// Skip files that need elevation if we don't have it
		if file.NeedsElevation {
			permissions := CheckPermissions()
			if !permissions.IsElevated {
				failures = append(failures, fmt.Sprintf("Access denied (requires elevation): %s", file.Path))
				continue
			}
		}

		// Skip if the file is less than 1 minute old (safety measure)
		info, err := os.Stat(file.Path)
		if err == nil {
			if time.Since(info.ModTime()) < 1*time.Minute {
				failures = append(failures, fmt.Sprintf("Skipped (recently modified): %s", file.Path))
				continue
			}
		}

		// Skip system critical directories/files
		if isCriticalFile(file.Path) {
			failures = append(failures, fmt.Sprintf("Skipped (critical file): %s", file.Path))
			continue
		}

		// Check write permission
		if !hasWritePermission(file.Path) {
			failures = append(failures, fmt.Sprintf("Access denied (no write permission): %s", file.Path))
			continue
		}

		err = os.RemoveAll(file.Path)
		if err == nil {
			cleanedFiles++
			cleanedSize += file.Size
		} else {
			failures = append(failures, fmt.Sprintf("Failed to remove: %s (Error: %s)", file.Path, err))
		}
	}

	return cleanedFiles, cleanedSize, failures
}

// GetFormattedSize converts bytes to human-readable format
func GetFormattedSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GetElevationCommand returns the command to run the application with elevated privileges
func GetElevationCommand(execPath string) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("Start-Process \"%s\" -Verb RunAs", execPath)
	case "linux":
		return fmt.Sprintf("pkexec %s", execPath)
	case "darwin":
		return fmt.Sprintf("osascript -e 'do shell script \"%s\" with administrator privileges'", execPath)
	default:
		return ""
	}
}

// Helper functions

type DirInfo struct {
	Path     string
	Location string
}

func getSystemTempDirs() []DirInfo {
	var dirs []DirInfo

	switch runtime.GOOS {
	case "windows":
		dirs = []DirInfo{
			{os.TempDir(), "System Temp"},
			{"C:\\Windows\\Temp", "Windows Temp"},
			{filepath.Join(os.Getenv("LOCALAPPDATA"), "Temp"), "User Temp"},
		}
	case "linux":
		dirs = []DirInfo{
			{"/tmp", "System Temp"},
			{"/var/tmp", "System Temp"},
			{filepath.Join(os.Getenv("HOME"), ".cache"), "User Cache"},
		}
	case "darwin":
		dirs = []DirInfo{
			{"/tmp", "System Temp"},
			{"/private/tmp", "System Temp"},
			{filepath.Join(os.Getenv("HOME"), "Library/Caches"), "User Cache"},
		}
	}

	return dirs
}

func getBrowserCacheDirs() []DirInfo {
	var dirs []DirInfo
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("LOCALAPPDATA")
		dirs = []DirInfo{
			{filepath.Join(appData, "Google", "Chrome", "User Data", "Default", "Cache"), "Chrome Cache"},
			{filepath.Join(appData, "Mozilla", "Firefox", "Profiles"), "Firefox Cache"},
			{filepath.Join(appData, "Microsoft", "Edge", "User Data", "Default", "Cache"), "Edge Cache"},
		}
	case "linux":
		dirs = []DirInfo{
			{filepath.Join(home, ".cache", "google-chrome"), "Chrome Cache"},
			{filepath.Join(home, ".mozilla", "firefox"), "Firefox Cache"},
			{filepath.Join(home, ".config", "microsoft-edge", "Default", "Cache"), "Edge Cache"},
		}
	case "darwin":
		dirs = []DirInfo{
			{filepath.Join(home, "Library", "Caches", "Google", "Chrome"), "Chrome Cache"},
			{filepath.Join(home, "Library", "Caches", "Firefox"), "Firefox Cache"},
			{filepath.Join(home, "Library", "Caches", "Microsoft Edge"), "Edge Cache"},
		}
	}

	return dirs
}

func scanDirectory(dirPath, category string, needsElevation bool) ([]FileInfo, int64) {
	var files []FileInfo
	var totalSize int64

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return files, 0
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())

		// Skip system critical files
		if isCriticalFile(fullPath) {
			continue
		}

		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		// Skip files being used by the system
		if time.Since(info.ModTime()) < 1*time.Minute {
			continue
		}

		size := int64(0)
		if info.IsDir() {
			// Calculate directory size
			filepath.Walk(fullPath, func(_ string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					size += info.Size()
				}
				return nil
			})
		} else {
			size = info.Size()
		}

		files = append(files, FileInfo{
			Path:           fullPath,
			Size:           size,
			Name:           entry.Name(),
			Location:       category,
			NeedsElevation: needsElevation,
		})
		totalSize += size
	}

	return files, totalSize
}

func isCriticalFile(path string) bool {
	// List of critical directories/files to avoid
	criticalPatterns := []string{
		"System Volume Information",
		"$Recycle.Bin",
		"pagefile.sys",
		"hiberfil.sys",
		"swapfile.sys",
		"/proc",
		"/sys",
		"/dev",
		"/boot",
	}

	for _, pattern := range criticalPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

func hasReadPermission(path string) bool {
	_, err := os.ReadDir(path)
	return err == nil
}

func hasWritePermission(path string) bool {
	// Check if file exists first
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// For directories, try to create a test file
	if info.IsDir() {
		testFile := filepath.Join(path, ".permission_test")
		f, err := os.Create(testFile)
		if err != nil {
			return false
		}
		f.Close()
		os.Remove(testFile)
		return true
	}

	// For files, check if we can open for writing
	f, err := os.OpenFile(path, os.O_WRONLY, 0666)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func CheckPermissions() PermissionStatus {
	status := PermissionStatus{
		IsElevated:        false,
		RequiresElevation: false,
		UnaccessiblePaths: []string{},
		CanCleanUserFiles: true,
	}

	// Check if we're running with elevated privileges
	switch runtime.GOOS {
	case "windows":
		// Check access to Windows\System32\config which requires admin
		status.IsElevated = hasWritePermission("C:\\Windows\\System32\\config")
		status.ElevationCommand = "Right-click app and select 'Run as administrator'"
	case "linux", "darwin":
		// Check if we're root
		status.IsElevated = os.Geteuid() == 0
		if runtime.GOOS == "linux" {
			status.ElevationCommand = "Run with 'sudo' or use pkexec"
		} else {
			status.ElevationCommand = "Run with 'sudo'"
		}
	}

	// Test if we can access user directories
	userTemp := os.TempDir()
	status.CanCleanUserFiles = hasWritePermission(userTemp)

	// Test if we can access system directories
	var systemPath string
	switch runtime.GOOS {
	case "windows":
		systemPath = "C:\\Windows\\Temp"
	case "linux", "darwin":
		systemPath = "/var/tmp"
	}

	status.CanCleanSystemFiles = hasWritePermission(systemPath)

	return status
}

func needsElevatedPermissions(path string) bool {
	switch runtime.GOOS {
	case "windows":
		return strings.HasPrefix(strings.ToLower(path), "c:\\windows")
	case "linux", "darwin":
		// Paths that typically need root
		systemPaths := []string{
			"/var",
			"/etc",
			"/usr",
			"/opt",
			"/private/var",
		}

		for _, syspath := range systemPaths {
			if strings.HasPrefix(path, syspath) {
				return true
			}
		}
	}

	return false
}

// GetUserFriendlyDirectories returns a list of directories that can be cleaned without elevation
func GetUserFriendlyDirectories() []DirInfo {
	var result []DirInfo

	// Always include user-specific directories
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		// home = os.Getenv("USERPROFILE")
		appData := os.Getenv("LOCALAPPDATA")

		result = append(result, DirInfo{
			Path:     filepath.Join(appData, "Temp"),
			Location: "User Temp",
		})

		// Browser caches
		result = append(result, DirInfo{
			Path:     filepath.Join(appData, "Google", "Chrome", "User Data", "Default", "Cache"),
			Location: "Chrome Cache",
		})
		result = append(result, DirInfo{
			Path:     filepath.Join(appData, "Mozilla", "Firefox", "Profiles"),
			Location: "Firefox Cache",
		})
		result = append(result, DirInfo{
			Path:     filepath.Join(appData, "Microsoft", "Edge", "User Data", "Default", "Cache"),
			Location: "Edge Cache",
		})

	} else if runtime.GOOS == "linux" {
		result = append(result, DirInfo{
			Path:     filepath.Join(home, ".cache"),
			Location: "User Cache",
		})

		// Browser caches
		result = append(result, DirInfo{
			Path:     filepath.Join(home, ".cache", "google-chrome"),
			Location: "Chrome Cache",
		})
		result = append(result, DirInfo{
			Path:     filepath.Join(home, ".mozilla", "firefox"),
			Location: "Firefox Cache",
		})

	} else if runtime.GOOS == "darwin" {
		result = append(result, DirInfo{
			Path:     filepath.Join(home, "Library", "Caches"),
			Location: "User Cache",
		})

		// Browser caches
		result = append(result, DirInfo{
			Path:     filepath.Join(home, "Library", "Caches", "Google", "Chrome"),
			Location: "Chrome Cache",
		})
		result = append(result, DirInfo{
			Path:     filepath.Join(home, "Library", "Caches", "Firefox"),
			Location: "Firefox Cache",
		})
		result = append(result, DirInfo{
			Path:     filepath.Join(home, "Library", "Caches", "Microsoft Edge"),
			Location: "Edge Cache",
		})
	}

	// Filter to only include directories that actually exist and we have permission for
	var filteredResult []DirInfo
	for _, dir := range result {
		if _, err := os.Stat(dir.Path); err == nil {
			if hasReadPermission(dir.Path) {
				filteredResult = append(filteredResult, dir)
			}
		}
	}

	return filteredResult
}

// SafeClean only cleans directories that don't require elevated permissions
func SafeClean() CleanerResult {
	result := CleanerResult{
		Files:       make(map[string][]FileInfo),
		TotalSize:   0,
		Permissions: CheckPermissions(),
	}

	// Only get directories that are safe to clean
	userDirs := GetUserFriendlyDirectories()

	for _, dirInfo := range userDirs {
		dirPath := dirInfo.Path
		dirCategory := dirInfo.Location

		if _, exists := result.Files[dirCategory]; !exists {
			result.Files[dirCategory] = []FileInfo{}
		}

		// We already filtered for permission, so just scan
		files, size := scanDirectory(dirPath, dirCategory, false) // Never needs elevation
		result.Files[dirCategory] = append(result.Files[dirCategory], files...)
		result.TotalSize += size
	}

	return result
}

// PurgeJunk scans user & system temp/cache directories and actually removes cleanable junk files from disk
func PurgeJunk() (int, int64, []string) {
	scanResult := SafeClean()
	var allFiles []FileInfo
	for _, fileList := range scanResult.Files {
		allFiles = append(allFiles, fileList...)
	}

	return CleanFiles(allFiles)
}

