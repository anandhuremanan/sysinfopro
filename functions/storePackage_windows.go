//go:build windows

package functions

import (
	"syscall"
	"unsafe"
)

const appModelErrorNoPackage = 15700

// IsStorePackage reports whether the app is running from an installed package
// (the Microsoft Store build). A plain .exe has no package identity.
func IsStorePackage() bool {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentPackageFullName")
	if proc.Find() != nil {
		return false // API missing: Windows older than 8, never packaged
	}
	var length uint32
	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&length)), 0)
	return ret != appModelErrorNoPackage
}
