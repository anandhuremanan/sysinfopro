//go:build windows

package locks

import (
	"os"
	"path/filepath"
	"testing"
)

func hasPID(result *Result, pid int) bool {
	for _, p := range result.Processes {
		if int(p.PID) == pid {
			return true
		}
	}
	return false
}

func TestFindReportsTheProcessHoldingAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "held.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	held, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	// this test process holds the file open, so it must be reported for the file...
	result, err := Find(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsDir || result.FilesChecked != 1 || !hasPID(result, os.Getpid()) {
		t.Errorf("file lookup = %+v, want this process (pid %d)", result, os.Getpid())
	}

	// ...and for the folder that contains it
	result, err = Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsDir || !hasPID(result, os.Getpid()) {
		t.Errorf("folder lookup = %+v, want this process (pid %d)", result, os.Getpid())
	}

	// once released, nothing holds it
	held.Close()
	result, err = Find(path)
	if err != nil {
		t.Fatal(err)
	}
	if hasPID(result, os.Getpid()) {
		t.Errorf("file is closed but this process is still reported: %+v", result.Processes)
	}
}

func TestFindEmptyFolderAndMissingPath(t *testing.T) {
	result, err := Find(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result.FilesChecked != 0 || len(result.Processes) != 0 {
		t.Errorf("empty folder = %+v, want nothing checked and no processes", result)
	}

	if _, err := Find(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Error("expected an error for a missing path")
	}
}
