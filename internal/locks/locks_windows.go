//go:build windows

package locks

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows Restart Manager: the supported way to ask "who has this file open?"
var (
	rstrtmgr                = windows.NewLazySystemDLL("rstrtmgr.dll")
	procRmStartSession      = rstrtmgr.NewProc("RmStartSession")
	procRmRegisterResources = rstrtmgr.NewProc("RmRegisterResources")
	procRmGetList           = rstrtmgr.NewProc("RmGetList")
	procRmEndSession        = rstrtmgr.NewProc("RmEndSession")
)

const (
	errorMoreData   = 234
	rmSessionKeyLen = 33 // CCH_RM_SESSION_KEY + 1
)

type rmUniqueProcess struct {
	ProcessID        uint32
	ProcessStartTime windows.Filetime
}

type rmProcessInfo struct {
	Process          rmUniqueProcess
	AppName          [256]uint16
	ServiceShortName [64]uint16
	ApplicationType  uint32
	AppStatus        uint32
	TSSessionID      uint32
	Restartable      int32
}

var appKinds = map[uint32]string{
	0:    "Unknown",
	1:    "Application",
	2:    "Application",
	3:    "Service",
	4:    "Explorer",
	5:    "Console program",
	1000: "System critical",
}

// Find lists the processes holding path open. For a folder it checks the files inside.
func Find(path string) (*Result, error) {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot find %s: %w", path, err)
	}

	result := &Result{Path: path, IsDir: info.IsDir(), Processes: []LockingProcess{}}
	files := []string{path}
	if info.IsDir() {
		files, result.Truncated = filesUnder(path)
	}
	result.FilesChecked = len(files)
	if len(files) == 0 {
		return result, nil
	}

	var session uint32
	var sessionKey [rmSessionKeyLen]uint16
	if ret, _, _ := procRmStartSession.Call(
		uintptr(unsafe.Pointer(&session)), 0, uintptr(unsafe.Pointer(&sessionKey[0]))); ret != 0 {
		return nil, fmt.Errorf("could not start a lock query (error %d)", ret)
	}
	defer procRmEndSession.Call(uintptr(session))

	names := make([]*uint16, 0, len(files))
	for _, f := range files {
		if p, err := windows.UTF16PtrFromString(f); err == nil {
			names = append(names, p)
		}
	}
	if ret, _, _ := procRmRegisterResources.Call(
		uintptr(session), uintptr(len(names)), uintptr(unsafe.Pointer(&names[0])), 0, 0, 0, 0); ret != 0 {
		return nil, fmt.Errorf("could not check %s (error %d)", path, ret)
	}

	// the list can grow between the size query and the fetch, so retry a few times
	var needed, count, reasons uint32
	var infos []rmProcessInfo
	for attempt := 0; attempt < 4; attempt++ {
		var first unsafe.Pointer
		if len(infos) > 0 {
			first = unsafe.Pointer(&infos[0])
		}
		count = uint32(len(infos))
		ret, _, _ := procRmGetList.Call(uintptr(session),
			uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)),
			uintptr(first), uintptr(unsafe.Pointer(&reasons)))
		if ret == 0 {
			infos = infos[:count]
			break
		}
		if ret != errorMoreData {
			return nil, fmt.Errorf("could not list locking processes (error %d)", ret)
		}
		infos = make([]rmProcessInfo, needed)
	}

	for _, p := range infos {
		kind, ok := appKinds[p.ApplicationType]
		if !ok {
			kind = "Unknown"
		}
		result.Processes = append(result.Processes, LockingProcess{
			PID:  p.Process.ProcessID,
			Name: windows.UTF16ToString(p.AppName[:]),
			Kind: kind,
		})
	}
	sort.Slice(result.Processes, func(i, j int) bool { return result.Processes[i].Name < result.Processes[j].Name })
	return result, nil
}

// filesUnder lists regular files below dir, up to the check limit
func filesUnder(dir string) (files []string, truncated bool) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if len(files) >= maxFilesChecked {
			truncated = true
			return filepath.SkipAll
		}
		files = append(files, path)
		return nil
	})
	return files, truncated
}
