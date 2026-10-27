package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sample struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
	Limit   int    `json:"limit"`
}

func useTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dirOverride = dir
	t.Cleanup(func() { dirOverride = "" })
	return dir
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := useTemp(t)
	if err := Save("a.json", sample{Enabled: true, Name: "x", Limit: 3}); err != nil {
		t.Fatal(err)
	}
	got := Load("a.json", sample{})
	if got != (sample{Enabled: true, Name: "x", Limit: 3}) {
		t.Errorf("got %+v", got)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "a.json"))
	if !strings.Contains(string(data), `"schema_version": 1`) {
		t.Errorf("no schema version in %s", data)
	}
	// no temporary files left behind
	if m, _ := filepath.Glob(filepath.Join(dir, "*.tmp")); len(m) != 0 {
		t.Errorf("leftover temp files: %v", m)
	}
}

func TestMissingFieldsKeepDefaults(t *testing.T) {
	dir := useTemp(t)
	// a file from an older version, without "limit" or a schema version
	os.WriteFile(filepath.Join(dir, "old.json"), []byte(`{"enabled": true}`), 0o644)
	got := Load("old.json", sample{Limit: 10})
	if !got.Enabled || got.Limit != 10 {
		t.Errorf("got %+v", got)
	}
}

func TestCorruptFileFallsBackAndReports(t *testing.T) {
	dir := useTemp(t)
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{not json`), 0o644)
	got, found, err := LoadExisting("bad.json", sample{Limit: 5})
	if !found || err == nil || got.Limit != 5 {
		t.Errorf("got %+v found=%v err=%v", got, found, err)
	}
	if got := Load("bad.json", sample{Limit: 5}); got.Limit != 5 {
		t.Errorf("Load did not fall back: %+v", got)
	}
}

func TestMissingFile(t *testing.T) {
	useTemp(t)
	got, found, err := LoadExisting("none.json", sample{Name: "d"})
	if found || err != nil || got.Name != "d" {
		t.Errorf("got %+v found=%v err=%v", got, found, err)
	}
}

func TestRemove(t *testing.T) {
	dir := useTemp(t)
	Save("r.json", sample{})
	Remove("r.json")
	Remove("r.json") // a second time is fine
	if _, err := os.Stat(filepath.Join(dir, "r.json")); !os.IsNotExist(err) {
		t.Error("file still there")
	}
}
