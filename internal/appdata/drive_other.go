//go:build !windows

package appdata

func isFixedDrive(path string) bool { return true }

func freeSpace(path string) (uint64, bool) { return 0, false }
