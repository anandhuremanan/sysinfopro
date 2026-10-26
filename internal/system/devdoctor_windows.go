//go:build windows

package system

import (
	"context"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

// storedVar reads a variable as new terminals will see it: the user's value,
// else the system's, with %VARIABLES% expanded
func storedVar(name string) string {
	if v, err := readValue(registry.CURRENT_USER, userEnvKey, name); err == nil && v != "" {
		return expandEnv(v)
	}
	if v, err := readValue(registry.LOCAL_MACHINE, systemEnvKey, name); err == nil && v != "" {
		return expandEnv(v)
	}
	return ""
}

// runQuiet runs a program without a console window, for at most 8 seconds
func runQuiet(exe string, args []string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}
