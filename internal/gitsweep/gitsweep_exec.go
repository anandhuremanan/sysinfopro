package gitsweep

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"sysinfopro/internal/winexec"
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
		// read-only: take no locks, never ask for a password, and pin the settings
		// that could make git start other programs while reading status or history.
		// Git's own ownership check stays on: a repo owned by someone else is
		// reported as unreadable rather than opened.
		full := append([]string{"-C", dir,
			"-c", "core.fsmonitor=false",
			"-c", "core.hooksPath=NUL",
			"-c", "log.showSignature=false",
			"-c", "diff.external=",
			"-c", "protocol.allow=never",
			"--no-optional-locks"}, args...)
		cmd := exec.CommandContext(c, git, full...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
		winexec.Hide(cmd)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
}
