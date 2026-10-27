//go:build !windows

package vdisk

func registeredDistros() []distro { return nil }

func allocated(path string, size int64) (int64, bool) { return size, false }
