//go:build !windows

package winget

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
