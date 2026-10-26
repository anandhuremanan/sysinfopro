//go:build windows

package dupes

import (
	"io/fs"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	attrOffline        = 0x1000
	attrRecallOnOpen   = 0x40000
	attrRecallOnAccess = 0x400000
)

// isCloudOnly is true for OneDrive-style files whose contents are not on this
// PC; reading one would download it
func isCloudOnly(info fs.FileInfo) bool {
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return d.FileAttributes&(attrOffline|attrRecallOnOpen|attrRecallOnAccess) != 0
	}
	return false
}

// UserFolders lists the personal folders where duplicates usually pile up,
// wherever they really are (Documents and Pictures may be in OneDrive)
func UserFolders() []Place {
	known := []struct {
		label string
		id    *windows.KNOWNFOLDERID
	}{
		{"Downloads", windows.FOLDERID_Downloads},
		{"Documents", windows.FOLDERID_Documents},
		{"Pictures", windows.FOLDERID_Pictures},
		{"Videos", windows.FOLDERID_Videos},
		{"Music", windows.FOLDERID_Music},
		{"Desktop", windows.FOLDERID_Desktop},
	}
	places := []Place{}
	for _, k := range known {
		path, err := windows.KnownFolderPath(k.id, 0)
		if err != nil {
			continue
		}
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			places = append(places, Place{Label: k.label, Path: path})
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		places = append(places, Place{Label: "User folder", Path: home})
	}
	return places
}

// SHFILEOPSTRUCTW, 64-bit layout
type shFileOp struct {
	hwnd          uintptr
	wFunc         uint32
	from          *uint16
	to            *uint16
	flags         uint16
	anyAborted    int32
	nameMappings  uintptr
	progressTitle *uint16
}

var shFileOperation = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

// ToRecycleBin moves one file to the Recycle Bin, so it can be restored. Where a
// drive has no Recycle Bin, Windows asks before deleting it for good.
func ToRecycleBin(path string) error {
	const (
		foDelete          = 3
		fofSilent         = 0x0004
		fofNoConfirmation = 0x0010
		fofAllowUndo      = 0x0040
		fofNoErrorUI      = 0x0400
		fofWantNukeWarn   = 0x4000
	)
	// a list of paths, each ending in a zero, and one more zero at the end
	from, err := windows.UTF16FromString(path)
	if err != nil {
		return err
	}
	from = append(from, 0)
	op := shFileOp{
		wFunc: foDelete,
		from:  &from[0],
		flags: fofSilent | fofNoConfirmation | fofAllowUndo | fofNoErrorUI | fofWantNukeWarn,
	}
	r, _, _ := shFileOperation.Call(uintptr(unsafe.Pointer(&op)))
	if r != 0 {
		return syscall.Errno(r)
	}
	if op.anyAborted != 0 {
		return windows.ERROR_CANCELLED
	}
	return nil
}
