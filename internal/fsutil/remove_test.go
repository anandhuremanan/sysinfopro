package fsutil

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestRemoveTreeCountsBytes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache")
	writeFile(t, filepath.Join(root, "a", "one.bin"), 300)
	writeFile(t, filepath.Join(root, "two.bin"), 200)
	locked := filepath.Join(root, "a", "locked.bin")
	writeFile(t, locked, 50)
	if err := os.Chmod(locked, 0o444); err != nil {
		t.Fatal(err)
	}

	var freed int64
	files := 0
	skipped, err := RemoveTree(context.Background(), root, func(size int64) {
		freed += size
		files++
	})
	if err != nil || skipped != 0 || exists(root) {
		t.Fatalf("err = %v, skipped = %d, root exists = %v", err, skipped, exists(root))
	}
	if freed != 550 || files != 3 {
		t.Errorf("freed %d bytes in %d files, want 550 in 3", freed, files)
	}
}

func TestRemoveTreeStopsWhenCancelled(t *testing.T) {
	root := filepath.Join(t.TempDir(), "node_modules")
	for i := 0; i < 20; i++ {
		writeFile(t, filepath.Join(root, "pkg", string(rune('a'+i))+".js"), 10)
	}

	ctx, cancel := context.WithCancel(context.Background())
	removed := 0
	_, err := RemoveTree(ctx, root, func(int64) {
		removed++
		cancel()
	})
	if !errors.Is(err, ErrStopped) {
		t.Fatalf("err = %v, want ErrStopped", err)
	}
	if removed != 1 || !exists(root) {
		t.Errorf("removed %d files, want 1 and the rest left in place", removed)
	}
}

func TestRemoveTreeDoesNotFollowLinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "my-project", "src")
	keep := filepath.Join(outside, "main.js")
	writeFile(t, keep, 50)
	target := filepath.Join(root, "node_modules")
	writeFile(t, filepath.Join(target, "real.js"), 10)

	// pnpm links packages with junctions on Windows, which need no special privilege
	link := filepath.Join(target, "linked")
	if err := os.Symlink(outside, link); err != nil {
		if runtime.GOOS != "windows" {
			t.Skipf("cannot create symlinks here: %v", err)
		}
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput(); err != nil {
			t.Skipf("cannot create a junction here: %v %s", err, out)
		}
	}

	if _, err := RemoveTree(context.Background(), target, func(int64) {}); err != nil || exists(target) {
		t.Fatalf("err = %v, want the folder removed", err)
	}
	if !exists(keep) {
		t.Fatal("a file outside the folder was deleted through a link")
	}
}
