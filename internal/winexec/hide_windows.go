//go:build windows

package winexec

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// Hide stops a console window appearing for cmd
func Hide(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}
