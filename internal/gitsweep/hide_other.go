//go:build !windows

package gitsweep

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
