package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var old = time.Now().Add(-48 * time.Hour)

// writeOld creates a file of size bytes and backdates it, and the folders above it, by two days
func writeOld(t *testing.T, root, rel string, size int) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	for p := path; p != root; p = filepath.Dir(p) {
		os.Chtimes(p, old, old)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func fixture(t *testing.T) (temp, browser string, locations []Location) {
	root := t.TempDir()
	temp = filepath.Join(root, "Temp")
	browser = filepath.Join(root, "Chrome", "Default", "Cache")
	writeOld(t, temp, "setup.log", 100)
	writeOld(t, temp, filepath.Join("installer", "a.bin"), 400)
	writeOld(t, browser, "data_1", 1000)
	// changed just now: probably in use, so never touched
	os.WriteFile(filepath.Join(temp, "in-use.tmp"), make([]byte, 5000), 0o644)
	// an old folder holding one fresh file counts as in use too
	writeOld(t, temp, filepath.Join("build-1234", "old.obj"), 700)
	os.WriteFile(filepath.Join(temp, "build-1234", "new.obj"), make([]byte, 10), 0o644)
	os.Chtimes(filepath.Join(temp, "build-1234"), old, old)

	return temp, browser, []Location{
		{Category: "Temporary files", Path: temp},
		{Category: "Temporary files", Path: temp + string(filepath.Separator)}, // same folder twice
		{Category: "Chrome cache", Path: browser},
		{Category: "Edge cache", Path: filepath.Join(root, "missing")},
	}
}

func TestScan(t *testing.T) {
	_, _, locations := fixture(t)
	result := Scan(context.Background(), locations)

	if result.TotalBytes != 1500 || len(result.Categories) != 2 {
		t.Fatalf("result = %+v, want 1500 bytes in 2 categories", result)
	}
	if c := result.Categories[0]; c.Name != "Temporary files" || c.SizeBytes != 500 || c.Items != 2 {
		t.Errorf("temp = %+v, want 500 bytes in 2 items (folder counted once, items in use left out)", c)
	}
	if c := result.Categories[1]; c.Name != "Chrome cache" || c.SizeBytes != 1000 {
		t.Errorf("chrome = %+v, want 1000 bytes", c)
	}
}

func TestCleanKeepsRecentFiles(t *testing.T) {
	temp, browser, locations := fixture(t)
	var last Progress
	summary := Clean(context.Background(), locations, func(p Progress) { last = p })

	if summary.FreedBytes != 1500 || summary.FilesDeleted != 3 || summary.Stopped {
		t.Fatalf("summary = %+v, want 1500 bytes in 3 files", summary)
	}
	if last.FreedBytes != 1500 || last.TotalBytes != 1500 {
		t.Errorf("last progress = %+v", last)
	}
	for _, gone := range []string{"setup.log", "installer"} {
		if exists(filepath.Join(temp, gone)) {
			t.Errorf("%s should be gone", gone)
		}
	}
	if exists(filepath.Join(browser, "data_1")) {
		t.Error("browser cache file should be gone")
	}
	for _, kept := range []string{"in-use.tmp", filepath.Join("build-1234", "old.obj")} {
		if !exists(filepath.Join(temp, kept)) {
			t.Errorf("%s was changed recently or sits beside a recent file, so it must be kept", kept)
		}
	}
}

func TestCleanStops(t *testing.T) {
	_, _, locations := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if summary := Clean(ctx, locations, nil); !summary.Stopped || summary.FreedBytes != 0 {
		t.Errorf("summary = %+v, want stopped with nothing freed", summary)
	}
}

func TestTempDirProblem(t *testing.T) {
	profile, windows := `C:\Users\dev`, `C:\Windows`
	ok := []string{`C:\Users\dev\AppData\Local\Temp`, `D:\tmp`, `C:\Users\dev\AppData\Local\Temp\`}
	for _, dir := range ok {
		if p := tempDirProblem(dir, profile, windows); p != "" {
			t.Errorf("%s should be fine, got %q", dir, p)
		}
	}
	bad := []string{``, `D:\`, `D:`, `C:\Users\dev`, `c:\users\DEV\`, `C:\Windows`, `C:\Users\dev\Documents`, `C:\Users\dev\Downloads`, `D:\Work`}
	for _, dir := range bad {
		if p := tempDirProblem(dir, profile, windows); p == "" {
			t.Errorf("%q must be refused", dir)
		}
	}
}

func TestRefusedLocationIsNotTouched(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "keep.txt")
	os.WriteFile(old, []byte("x"), 0o644)
	past := time.Now().Add(-2 * time.Hour)
	os.Chtimes(old, past, past)
	locs := []Location{{Category: "Temporary files", Path: dir, Refused: "not a temp folder"}}
	res := Scan(context.Background(), locs)
	if res.TotalBytes != 0 || len(res.Refused) != 1 {
		t.Fatalf("scan = %+v", res)
	}
	Clean(context.Background(), locs, nil)
	if _, err := os.Stat(old); err != nil {
		t.Fatal("a refused folder was cleaned")
	}
}
