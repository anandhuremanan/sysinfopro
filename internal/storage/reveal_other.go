//go:build !windows

package storage

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

func revealCommand(path string) *exec.Cmd {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-R", path)
	}
	return exec.Command("xdg-open", filepath.Dir(path))
}
