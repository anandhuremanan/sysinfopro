//go:build windows

package devcache

import (
	"os/exec"
	"syscall"
)

// hideWindow stops a console window flashing when a command-line tool is run
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
