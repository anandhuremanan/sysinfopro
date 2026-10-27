//go:build windows

package system

import (
	"context"
	"os/exec"
	"time"

	"golang.org/x/sys/windows/registry"

	"sysinfopro/internal/winexec"
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
	winexec.Hide(cmd)
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}
