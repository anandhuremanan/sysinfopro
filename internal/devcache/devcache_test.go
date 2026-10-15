package devcache

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestFindNodeModules(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app-a", "package.json"), 10)
	writeFile(t, filepath.Join(root, "app-a", "node_modules", "x", "index.js"), 4000)
	// nested node_modules belongs to app-a's total, not a separate result
	writeFile(t, filepath.Join(root, "app-a", "node_modules", "x", "node_modules", "y", "index.js"), 1000)
	writeFile(t, filepath.Join(root, "deep", "er", "app-b", "node_modules", "z.js"), 300)
	writeFile(t, filepath.Join(root, "app-c", "src", "main.js"), 50)
	// folders inside .git are never searched
	writeFile(t, filepath.Join(root, "app-c", ".git", "node_modules", "ignored.js"), 9999)

	items, err := FindNodeModules(context.Background(), root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("found %d node_modules folders, want 2: %+v", len(items), items)
	}
	if items[0].Name != "app-a" || items[0].SizeBytes != 5000 || items[0].Files != 2 {
		t.Errorf("first = %+v, want app-a with 5000 bytes in 2 files", items[0])
	}
	if items[0].LastUsed == "" {
		t.Error("app-a has a package.json, so LastUsed should be set")
	}
	if items[1].Name != "app-b" || items[1].SizeBytes != 300 || items[1].LastUsed != "" {
		t.Errorf("second = %+v, want app-b with 300 bytes and no LastUsed", items[1])
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
	// an untouched package.json must not hide a recent install
	if err := os.Chtimes(manifest, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(installMarker, recent, recent); err != nil {
		t.Fatal(err)
	}

	if got := projectLastUsed(project); got != "2026-09-20" {
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

	caches := []knownCache{
		{"npm", "npm cache", cache, "", true},
		{"docker", "Docker data", docker, "", false},
	}
	result := deletePaths(context.Background(), []string{
		cache,
		modules,
		docker,         // known but marked not deletable
		source,         // not a cache at all
		root,           // parent of everything
		"node_modules", // relative path
		filepath.Join(root, "proj", "node_modules", "..", "src"), // traversal back out of node_modules
	}, caches, nil)

	if result.FreedBytes != 1200 || len(result.Deleted) != 2 {
		t.Errorf("freed %d bytes from %v, want 1200 from the cache and node_modules", result.FreedBytes, result.Deleted)
	}
	if len(result.Failed) != 5 {
		t.Errorf("refused %d paths, want 5: %v", len(result.Failed), result.Failed)
	}
	if exists(cache) || exists(modules) {
		t.Error("allowed folders should be gone")
	}
	for _, keep := range []string{docker, source, filepath.Join(source, "main.js"), filepath.Join(root, "proj")} {
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

func TestRemoveTreeStopsWhenCancelled(t *testing.T) {
	root := t.TempDir()
	modules := filepath.Join(root, "node_modules")
	for i := 0; i < 20; i++ {
		writeFile(t, filepath.Join(modules, "pkg", string(rune('a'+i))+".js"), 10)
	}

	ctx, cancel := context.WithCancel(context.Background())
	removed := 0
	err := removeTree(ctx, modules, func(int64) {
		removed++
		cancel()
	})
	if err == nil || !strings.Contains(err.Error(), "stopped") {
		t.Fatalf("err = %v, want stopped", err)
	}
	if removed != 1 || !exists(modules) {
		t.Errorf("removed %d files, want 1 and the rest of the folder left in place", removed)
	}
}

func TestDeleteDoesNotFollowLinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "my-project", "src")
	keep := filepath.Join(outside, "main.js")
	writeFile(t, keep, 50)
	modules := filepath.Join(root, "node_modules")
	writeFile(t, filepath.Join(modules, "real.js"), 10)
	// pnpm links packages with junctions on Windows, which need no special privilege
	link := filepath.Join(modules, "linked")
	if err := os.Symlink(outside, link); err != nil {
		if runtime.GOOS != "windows" {
			t.Skipf("cannot create symlinks here: %v", err)
		}
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput(); err != nil {
			t.Skipf("cannot create a junction here: %v %s", err, out)
		}
	}

	result := deletePaths(context.Background(), []string{modules}, nil, nil)
	if len(result.Failed) != 0 || exists(modules) {
		t.Fatalf("failed = %v, want node_modules removed", result.Failed)
	}
	if !exists(keep) {
		t.Fatal("a file outside node_modules was deleted through a link")
	}
}
