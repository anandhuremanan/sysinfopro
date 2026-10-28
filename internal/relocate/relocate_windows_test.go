//go:build windows

package relocate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// sandbox keeps move records in memory and allows moves within one drive
func sandbox(t *testing.T) string {
	t.Helper()
	store := map[string]record{}
	oldRead, oldWrite := readRecords, writeRecords
	readRecords = func() map[string]record {
		copy := map[string]record{}
		for k, v := range store {
			copy[k] = v
		}
		return copy
	}
	writeRecords = func(all map[string]record) error {
		store = all
		return nil
	}
	allowSameDrive = true
	t.Cleanup(func() {
		readRecords, writeRecords = oldRead, oldWrite
		allowSameDrive = false
	})
	return t.TempDir()
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// a pretend tool whose cache setting lives in a variable
func fakeTool(setting *string, works bool) kind {
	return kind{
		id: "fake", name: "Fake cache", group: "dev", method: "setting", folder: "fake-cache",
		locate: func(context.Context) (string, bool, error) { return *setting, isDir(*setting), nil },
		point: func(_ context.Context, dir string) error {
			if !works {
				return errors.New("the tool refused")
			}
			*setting = dir
			return nil
		},
	}
}

func fill(t *testing.T, dir string) {
	write(t, filepath.Join(dir, "index.json"), "index")
	write(t, filepath.Join(dir, "packages", "a", "1.0.0.tgz"), "package a")
	ro := filepath.Join(dir, "packages", "go.mod")
	write(t, ro, "module x")
	os.Chmod(ro, 0o444)
}

func TestSettingMoveAndBack(t *testing.T) {
	base := sandbox(t)
	src := filepath.Join(base, "C", "npm-cache")
	fill(t, src)
	setting := src
	k := fakeTool(&setting, true)
	dst := filepath.Join(base, "D", "Moved from C", k.folder)

	var phases []string
	res, err := moveKind(context.Background(), k, src, dst, dirSize(src), func(p Progress) {
		if len(phases) == 0 || phases[len(phases)-1] != p.Phase {
			phases = append(phases, p.Phase)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if setting != dst || res.To != dst || res.Note != "" {
		t.Fatalf("setting %s, result %+v", setting, res)
	}
	if read(t, filepath.Join(dst, "packages", "a", "1.0.0.tgz")) != "package a" || exists(src) {
		t.Fatal("files should be on the new drive only")
	}
	if info, _ := os.Stat(filepath.Join(dst, "packages", "go.mod")); info.Mode().Perm()&0o200 != 0 {
		t.Fatal("read-only files should stay read-only")
	}
	if strings.Join(phases, ",") != "copying,switching,cleaning" {
		t.Fatalf("phases %v", phases)
	}
	rec, ok := loadRecords()["fake"]
	if !ok || !samePath(rec.From, src) || !samePath(rec.To, dst) {
		t.Fatalf("record %+v", rec)
	}

	if _, err := moveKindBack(context.Background(), k, rec, nil); err != nil {
		t.Fatal(err)
	}
	if setting != src || read(t, filepath.Join(src, "index.json")) != "index" || exists(dst) {
		t.Fatal("moving back should restore the folder and the setting")
	}
	if _, ok := loadRecords()["fake"]; ok {
		t.Fatal("the record should be gone")
	}
}

func TestSettingRefusedLeavesEverything(t *testing.T) {
	base := sandbox(t)
	src := filepath.Join(base, "C", "cache")
	fill(t, src)
	setting := src
	k := fakeTool(&setting, false)
	dst := filepath.Join(base, "D", k.folder)
	if _, err := moveKind(context.Background(), k, src, dst, dirSize(src), nil); err == nil {
		t.Fatal("want an error")
	}
	if setting != src || !exists(filepath.Join(src, "index.json")) || exists(dst) {
		t.Fatal("a refused setting must leave the original and remove the copy")
	}
}

func TestSettingIgnoredIsPutBack(t *testing.T) {
	base := sandbox(t)
	src := filepath.Join(base, "C", "cache")
	fill(t, src)
	setting := src
	k := fakeTool(&setting, true)
	// a tool that accepts the change but keeps reporting the old place
	k.locate = func(context.Context) (string, bool, error) { return src, true, nil }
	dst := filepath.Join(base, "D", k.folder)
	_, err := moveKind(context.Background(), k, src, dst, dirSize(src), nil)
	if err == nil || !strings.Contains(err.Error(), "still reports") {
		t.Fatalf("err = %v", err)
	}
	if setting != src || !exists(filepath.Join(src, "index.json")) || exists(dst) {
		t.Fatal("the setting should be put back and the copy removed")
	}
}

func junctionKind(src string) kind {
	return kind{
		id: "browser", name: "Browser", group: "browser", method: "junction", folder: "Browser User Data",
		locate: func(context.Context) (string, bool, error) { return src, isDir(src), nil },
	}
}

func TestJunctionMoveAndBack(t *testing.T) {
	base := sandbox(t)
	src := filepath.Join(base, "C", "User Data")
	fill(t, src)
	k := junctionKind(src)
	dst := filepath.Join(base, "D", k.folder)

	if _, err := moveKind(context.Background(), k, src, dst, dirSize(src), nil); err != nil {
		t.Fatal(err)
	}
	target, isLink := junctionTarget(src)
	if !isLink || !samePath(target, dst) {
		t.Fatalf("the old place should be a link to %s, got %q %v", dst, target, isLink)
	}
	// the app still finds its files in the old place
	if read(t, filepath.Join(src, "packages", "a", "1.0.0.tgz")) != "package a" {
		t.Fatal("files should be reachable through the link")
	}
	if exists(src + ".sysinfopro-old") {
		t.Fatal("the old copy should be removed")
	}
	item, ok := describe(context.Background(), k, loadRecords(), map[string]bool{}, "Z:")
	if !ok || !item.Moved || !samePath(item.Path, dst) {
		t.Fatalf("describe after the move: %+v", item)
	}

	rec := loadRecords()["browser"]
	if _, err := moveKindBack(context.Background(), k, rec, nil); err != nil {
		t.Fatal(err)
	}
	if _, isLink := junctionTarget(src); isLink {
		t.Fatal("the link should be replaced by the real folder")
	}
	if read(t, filepath.Join(src, "index.json")) != "index" || exists(dst) {
		t.Fatal("files should be back and the copy gone")
	}
}

func TestJunctionInUseLeavesEverything(t *testing.T) {
	base := sandbox(t)
	src := filepath.Join(base, "C", "User Data")
	fill(t, src)
	k := junctionKind(src)
	dst := filepath.Join(base, "D", k.folder)

	// a running browser holding a file open
	name, _ := windows.UTF16PtrFromString(filepath.Join(src, "index.json"))
	h, err := windows.CreateFile(name, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = moveKind(context.Background(), k, src, dst, dirSize(src), nil)
	windows.CloseHandle(h)
	if err == nil {
		t.Fatal("want an error while a file is open")
	}
	if _, isLink := junctionTarget(src); isLink || !exists(filepath.Join(src, "index.json")) || exists(dst) {
		t.Fatal("the original must stay as it was and the copy be removed")
	}
}

func TestTargetChecks(t *testing.T) {
	base := sandbox(t)
	busy := filepath.Join(base, "busy")
	write(t, filepath.Join(busy, "x.txt"), "x")
	if err := checkTarget(filepath.Join(base, "src"), busy, 1); err == nil {
		t.Fatal("a folder with files should be refused")
	}
	if err := checkTarget(filepath.Join(base, "src"), `\\server\share\x`, 1); err == nil {
		t.Fatal("network folders should be refused")
	}
	allowSameDrive = false
	if err := checkTarget(`C:\Users\x\cache`, `C:\Other\cache`, 1); err == nil || !strings.Contains(err.Error(), "same drive") {
		t.Fatalf("same drive: %v", err)
	}
}

func TestCopyRefusesLinksInside(t *testing.T) {
	base := sandbox(t)
	src := filepath.Join(base, "src")
	write(t, filepath.Join(src, "a.txt"), "a")
	outside := filepath.Join(base, "outside")
	write(t, filepath.Join(outside, "b.txt"), "b")
	if err := createJunction(filepath.Join(src, "link"), outside); err != nil {
		t.Fatal(err)
	}
	if err := copyTree(context.Background(), src, filepath.Join(base, "dst"), nil); err == nil || !strings.Contains(err.Error(), "link") {
		t.Fatalf("err = %v", err)
	}
}

func TestCloseFirst(t *testing.T) {
	procs := [][2]string{{"chrome.exe", "Chrome"}, {"node.exe", "Node.js programs"}}
	if got := closeFirst(procs, map[string]bool{"chrome.exe": true, "node.exe": true}); got != "Close Chrome and Node.js programs first" {
		t.Fatal(got)
	}
	if closeFirst(procs, map[string]bool{"notepad.exe": true}) != "" {
		t.Fatal("nothing to close")
	}
}
