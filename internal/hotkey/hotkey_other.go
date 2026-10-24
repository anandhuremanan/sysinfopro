//go:build !windows

package hotkey

import "errors"

var ErrInUse = errors.New("another app is already using this shortcut")

type Listener struct{}

// Listen is only implemented on Windows
func Listen(combo string, onPress func()) (*Listener, error) {
	return nil, errors.New("keyboard shortcuts are not supported on this platform")
}

func (l *Listener) Stop() {}
