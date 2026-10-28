//go:build windows

package appdata

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

// isFixedDrive is true for a folder on an internal drive (not removable,
// network, optical or RAM disk)
func isFixedDrive(path string) bool {
	vol := filepath.VolumeName(path)
	if vol == "" {
		return false
	}
	root, err := windows.UTF16PtrFromString(vol + `\`)
	if err != nil {
		return false
	}
	return windows.GetDriveType(root) == windows.DRIVE_FIXED
}

// freeSpace is the space the current user can use on path's drive
func freeSpace(path string) (uint64, bool) {
	root, err := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	if err != nil {
		return 0, false
	}
	var free uint64
	if windows.GetDiskFreeSpaceEx(root, &free, nil, nil) != nil {
		return 0, false
	}
	return free, true
}
