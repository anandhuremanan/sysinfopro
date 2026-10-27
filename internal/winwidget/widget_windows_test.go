//go:build windows

package winwidget

import "testing"

// Windows allows about 2000 Go callbacks per process; searching more often
// than that must not create new ones (it used to, and would then crash)
func TestWindowSearchReusesOneCallback(t *testing.T) {
	for i := 0; i < 3000; i++ {
		if hwnd := mainWindow("no window has this title 9f2c"); hwnd != 0 {
			t.Fatalf("found a window that should not exist: %v", hwnd)
		}
	}
}
