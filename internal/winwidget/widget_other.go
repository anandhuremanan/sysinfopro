//go:build !windows

package winwidget

// SetActive is only implemented on Windows
func SetActive(title string, active bool) {}
