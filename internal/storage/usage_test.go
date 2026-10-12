package storage

import (
	"context"
	"os"
	"path/filepath"
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

func TestAnalyzeFolderSizesAndOrder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "big", "a.bin"), 5000)
	writeFile(t, filepath.Join(root, "big", "nested", "b.bin"), 3000)
	writeFile(t, filepath.Join(root, "small", "c.bin"), 100)
	writeFile(t, filepath.Join(root, "top.bin"), 700)
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	usage, err := AnalyzeFolder(context.Background(), root, nil)
	if err != nil {
		t.Fatal(err)
	}

	if usage.TotalBytes != 8800 || usage.Files != 4 {
		t.Fatalf("total = %d bytes in %d files, want 8800 in 4", usage.TotalBytes, usage.Files)
	}

	want := []struct {
		name  string
		size  int64
		files int64
		isDir bool
	}{
		{"big", 8000, 2, true},
		{"top.bin", 700, 1, false},
		{"small", 100, 1, true},
		{"empty", 0, 0, true},
	}
	if len(usage.Entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(usage.Entries), len(want))
	}
	for i, w := range want {
		got := usage.Entries[i]
		if got.Name != w.name || got.SizeBytes != w.size || got.Files != w.files || got.IsDir != w.isDir {
			t.Errorf("entry %d = %+v, want %+v", i, got, w)
		}
	}

	if len(usage.LargestFiles) != 4 || usage.LargestFiles[0].Name != "a.bin" || usage.LargestFiles[3].Name != "c.bin" {
		t.Errorf("largest files out of order: %+v", usage.LargestFiles)
	}
}

func TestAnalyzeFolderKeepsOnlyLargestFiles(t *testing.T) {
	root := t.TempDir()
	for i := 1; i <= 80; i++ {
		writeFile(t, filepath.Join(root, "d", "f"+string(rune('a'+i%26))+string(rune('a'+i/26))+".bin"), i*10)
	}

	usage, err := AnalyzeFolder(context.Background(), root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(usage.LargestFiles) != largestFilesKept {
		t.Fatalf("kept %d largest files, want %d", len(usage.LargestFiles), largestFilesKept)
	}
	if usage.LargestFiles[0].SizeBytes != 800 || usage.LargestFiles[largestFilesKept-1].SizeBytes != 560 {
		t.Errorf("largest = %d, smallest kept = %d, want 800 and 560",
			usage.LargestFiles[0].SizeBytes, usage.LargestFiles[largestFilesKept-1].SizeBytes)
	}
}

func TestAnalyzeFolderCancelled(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "d", "a.bin"), 10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	usage, err := AnalyzeFolder(ctx, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !usage.Cancelled {
		t.Error("expected Cancelled to be set")
	}
}

func TestAnalyzeFolderMissing(t *testing.T) {
	if _, err := AnalyzeFolder(context.Background(), filepath.Join(t.TempDir(), "nope"), nil); err == nil {
		t.Error("expected an error for a missing folder")
	}
}

func TestRootPath(t *testing.T) {
	sep := string(filepath.Separator)
	cases := map[string]string{
		"C:":        "C:" + sep,
		"d:":        "d:" + sep,
		`C:\`:       `C:\`,
		`C:\Users`:  `C:\Users`,
		"/":         "/",
		"/mnt/data": "/mnt/data",
	}
	for in, want := range cases {
		if got := RootPath(in); got != want {
			t.Errorf("RootPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestListDrives(t *testing.T) {
	drives, err := ListDrives()
	if err != nil {
		t.Fatal(err)
	}
	if len(drives) == 0 {
		t.Fatal("expected at least one drive")
	}
	for _, d := range drives {
		if d.Path == "" || d.TotalBytes == 0 || d.UsedBytes > d.TotalBytes {
			t.Errorf("implausible drive: %+v", d)
		}
	}
}
