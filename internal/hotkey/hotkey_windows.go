//go:build windows

package hotkey

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessage       = user32.NewProc("GetMessageW")
	procPostThreadMsg    = user32.NewProc("PostThreadMessageW")
)

const (
	wmHotkey    = 0x0312
	wmQuit      = 0x0012
	modNoRepeat = 0x4000
	hotkeyID    = 1
)

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

// Listener owns one registered shortcut. A hotkey belongs to the thread that
// registered it, so each Listener runs its own locked thread with a message loop.
type Listener struct {
	threadID uint32
	done     chan struct{}
}

// ErrInUse means another program already owns the shortcut
var ErrInUse = errors.New("another app is already using this shortcut")

// Listen registers combo and calls onPress on every press until Stop is called
func Listen(combo string, onPress func()) (*Listener, error) {
	mods, vk, err := parseCombo(combo)
	if err != nil {
		return nil, err
	}

	l := &Listener{done: make(chan struct{})}
	ready := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(l.done)

		l.threadID = windows.GetCurrentThreadId()
		ok, _, callErr := procRegisterHotKey.Call(0, hotkeyID, uintptr(mods|modNoRepeat), uintptr(vk))
		if ok == 0 {
			if errors.Is(callErr, windows.Errno(1409)) { // ERROR_HOTKEY_ALREADY_REGISTERED
				ready <- ErrInUse
			} else {
				ready <- fmt.Errorf("could not register %s: %v", combo, callErr)
			}
			return
		}
		defer procUnregisterHotKey.Call(0, hotkeyID)
		ready <- nil

		var m msg
		for {
			// GetMessage returns 0 on WM_QUIT and -1 on error
			r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(r) <= 0 {
				return
			}
			if m.message == wmHotkey && m.wParam == hotkeyID {
				onPress()
			}
		}
	}()

	if err := <-ready; err != nil {
		<-l.done
		return nil, err
	}
	return l, nil
}

// Stop releases the shortcut
func (l *Listener) Stop() {
	if l == nil {
		return
	}
	procPostThreadMsg.Call(uintptr(l.threadID), wmQuit, 0, 0)
	<-l.done
}
