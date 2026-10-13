//go:build !windows

package devcache

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
