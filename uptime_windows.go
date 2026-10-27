//go:build windows

package main

import "golang.org/x/sys/windows"

var procGetTickCount64 = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetTickCount64")

// windowsTickMs is the time since Windows started, in milliseconds
func windowsTickMs() uint64 {
	r, _, _ := procGetTickCount64.Call()
	return uint64(r)
}
