//go:build windows

package shred

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("secret!", size/7+1))[:size], 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestShredFilesAndFolders(t *testing.T) {
	dir := t.TempDir()
	single := filepath.Join(dir, "passwords.txt")
	write(t, single, 5000)
	folder := filepath.Join(dir, "old-taxes")
	write(t, filepath.Join(folder, "2024", "return.pdf"), 3<<20)
	readOnly := filepath.Join(folder, "scan.png")
	write(t, readOnly, 100)
	os.Chmod(readOnly, 0o444)
	empty := filepath.Join(folder, "empty.txt")
	write(t, empty, 0)

	var last Progress
	result := Shred(context.Background(), []string{single, folder}, 3, func(p Progress) { last = p })
	if result.Shredded != 4 || len(result.Skipped) != 0 {
		t.Fatalf("result %+v", result)
	}
	if exists(single) || exists(folder) {
		t.Fatal("shredded files and their folder should be gone")
	}
	left, _ := os.ReadDir(dir)
	if len(left) != 0 {
		t.Fatalf("renamed leftovers in the folder: %v", left)
	}
	if last.DoneBytes != last.TotalBytes || last.TotalBytes != 3*(5000+3<<20+100) {
		t.Fatalf("progress %+v", last)
	}
}

func TestShredLeavesFilesInUse(t *testing.T) {
	dir := t.TempDir()
	busy := filepath.Join(dir, "open.log")
	write(t, busy, 100)
	name, _ := windows.UTF16PtrFromString(busy)
	// another program holding it open without sharing
	h, err := windows.CreateFile(name, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)

	result := Shred(context.Background(), []string{busy}, 1, nil)
	if result.Shredded != 0 || len(result.Skipped) != 1 || !strings.Contains(result.Skipped[0].Reason, "open in another program") {
		t.Fatalf("result %+v", result)
	}
}

func TestShredRefusesHardLinks(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "store", "package.js")
	write(t, original, 200)
	twin := filepath.Join(dir, "project", "package.js")
	os.MkdirAll(filepath.Dir(twin), 0o755)
	if err := os.Link(original, twin); err != nil {
		t.Skipf("hard links not available here: %v", err)
	}
	result := Shred(context.Background(), []string{twin}, 1, nil)
	if result.Shredded != 0 || !strings.Contains(result.Skipped[0].Reason, "hard link") {
		t.Fatalf("result %+v", result)
	}
	if data, _ := os.ReadFile(original); len(data) != 200 || !strings.HasPrefix(string(data), "secret!") {
		t.Fatal("the other copy must be untouched")
	}
}

func TestShredDoesNotFollowJunctions(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "keep")
	keep := filepath.Join(outside, "important.doc")
	write(t, keep, 100)
	folder := filepath.Join(dir, "to-shred")
	write(t, filepath.Join(folder, "a.txt"), 10)
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(folder, "link"), outside).CombinedOutput(); err != nil {
		t.Skipf("cannot make a junction: %v %s", err, out)
	}
	Shred(context.Background(), []string{folder}, 1, nil)
	if !exists(keep) {
		t.Fatal("a file outside the folder was shredded through a junction")
	}
}

func TestProtected(t *testing.T) {
	for _, p := range []string{`C:\`, os.Getenv("WINDIR"), filepath.Join(os.Getenv("WINDIR"), "System32", "x.dll"), os.Getenv("USERPROFILE")} {
		if Protected(p) == "" {
			t.Errorf("%s should be refused", p)
		}
	}
	if r := Protected(filepath.Join(os.Getenv("USERPROFILE"), "Documents", "old.txt")); r != "" {
		t.Errorf("a file in Documents should be allowed, got %q", r)
	}
}

func TestDriveOf(t *testing.T) {
	if DriveOf(`d:\stuff\a.txt`) != "D:" || DriveOf(`\\server\share\a.txt`) != "" {
		t.Fatal("DriveOf")
	}
}

func TestShredOverwritesBeforeDeleting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.bin")
	write(t, path, 4<<20)
	ctx, cancel := context.WithCancel(context.Background())
	// stop after the first megabyte, so the file is left part-overwritten
	result := Shred(ctx, []string{path}, 1, func(p Progress) {
		if p.DoneBytes >= 1<<20 {
			cancel()
		}
	})
	if !result.Stopped || result.Shredded != 0 {
		t.Fatalf("result %+v", result)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data[:1<<20]), "secret!") {
		t.Fatal("the start of the file should already be overwritten")
	}
	if !strings.Contains(string(data[3<<20:]), "secret!") {
		t.Fatal("the end should still be the original, since it stopped early")
	}
}
