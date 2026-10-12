//go:build windows

package storage

import (
	"os/exec"
	"syscall"
)

func revealCommand(path string) *exec.Cmd {
	cmd := exec.Command("explorer.exe")
	// Explorer parses its own command line: the path must be quoted after "/select,"
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + path + `"`}
	return cmd
}
