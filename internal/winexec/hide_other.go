//go:build !windows

package winexec

import "os/exec"

// Hide does nothing outside Windows
func Hide(cmd *exec.Cmd) {}
