//go:build windows

package relocate

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// createJunction makes link a folder link (junction) to target. Unlike
// symbolic links, junctions need no administrator rights or Developer Mode.
func createJunction(link, target string) error {
	target, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if err := os.Mkdir(link, 0o755); err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			os.Remove(link)
		}
	}()

	name, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(name, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	// REPARSE_DATA_BUFFER for a mount point: the "\??\" name Windows follows,
	// then the plain name tools show, each ending in a NUL
	substitute := utf16.Encode([]rune(`\??\` + target))
	shown := utf16.Encode([]rune(target))
	names := append(append(append(substitute, 0), shown...), 0)
	dataLen := 8 + len(names)*2
	buf := make([]byte, 8+dataLen)
	binary.LittleEndian.PutUint32(buf[0:], windows.IO_REPARSE_TAG_MOUNT_POINT)
	binary.LittleEndian.PutUint16(buf[4:], uint16(dataLen))
	binary.LittleEndian.PutUint16(buf[8:], 0)
	binary.LittleEndian.PutUint16(buf[10:], uint16(len(substitute)*2))
	binary.LittleEndian.PutUint16(buf[12:], uint16((len(substitute)+1)*2))
	binary.LittleEndian.PutUint16(buf[14:], uint16(len(shown)*2))
	for i, c := range names {
		binary.LittleEndian.PutUint16(buf[16+i*2:], c)
	}
	var returned uint32
	if err := windows.DeviceIoControl(h, windows.FSCTL_SET_REPARSE_POINT, &buf[0], uint32(len(buf)), nil, 0, &returned, nil); err != nil {
		return err
	}
	ok = true
	return nil
}

// junctionTarget reports where a folder link points, if path is one
func junctionTarget(path string) (string, bool) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", false
	}
	attrs, err := windows.GetFileAttributes(name)
	if err != nil || attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
		return "", false
	}
	target, err := os.Readlink(path)
	if err != nil {
		return "", true
	}
	return strings.TrimPrefix(target, `\??\`), true
}

// runningPrograms lists the file names of running programs, in lower case
func runningPrograms() map[string]bool {
	names := map[string]bool{}
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return names
	}
	defer windows.CloseHandle(snap)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(snap, &entry); err == nil; err = windows.Process32Next(snap, &entry) {
		names[strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))] = true
	}
	return names
}
