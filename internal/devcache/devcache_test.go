package devcache

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestFindLeftovers(t *testing.T) {
	root := t.TempDir()
	write := func(rel string, size int) { writeFile(t, filepath.Join(root, rel), size) }

	write("app-a/package.json", 10)
	write("app-a/node_modules/x/index.js", 4000)
	// nested node_modules belongs to app-a's total, not a separate result
	write("app-a/node_modules/x/node_modules/y/index.js", 1000)
	write("rust-tool/Cargo.toml", 10)
	write("rust-tool/target/debug/tool.exe", 3000)
	write("py-app/.venv/pyvenv.cfg", 10)
	write("py-app/.venv/Lib/site-packages/pkg.py", 1990)
	write("web/package.json", 10)
	write("web/.next/cache/page.js", 1500)
	write("svc/svc.csproj", 10)
	write("svc/bin/Debug/svc.dll", 700)
	write("svc/obj/project.assets.json", 600)

	// look-alikes that must never be offered
	write("photos/target/holiday.jpg", 8000)    // no Cargo.toml or pom.xml
	write("scripts/venv/notes.txt", 900)        // no pyvenv.cfg
	write("site/.next/readme.txt", 500)         // no package.json
	write("tool/bin/run.sh", 300)               // no .NET project
	write("wails-app/build/appicon.png", 400)   // build is never a leftover
	write("app-c/.git/node_modules/x.js", 9999) // nothing inside .git is searched

	items, err := FindLeftovers(context.Background(), root, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	for _, item := range items {
		got[item.Name+" "+item.Label] = item.SizeBytes
		if item.LastUsed == "" || item.Kind != "leftover" || !item.Deletable {
			t.Errorf("%s: %+v", item.Path, item)
		}
	}
	want := map[string]int64{
		"app-a node_modules":          5000,
		"rust-tool Rust build":        3000,
		"py-app Python environment":   2000,
		"web Next.js build":           1500,
		"svc .NET build output":       700,
		"svc .NET intermediate files": 600,
	}
	if len(got) != len(want) {
		t.Fatalf("found %v\nwant %v", got, want)
	}
	for key, size := range want {
		if got[key] != size {
			t.Errorf("%s: %d bytes, want %d", key, got[key], size)
		}
	}
	if items[0].Label != "node_modules" || items[0].Folder != "node_modules" {
		t.Errorf("largest first: got %+v", items[0])
	}
}

func TestProjectLastUsedTakesNewestMarker(t *testing.T) {
	project := t.TempDir()
	old := time.Date(2024, 3, 1, 12, 0, 0, 0, time.Local)
	recent := time.Date(2026, 9, 20, 12, 0, 0, 0, time.Local)

	manifest := filepath.Join(project, "package.json")
	installMarker := filepath.Join(project, "node_modules", ".package-lock.json")
	writeFile(t, manifest, 10)
	writeFile(t, installMarker, 10)
	// the folder itself was last changed long ago too
	modules := filepath.Join(project, "node_modules")
	// an untouched package.json must not hide a recent install
	if err := os.Chtimes(manifest, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(installMarker, recent, recent); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(modules, old, old); err != nil {
		t.Fatal(err)
	}

	if got := projectLastUsed(project, filepath.Join(project, "node_modules")); got != "2026-09-20" {
		t.Errorf("projectLastUsed = %q, want 2026-09-20", got)
	}
}

func TestScanCachesSkipsMissing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "npm-cache", "a", "b.tgz"), 2000)
	writeFile(t, filepath.Join(root, "pip", "c.whl"), 500)

	caches := []knownCache{
		{"pip", "pip cache", filepath.Join(root, "pip"), "", true},
		{"npm", "npm cache", filepath.Join(root, "npm-cache"), "", true},
		{"gradle", "Gradle caches", filepath.Join(root, "missing"), "", true},
	}
	items := scanCaches(context.Background(), caches)
	if len(items) != 2 || items[0].ID != "npm" || items[0].SizeBytes != 2000 || items[1].ID != "pip" {
		t.Fatalf("items = %+v, want npm (2000) then pip, with the missing cache left out", items)
	}
}

func TestDeleteOnlyAllowedFolders(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "npm-cache")
	docker := filepath.Join(root, "docker")
	modules := filepath.Join(root, "proj", "node_modules")
	source := filepath.Join(root, "proj", "src")
	writeFile(t, filepath.Join(cache, "a.tgz"), 1000)
	writeFile(t, filepath.Join(docker, "disk.vhdx"), 700)
	writeFile(t, filepath.Join(modules, "x", "i.js"), 200)
	writeFile(t, filepath.Join(source, "main.js"), 50)
	rustTarget := filepath.Join(root, "tool", "target")
	writeFile(t, filepath.Join(root, "tool", "Cargo.toml"), 10)
	writeFile(t, filepath.Join(rustTarget, "debug", "tool.exe"), 300)
	photos := filepath.Join(root, "photos", "target")
	writeFile(t, filepath.Join(photos, "holiday.jpg"), 900)

	caches := []knownCache{
		{"npm", "npm cache", cache, "", true},
		{"docker", "Docker data", docker, "", false},
	}
	result := deletePaths(context.Background(), []string{
		cache,
		modules,
		rustTarget,
		photos,         // named target, but not a Rust or Maven project
		docker,         // known but marked not deletable
		source,         // not a cache at all
		root,           // parent of everything
		"node_modules", // relative path
		filepath.Join(root, "proj", "node_modules", "..", "src"), // traversal back out of node_modules
	}, caches, nil)

	if result.FreedBytes != 1500 || len(result.Deleted) != 3 {
		t.Errorf("freed %d bytes from %v, want 1500 from the cache, node_modules and the Rust target", result.FreedBytes, result.Deleted)
	}
	if len(result.Failed) != 6 {
		t.Errorf("refused %d paths, want 6: %v", len(result.Failed), result.Failed)
	}
	if exists(cache) || exists(modules) || exists(rustTarget) {
		t.Error("allowed folders should be gone")
	}
	for _, keep := range []string{docker, source, filepath.Join(source, "main.js"), filepath.Join(root, "proj"), filepath.Join(photos, "holiday.jpg")} {
		if !exists(keep) {
			t.Errorf("%s must not be deleted", keep)
		}
	}
}

func TestDeleteReadOnlyFiles(t *testing.T) {
	root := t.TempDir()
	modules := filepath.Join(root, "node_modules")
	file := filepath.Join(modules, "pkg", "locked.js")
	writeFile(t, file, 100)
	if err := os.Chmod(file, 0o444); err != nil {
		t.Fatal(err)
	}

	result := deletePaths(context.Background(), []string{modules}, nil, nil)
	if len(result.Failed) != 0 || exists(modules) {
		t.Fatalf("read-only files should still be removed, failed = %v", result.Failed)
	}
}

func TestDeleteMissingFolderReported(t *testing.T) {
	result := deletePaths(context.Background(), []string{filepath.Join(t.TempDir(), "node_modules")}, nil, nil)
	if len(result.Failed) != 1 || !strings.Contains(result.Failed[0], "not found") {
		t.Errorf("failed = %v, want one 'folder not found'", result.Failed)
	}
}

func TestDeleteReportsProgress(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "a", "node_modules")
	second := filepath.Join(root, "b", "node_modules")
	writeFile(t, filepath.Join(first, "x", "one.js"), 300)
	writeFile(t, filepath.Join(first, "x", "two.js"), 200)
	writeFile(t, filepath.Join(second, "three.js"), 100)

	var reports []DeleteProgress
	result := deletePaths(context.Background(), []string{first, second}, nil, func(p DeleteProgress) {
		reports = append(reports, p)
	})

	if result.FreedBytes != 600 || len(result.Deleted) != 2 || exists(first) || exists(second) {
		t.Fatalf("result = %+v, want both folders gone and 600 bytes freed", result)
	}
	if len(reports) == 0 {
		t.Fatal("no progress reported")
	}
	if reports[0].Current != first || reports[0].Folder != 1 || reports[0].Folders != 2 {
		t.Errorf("first report = %+v, want folder 1 of 2 at %s", reports[0], first)
	}
	last := reports[len(reports)-1]
	if last.FreedBytes != 600 || last.FilesDeleted != 3 || last.Folder != 2 {
		t.Errorf("last report = %+v, want 600 bytes in 3 files at folder 2", last)
	}
}
