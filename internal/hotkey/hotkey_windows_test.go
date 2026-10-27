//go:build windows && integration

package hotkey

import (
	"errors"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// keyboard input, as SendInput expects it on 64-bit Windows
type keyInput struct {
	kind  uint32
	_     uint32
	vk    uint16
	scan  uint16
	flags uint32
	time  uint32
	extra uintptr
	_     [8]byte
}

func press(t *testing.T, vks ...uint16) {
	t.Helper()
	var inputs []keyInput
	for _, vk := range vks {
		inputs = append(inputs, keyInput{kind: 1, vk: vk})
	}
	for i := len(vks) - 1; i >= 0; i-- {
		inputs = append(inputs, keyInput{kind: 1, vk: vks[i], flags: 0x0002}) // key up
	}
	sendInput := windows.NewLazySystemDLL("user32.dll").NewProc("SendInput")
	n, _, err := sendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
	if int(n) != len(inputs) {
		t.Skipf("cannot simulate key presses here: %v", err)
	}
}

func TestListenAndInUse(t *testing.T) {
	const combo = "Ctrl+Alt+Shift+9"
	pressed := make(chan struct{}, 1)
	l, err := Listen(combo, func() { pressed <- struct{}{} })
	if err != nil {
		t.Skipf("cannot register %s here: %v", combo, err)
	}
	defer l.Stop()

	if _, err := Listen(combo, func() {}); !errors.Is(err, ErrInUse) {
		t.Errorf("registering the same shortcut twice: got %v, want ErrInUse", err)
	}

	press(t, 0x11, 0x12, 0x10, '9') // Ctrl, Alt, Shift, 9
	select {
	case <-pressed:
	case <-time.After(3 * time.Second):
		t.Fatal("the shortcut was pressed but no press was reported")
	}

	l.Stop()
	// once stopped, the shortcut is free again
	again, err := Listen(combo, func() {})
	if err != nil {
		t.Fatalf("after Stop the shortcut should be free: %v", err)
	}
	again.Stop()
}
