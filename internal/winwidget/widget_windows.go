//go:build windows

// Package winwidget keeps the app window visible while it is shrunk to a widget.
// "Show desktop" (Win+D) minimises every window that can be minimised, including
// always-on-top ones, so the widget gives up its minimise ability while active and
// is put back if something minimises it anyway.
package winwidget

import (
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procGetWindowLongPtr = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtr = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procIsIconic         = user32.NewProc("IsIconic")
	procShowWindow       = user32.NewProc("ShowWindow")
	procGetWindowTextW   = user32.NewProc("GetWindowTextW")
)

const (
	gwlStyle      = ^uintptr(15) // GWL_STYLE (-16)
	wsMinimizeBox = 0x00020000

	hwndTopmost = ^uintptr(0) // HWND_TOPMOST (-1)

	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020

	swShowNoActivate = 4

	watchEvery = 400 * time.Millisecond
)

var (
	mu   sync.Mutex
	stop chan struct{}
)

// mainWindow finds this process's visible top-level window with the given title
func mainWindow(title string) windows.HWND {
	var found windows.HWND
	pid := windows.GetCurrentProcessId()

	callback := windows.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
		var owner uint32
		windows.GetWindowThreadProcessId(hwnd, &owner)
		if owner != pid || !windows.IsWindowVisible(hwnd) {
			return 1 // keep looking
		}
		buf := make([]uint16, 128)
		procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if windows.UTF16ToString(buf) == title {
			found = hwnd
			return 0 // stop
		}
		return 1
	})
	windows.EnumWindows(callback, nil)
	return found
}

func setMinimizable(hwnd windows.HWND, allowed bool) {
	style, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), gwlStyle)
	if allowed {
		style |= wsMinimizeBox
	} else {
		style &^= wsMinimizeBox
	}
	procSetWindowLongPtr.Call(uintptr(hwnd), gwlStyle, style)
	// tell Windows the frame style changed, without moving or activating the window
	procSetWindowPos.Call(uintptr(hwnd), 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
}

func keepOnTop(hwnd windows.HWND) {
	if minimized, _, _ := procIsIconic.Call(uintptr(hwnd)); minimized != 0 {
		procShowWindow.Call(uintptr(hwnd), swShowNoActivate)
	}
	procSetWindowPos.Call(uintptr(hwnd), hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
}

// SetActive turns widget behaviour on or off for the window with the given title
func SetActive(title string, active bool) {
	mu.Lock()
	defer mu.Unlock()

	if stop != nil {
		close(stop)
		stop = nil
	}

	hwnd := mainWindow(title)
	if hwnd == 0 {
		return
	}
	setMinimizable(hwnd, !active)
	if !active {
		return
	}

	done := make(chan struct{})
	stop = done
	go func() {
		ticker := time.NewTicker(watchEvery)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				keepOnTop(hwnd)
			}
		}
	}()
}
