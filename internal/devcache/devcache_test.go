package devcache

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	}, caches)

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

	result := deletePaths(context.Background(), []string{modules}, nil)
	if len(result.Failed) != 0 || exists(modules) {
		t.Fatalf("read-only files should still be removed, failed = %v", result.Failed)
	}
}

func TestDeleteMissingFolderReported(t *testing.T) {
	result := deletePaths(context.Background(), []string{filepath.Join(t.TempDir(), "node_modules")}, nil)
	if len(result.Failed) != 1 || !strings.Contains(result.Failed[0], "not found") {
		t.Errorf("failed = %v, want one 'folder not found'", result.Failed)
	}
}
