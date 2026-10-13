//go:build !windows

package storage

// DiskType is only implemented on Windows
func DiskType(mount string) string {
	return ""
}
