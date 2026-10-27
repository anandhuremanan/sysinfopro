//go:build windows

package main

import "golang.org/x/sys/windows"

// knownDownloads asks Windows where the Downloads folder really is
func knownDownloads() string {
	dir, err := windows.KnownFolderPath(windows.FOLDERID_Downloads, 0)
	if err != nil {
		return ""
	}
	return dir
}
