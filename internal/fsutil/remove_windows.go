//go:build windows

package fsutil

import (
	"errors"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// removeFile deletes one file or link in a single step: it opens the file for
// deletion and marks it deleted, ignoring the read-only flag (Windows 10 1809
// and later, on NTFS). That is one open instead of the five calls os.Remove
// makes for a read-only file, and each of those is checked by antivirus, so
// it matters a lot for caches of many small read-only files, like Go's. Links
// and junctions are opened as themselves, never followed. Anything this
// cannot do (older Windows, FAT drives, network shares) goes the usual way.
func removeFile(path string) error {
	name, err := windows.UTF16PtrFromString(longPath(path))
	if err != nil {
		return RemoveOne(path)
	}
	h, err := windows.CreateFile(name, windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
			return nil
		}
		return RemoveOne(path)
	}
	info := struct{ flags uint32 }{windows.FILE_DISPOSITION_DELETE |
		windows.FILE_DISPOSITION_POSIX_SEMANTICS |
		windows.FILE_DISPOSITION_IGNORE_READONLY_ATTRIBUTE}
	err = windows.SetFileInformationByHandle(h, windows.FileDispositionInfoEx,
		(*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	windows.CloseHandle(h)
	if err == nil {
		return nil
	}
	return RemoveOne(path)
}

// longPath lets paths longer than 260 characters through, as os functions do
func longPath(path string) string {
	if len(path) < 248 || strings.HasPrefix(path, `\\`) || !filepath.IsAbs(path) {
		return path
	}
	return `\\?\` + filepath.Clean(path)
}
