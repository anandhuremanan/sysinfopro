//go:build !windows

package fsutil

func IsFixedDrive(path string) bool { return true }

func FreeSpace(path string) (uint64, bool) { return 0, false }
