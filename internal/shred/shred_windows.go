//go:build windows

package shred

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

const (
	attrRecallOnOpen       = 0x00040000
	attrRecallOnDataAccess = 0x00400000
)

// shredFile overwrites one file pass by pass, flushing each pass to the disk
// so the passes are not merged in Windows' write cache, then empties it,
// gives it a meaningless name and old timestamps, and deletes it.
func shredFile(ctx context.Context, path string, method []pattern, wrote func(pass int, n int64)) (note string, err error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", errors.New("the name could not be read")
	}
	attrs, err := windows.GetFileAttributes(name)
	if err != nil {
		return "", errors.New("not found")
	}
	switch {
	case attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0:
		return "", errors.New("a link, which is never followed")
	case attrs&(attrRecallOnOpen|attrRecallOnDataAccess|windows.FILE_ATTRIBUTE_OFFLINE) != 0:
		return "", errors.New("only stored online (OneDrive or similar); delete it there and empty its recycle bin")
	}
	if attrs&windows.FILE_ATTRIBUTE_READONLY != 0 {
		if err := windows.SetFileAttributes(name, attrs&^windows.FILE_ATTRIBUTE_READONLY); err != nil {
			return "", errors.New("read-only, and Windows does not allow changing it")
		}
	}
	switch {
	case attrs&windows.FILE_ATTRIBUTE_COMPRESSED != 0:
		note = "compressed by Windows, so the overwrite may not reach all of its old data"
	case attrs&windows.FILE_ATTRIBUTE_SPARSE_FILE != 0:
		note = "a sparse file, so the overwrite may not reach all of its old data"
	case attrs&windows.FILE_ATTRIBUTE_ENCRYPTED != 0:
		note = "encrypted with EFS; its old contents were encrypted too"
	}

	// no sharing: if another program has it open, leave it alone
	h, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		if errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return "", errors.New("open in another program; close it and try again")
		}
		return "", errors.New("Windows does not allow changing this file")
	}
	f := os.NewFile(uintptr(h), path)
	closed := false
	defer func() {
		if !closed {
			f.Close()
		}
	}()

	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		return "", errors.New("could not be read")
	}
	if info.NumberOfLinks > 1 {
		return "", errors.New("also saved under another name (a hard link); shredding it would destroy that copy too")
	}
	size := int64(info.FileSizeHigh)<<32 | int64(info.FileSizeLow)

	buf := make([]byte, chunk)
	for pass, p := range method {
		if _, err := f.Seek(0, 0); err != nil {
			return "", errors.New("could not be overwritten")
		}
		for left := size; left > 0; {
			if ctx.Err() != nil {
				return "", context.Canceled
			}
			n := int64(len(buf))
			if left < n {
				n = left
			}
			if err := fill(buf[:n], p); err != nil {
				return "", errors.New("could not make random data")
			}
			if _, err := f.Write(buf[:n]); err != nil {
				return "", errors.New("could not be overwritten (is the drive full or failing?)")
			}
			left -= n
			wrote(pass+1, n)
		}
		if err := f.Sync(); err != nil {
			return "", errors.New("the overwrite could not be written to the drive")
		}
	}

	// empty it and set its times to 1 January 2000, so neither the size nor
	// when it was used survives in the file system's records
	if err := f.Truncate(0); err == nil {
		f.Sync()
	}
	old := windows.NsecToFiletime(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano())
	windows.SetFileTime(h, &old, &old, &old)
	f.Close()
	closed = true

	// a meaningless name, so the old one does not stay in the folder's records
	gone := filepath.Join(filepath.Dir(path), randomName(filepath.Base(path)))
	if err := os.Rename(path, gone); err != nil {
		gone = path
	}
	if err := os.Remove(gone); err != nil {
		return note, errors.New("overwritten, but could not be deleted")
	}
	return note, nil
}
