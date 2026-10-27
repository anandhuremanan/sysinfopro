package devcache

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"sysinfopro/internal/winexec"
)

// goEnvDirs asks the Go toolchain where its module and build caches really are.
// These are often moved off the system drive with `go env -w`, which environment
// variables alone do not reveal. Returns empty strings when Go is not installed.
func goEnvDirs() (modCache, buildCache string) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "env", "GOMODCACHE", "GOCACHE")
	winexec.Hide(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(string(out)), "\r", ""), "\n")
	if len(lines) != 2 {
		return "", ""
	}
	return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1])
}
