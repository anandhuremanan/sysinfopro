package dupes

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, path string, data []byte, age time.Duration) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	os.Chtimes(path, when, when)
}

func TestFind(t *testing.T) {
	root := t.TempDir()
	big := bytes.Repeat([]byte("abcdefgh"), 40_000) // 320 KB: needs a full read
	// same size, same first and last 64 KB, different middle: only a full read tells them apart
	almost := bytes.Clone(big)
	almost[len(almost)/2] ^= 1
	small := []byte("hello duplicate world")

	write(t, filepath.Join(root, "a", "photo.jpg"), big, 3*time.Hour)
	write(t, filepath.Join(root, "b", "photo (1).jpg"), big, time.Hour)
	write(t, filepath.Join(root, "c", "photo-edited.jpg"), almost, 0)
	write(t, filepath.Join(root, "notes.txt"), small, 0)
	write(t, filepath.Join(root, "Downloads", "notes.txt"), small, 0)
	write(t, filepath.Join(root, "unique.txt"), []byte("only one of me"), 0)
	// a project's dependencies are never offered for deletion
	write(t, filepath.Join(root, "proj", "node_modules", "x", "photo.jpg"), big, 0)

	var phases = map[string]bool{}
	res, err := Find(context.Background(), root, 1, func(p Progress) { phases[p.Phase] = true })
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 2 {
		t.Fatalf("got %d groups, want 2: %+v", len(res.Groups), res.Groups)
	}
	first := res.Groups[0] // biggest waste first
	if first.Size != int64(len(big)) || len(first.Files) != 2 || first.Wasted != int64(len(big)) {
		t.Errorf("big group = %+v", first)
	}
	if filepath.Base(filepath.Dir(first.Files[0].Path)) != "a" {
		t.Errorf("oldest copy should be listed first: %+v", first.Files)
	}
	if res.Groups[1].Size != int64(len(small)) {
		t.Errorf("small group = %+v", res.Groups[1])
	}
	if res.Wasted != int64(len(big)+len(small)) {
		t.Errorf("wasted = %d", res.Wasted)
	}

	// a minimum size leaves the small files out
	res, _ = Find(context.Background(), root, 1024, nil)
	if len(res.Groups) != 1 {
		t.Errorf("with a minimum size: %d groups", len(res.Groups))
	}
}

func TestHardLinksAreNotDuplicates(t *testing.T) {
	root := t.TempDir()
	data := bytes.Repeat([]byte("x"), 5000)
	write(t, filepath.Join(root, "one.bin"), data, 0)
	if err := os.Link(filepath.Join(root, "one.bin"), filepath.Join(root, "two.bin")); err != nil {
		t.Skip("hard links not supported here:", err)
	}
	res, err := Find(context.Background(), root, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 0 {
		t.Errorf("two names for one file take no extra space: %+v", res.Groups)
	}
}

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Find(ctx, t.TempDir(), 1, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
}

func TestCheckKeepsOne(t *testing.T) {
	root := t.TempDir()
	a, b, c := filepath.Join(root, "a"), filepath.Join(root, "b"), filepath.Join(root, "c")
	for _, p := range []string{a, b, c} {
		write(t, p, []byte("same"), 0)
	}
	groups := []Group{{Size: 4, Files: []File{{Path: a}, {Path: b}, {Path: c}}}}

	if err := CheckKeepsOne(groups, []string{a, b}); err != nil {
		t.Errorf("keeping c: %v", err)
	}
	if err := CheckKeepsOne(groups, []string{a, b, c}); !errors.Is(err, ErrWouldRemoveAll) {
		t.Errorf("removing all: %v", err)
	}
	// c was deleted outside the app since the search: a and b are now the last copies
	os.Remove(c)
	if err := CheckKeepsOne(groups, []string{a, b}); !errors.Is(err, ErrWouldRemoveAll) {
		t.Errorf("kept copy gone: %v", err)
	}
	if err := CheckKeepsOne(groups, []string{filepath.Join(root, "other")}); err == nil {
		t.Error("a path from outside the search must be refused")
	}
}

func TestProgramPartsAreRisky(t *testing.T) {
	root := t.TempDir()
	data := bytes.Repeat([]byte("lib"), 3000)
	write(t, filepath.Join(root, "app1", "engine.dll"), data, 0)
	write(t, filepath.Join(root, "app2", "engine.dll"), data, 0)
	res, err := Find(context.Background(), root, 1, nil)
	if err != nil || len(res.Groups) != 1 || !res.Groups[0].Risky {
		t.Errorf("got %+v, %v", res, err)
	}
}
