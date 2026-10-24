//go:build !windows

package vdisk

import "os/exec"

func registeredDistros() []distro { return nil }

func allocated(path string, size int64) (int64, bool) { return size, false }

func hideWindow(cmd *exec.Cmd) {}
