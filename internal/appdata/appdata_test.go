package appdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sandbox points the default folders at a temporary folder for one test
func sandbox(t *testing.T) (root, logs, base string) {
	t.Helper()
	base = t.TempDir()
	root = filepath.Join(base, "Roaming", folderName)
	logs = filepath.Join(base, "Local", folderName, logsName)
	oldRoot, oldLogs := defaultRoot, defaultLogs
	defaultRoot = func() string { return root }
	defaultLogs = func() string { return logs }
	reset()
	t.Cleanup(func() {
		defaultRoot, defaultLogs = oldRoot, oldLogs
		reset()
	})
	return root, logs, base
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

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

// fill puts typical app files in the folders for dir
func fill(t *testing.T, root, logs string) {
	write(t, filepath.Join(root, "alerts.json"), `{"enabled":true}`)
	write(t, filepath.Join(root, "snapshots", "1.json"), "snap")
	write(t, filepath.Join(root, webviewName, "EBWebView", "Default", "Local Storage", "leveldb", "000003.log"), "theme=dark")
	write(t, filepath.Join(root, webviewName, "EBWebView", "Default", "Cache", "data_1"), strings.Repeat("x", 1000))
	write(t, filepath.Join(logs, "app.log"), "started")
}

func TestDefaultFolders(t *testing.T) {
	root, logs, _ := sandbox(t)
	Init()
	if Dir() != root || LogDir() != logs || WebViewDir() != filepath.Join(root, webviewName) {
		t.Fatalf("got %s, %s, %s", Dir(), LogDir(), WebViewDir())
	}
	if s := GetStatus(); !s.IsDefault || s.Pending || s.Notice != "" {
		t.Fatalf("status %+v", s)
	}
	if !FreshInstall() {
		t.Fatal("an empty data folder is a fresh install")
	}
	write(t, filepath.Join(root, "alerts.json"), "{}")
	reset()
	Init()
	if FreshInstall() {
		t.Fatal("a data folder with files is not a fresh install")
	}
}

func TestMoveAndBack(t *testing.T) {
	root, logs, base := sandbox(t)
	fill(t, root, logs)
	Init()

	picked := filepath.Join(base, "Data")
	target, err := RequestMove(picked)
	if err != nil {
		t.Fatal(err)
	}
	if target != filepath.Join(picked, folderName) {
		t.Fatalf("target %s", target)
	}
	if exists(target) {
		t.Fatal("the check should leave no folder behind before the move")
	}
	if s := GetStatus(); !s.Pending || s.PendingTo != target || s.Dir != root {
		t.Fatalf("status before restart %+v", s)
	}

	// the next start carries out the move
	reset()
	Init()
	if Dir() != target || LogDir() != filepath.Join(target, logsName) {
		t.Fatalf("after the move: %s, %s", Dir(), LogDir())
	}
	if read(t, filepath.Join(target, "alerts.json")) != `{"enabled":true}` ||
		read(t, filepath.Join(target, "snapshots", "1.json")) != "snap" ||
		read(t, filepath.Join(target, webviewName, "EBWebView", "Default", "Local Storage", "leveldb", "000003.log")) != "theme=dark" ||
		read(t, filepath.Join(target, logsName, "app.log")) != "started" {
		t.Fatal("files were not copied intact")
	}
	if exists(filepath.Join(target, webviewName, "EBWebView", "Default", "Cache")) {
		t.Fatal("the browser cache should not be copied")
	}
	// only the pointer is left in the old place
	left, _ := os.ReadDir(root)
	if len(left) != 1 || left[0].Name() != pointerName {
		t.Fatalf("old folder still has %v", left)
	}
	if exists(logs) {
		t.Fatal("old logs folder was not removed")
	}
	s := GetStatus()
	if s.IsDefault || s.Pending || s.NoticeIsErr || !strings.Contains(s.Notice, target) {
		t.Fatalf("status after the move %+v", s)
	}

	// and back to the default folders
	if err := RequestDefault(); err != nil {
		t.Fatal(err)
	}
	reset()
	Init()
	if Dir() != root || LogDir() != logs {
		t.Fatalf("after moving back: %s, %s", Dir(), LogDir())
	}
	if read(t, filepath.Join(root, "alerts.json")) != `{"enabled":true}` || read(t, filepath.Join(logs, "app.log")) != "started" {
		t.Fatal("files did not come back")
	}
	if exists(target) {
		t.Fatal("the chosen folder should be removed once empty")
	}
	if exists(filepath.Join(root, pointerName)) {
		t.Fatal("back on the defaults, the pointer is not needed")
	}
}

func TestCancelMove(t *testing.T) {
	root, logs, base := sandbox(t)
	fill(t, root, logs)
	Init()
	if _, err := RequestMove(filepath.Join(base, "Data")); err != nil {
		t.Fatal(err)
	}
	if err := CancelMove(); err != nil {
		t.Fatal(err)
	}
	reset()
	Init()
	if Dir() != root || !exists(filepath.Join(root, "alerts.json")) {
		t.Fatal("a cancelled move should change nothing")
	}
}

func TestRefusedTargets(t *testing.T) {
	root, logs, base := sandbox(t)
	fill(t, root, logs)
	Init()

	busy := filepath.Join(base, "Busy", folderName)
	write(t, filepath.Join(busy, "notes.txt"), "mine")

	cases := map[string]string{
		"":                                 "no folder",
		`\\server\share`:                   "network",
		"relative":                         "drive",
		root:                               "already",
		filepath.Dir(root):                 "already",
		filepath.Join(root, "snapshots"):   "outside",
		filepath.Join(base, "Busy"):        "already has files",
		filepath.Join(os.Getenv("WINDIR")): "Windows",
	}
	for picked, want := range cases {
		if picked == "" && want != "no folder" {
			continue
		}
		_, err := RequestMove(picked)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("RequestMove(%q) = %v, want an error about %q", picked, err, want)
		}
	}
	if read(t, filepath.Join(busy, "notes.txt")) != "mine" {
		t.Fatal("a refused folder must be left alone")
	}
}

func TestMissingFolderFallsBack(t *testing.T) {
	root, _, base := sandbox(t)
	if err := writePointer(pointer{Dir: filepath.Join(base, "Gone", folderName)}); err != nil {
		t.Fatal(err)
	}
	Init()
	if Dir() != root {
		t.Fatalf("got %s", Dir())
	}
	s := GetStatus()
	if !s.NoticeIsErr || !strings.Contains(s.Notice, "could not be found") {
		t.Fatalf("status %+v", s)
	}
	// the chosen folder is remembered for when its drive comes back
	if p, _ := readPointer(); p.Dir == "" {
		t.Fatal("the pointer should keep the missing folder")
	}
	// choosing the default forgets it
	if err := RequestDefault(); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(root, pointerName)) {
		t.Fatal("the pointer should be gone")
	}
}
