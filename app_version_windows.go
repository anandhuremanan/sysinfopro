//go:build windows

package main

import (
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// windowsRelease reads the release names from the registry. Windows 11 still
// reports "Windows 10" as its product name, so the build number decides.
func windowsRelease() (name, display, build string, ubr uint64) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "", "", "", 0
	}
	defer k.Close()
	name, _, _ = k.GetStringValue("ProductName")
	display, _, _ = k.GetStringValue("DisplayVersion")
	build, _, _ = k.GetStringValue("CurrentBuildNumber")
	ubr, _, _ = k.GetIntegerValue("UBR")
	if n, err := strconv.Atoi(build); err == nil && n >= 22000 {
		name = strings.Replace(name, "Windows 10", "Windows 11", 1)
	}
	return name, display, build, ubr
}
