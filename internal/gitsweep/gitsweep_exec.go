package gitsweep

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// gitPath finds git: on PATH, or where its installers put it
func gitPath() string {
	if p, err := exec.LookPath("git"); err == nil {
		return p
	}
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "cmd", "git.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Git", "cmd", "git.exe"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

func gitRunner(git string) runner {
	return func(ctx context.Context, dir string, args ...string) (string, error) {
		c, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		// read-only: trust every folder (only for reading), take no locks, never ask
		// for a password, and leave file-system monitors alone
		full := append([]string{"-C", dir, "-c", "safe.directory=*", "-c", "core.fsmonitor=false", "--no-optional-locks"}, args...)
		cmd := exec.CommandContext(c, git, full...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
		hideWindow(cmd)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
}
