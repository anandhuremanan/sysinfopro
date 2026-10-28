//go:build !windows

package fsutil

func removeFile(path string) error { return RemoveOne(path) }
