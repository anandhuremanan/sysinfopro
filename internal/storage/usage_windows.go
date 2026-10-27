//go:build windows

package storage

import (
	"io/fs"
	"syscall"
)

const (
	attrOffline        = 0x1000
	attrRecallOnOpen   = 0x40000
	attrRecallOnAccess = 0x400000
)

// onDisk is the space a file takes on this drive. OneDrive and other cloud
// files that are only online show their full size but take none until opened.
func onDisk(info fs.FileInfo) int64 {
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok &&
		d.FileAttributes&(attrOffline|attrRecallOnOpen|attrRecallOnAccess) != 0 {
		return 0
	}
	return info.Size()
}
