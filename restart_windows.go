//go:build windows

package main

import (
	"os"
	"strconv"

	"golang.org/x/sys/windows"
)

// restartFlag tells a new copy of the app which process it replaces
const restartFlag = "--after-restart"

// waitForRestart waits, up to 15 s, for the copy of the app that started this
// one with restartFlag to exit
func waitForRestart() {
	for i, arg := range os.Args {
		if arg != restartFlag || i+1 >= len(os.Args) {
			continue
		}
		pid, err := strconv.ParseUint(os.Args[i+1], 10, 32)
		if err != nil {
			return
		}
		h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
		if err != nil {
			return // already gone
		}
		windows.WaitForSingleObject(h, 15000)
		windows.CloseHandle(h)
		return
	}
}
