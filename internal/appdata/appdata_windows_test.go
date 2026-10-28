//go:build windows

package appdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// lock opens path so no one else can read it, like a browser process that has
// not exited yet
func lock(t *testing.T, path string) windows.Handle {
	t.Helper()
	name, _ := windows.UTF16PtrFromString(path)
	h, err := windows.CreateFile(name, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func shortRetries(t *testing.T, waits ...time.Duration) {
	old := retryWaits
	retryWaits = waits
	t.Cleanup(func() { retryWaits = old })
}

func TestLockedFileLeavesDataInPlace(t *testing.T) {
	root, logs, base := sandbox(t)
	fill(t, root, logs)
	shortRetries(t, 10*time.Millisecond, 10*time.Millisecond)
	Init()
	target, err := RequestMove(filepath.Join(base, "Data"))
	if err != nil {
		t.Fatal(err)
	}

	locked := filepath.Join(root, webviewName, "EBWebView", "Default", "Local Storage", "leveldb", "000003.log")
	h := lock(t, locked)
	defer windows.CloseHandle(h)

	reset()
	Init()
	if Dir() != root || LogDir() != logs {
		t.Fatalf("a failed move must keep the old folders, got %s", Dir())
	}
	if exists(target) {
		t.Fatal("the half-made copy should be removed")
	}
	if !exists(filepath.Join(root, "alerts.json")) || !exists(filepath.Join(logs, "app.log")) {
		t.Fatal("the old data must be untouched")
	}
	s := GetStatus()
	if s.Pending || !s.NoticeIsErr || !strings.Contains(s.Notice, "did not work") {
		t.Fatalf("status %+v", s)
	}
}

func TestLockedFileRetried(t *testing.T) {
	root, logs, base := sandbox(t)
	fill(t, root, logs)
	shortRetries(t, 300*time.Millisecond)
	Init()
	target, err := RequestMove(filepath.Join(base, "Data"))
	if err != nil {
		t.Fatal(err)
	}

	// the file is let go while the move waits to try again
	h := lock(t, filepath.Join(root, "alerts.json"))
	go func() {
		time.Sleep(100 * time.Millisecond)
		windows.CloseHandle(h)
	}()

	reset()
	Init()
	if Dir() != target {
		t.Fatalf("the second try should succeed, got %s (%s)", Dir(), GetStatus().Notice)
	}
	if read(t, filepath.Join(target, "alerts.json")) != `{"enabled":true}` {
		t.Fatal("file not copied")
	}
	if _, err := os.Stat(filepath.Join(root, "alerts.json")); err == nil {
		t.Fatal("old copy should be gone")
	}
}
