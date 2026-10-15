// Package devcache finds and removes developer tool caches and node_modules folders.
package devcache

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const nodeModules = "node_modules"

type Item struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"` // cache | node_modules
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Files     int64  `json:"files"`
	Note      string `json:"note"`
	Deletable bool   `json:"deletable"`
	// When the owning project was last touched (node_modules only), YYYY-MM-DD
	LastUsed string `json:"last_used"`
}

type DeleteResult struct {
	FreedBytes int64    `json:"freed_bytes"`
	Deleted    []string `json:"deleted"`
	Failed     []string `json:"failed"` // "path: reason"
}

// DeleteProgress is reported while Delete works through its folders
type DeleteProgress struct {
	Current      string `json:"current"` // folder being deleted
	Folder       int    `json:"folder"`  // 1-based position of Current
	Folders      int    `json:"folders"`
	FreedBytes   int64  `json:"freed_bytes"` // across all folders so far
	FilesDeleted int64  `json:"files_deleted"`
}

type knownCache struct {
	id, name, path, note string
	deletable            bool
}

// knownCaches lists the caches of common package managers and build tools
func knownCaches() []knownCache {
	home, _ := os.UserHomeDir()
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		local = filepath.Join(home, "AppData", "Local")
	}

	goModCache, goBuildCache := goEnvDirs()
	if goModCache == "" {
		goPath := os.Getenv("GOPATH")
		if goPath == "" {
			goPath = filepath.Join(home, "go")
		}
		goModCache = filepath.Join(filepath.SplitList(goPath)[0], "pkg", "mod")
	}
	if goBuildCache == "" {
		goBuildCache = filepath.Join(local, "go-build")
	}

	const redownload = "Packages are downloaded again when a project needs them"
	return []knownCache{
		{"npm", "npm cache", filepath.Join(local, "npm-cache"), redownload, true},
		{"yarn", "Yarn cache", filepath.Join(local, "Yarn", "Cache"), redownload, true},
		{"pnpm-store", "pnpm store", filepath.Join(local, "pnpm", "store"), "Projects re-link packages from a fresh store on next install", true},
		{"pnpm-cache", "pnpm cache", filepath.Join(local, "pnpm-cache"), redownload, true},
		{"pip", "pip cache", filepath.Join(local, "pip", "Cache"), redownload, true},
		{"nuget", "NuGet packages", filepath.Join(home, ".nuget", "packages"), "Restored on the next build", true},
		{"gradle", "Gradle caches", filepath.Join(home, ".gradle", "caches"), "Rebuilt on the next Gradle build, which will be slower once", true},
		{"maven", "Maven repository", filepath.Join(home, ".m2", "repository"), "Dependencies are downloaded again on the next build", true},
		{"cargo", "Cargo registry", filepath.Join(home, ".cargo", "registry"), "Crates are downloaded again on the next build", true},
		{"go-build", "Go build cache", goBuildCache, "Rebuilt on the next go build, which will be slower once", true},
		{"go-mod", "Go module cache", goModCache, "Modules are downloaded again when needed", true},
		{"docker", "Docker Desktop data", filepath.Join(local, "Docker", "wsl"), "Images and containers. Clean these from Docker itself (docker system prune)", false},
	}
}

// dirSize totals the regular files under root without following links
func dirSize(ctx context.Context, root string) (size, files int64) {
	filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
				files++
			}
		}
		return nil
	})
	return size, files
}

func isDir(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}

// ScanCaches sizes every known cache that exists on this machine, largest first
func ScanCaches(ctx context.Context) []Item {
	return scanCaches(ctx, knownCaches())
}

func scanCaches(ctx context.Context, caches []knownCache) []Item {
	items := []Item{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, cache := range caches {
		if !isDir(cache.path) {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			size, files := dirSize(ctx, cache.path)
			mu.Lock()
			defer mu.Unlock()
			items = append(items, Item{
				ID: cache.id, Name: cache.name, Kind: "cache", Path: cache.path,
				SizeBytes: size, Files: files, Note: cache.note, Deletable: cache.deletable,
			})
		}()
	}
	wg.Wait()

	sort.Slice(items, func(i, j int) bool { return items[i].SizeBytes > items[j].SizeBytes })
	return items
}

// FindNodeModules lists every top-level node_modules folder under root, largest first.
// progress receives the folder currently being searched.
func FindNodeModules(ctx context.Context, root string, progress func(current string)) ([]Item, error) {
	if !isDir(root) {
		return nil, fmt.Errorf("%s is not a folder", root)
	}

	items := []Item{}
	lastReport := time.Now()

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if progress != nil && time.Since(lastReport) > 200*time.Millisecond {
			lastReport = time.Now()
			progress(path)
		}
		if d.Name() != nodeModules {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		project := filepath.Dir(path)
		size, files := dirSize(ctx, path)
		items = append(items, Item{
			ID: path, Name: filepath.Base(project), Kind: nodeModules, Path: path,
			SizeBytes: size, Files: files, Deletable: true,
			Note:     "Restored by running your package manager's install in the project",
			LastUsed: projectLastUsed(project),
		})
		// nested node_modules are already counted in this one
		return filepath.SkipDir
	})

	sort.Slice(items, func(i, j int) bool { return items[i].SizeBytes > items[j].SizeBytes })
	if ctx.Err() != nil {
		return items, ctx.Err()
	}
	return items, nil
}

// Files whose last change shows when a project was last worked on: its manifest
// and lock files, the markers npm, pnpm and yarn write into node_modules on every
// install, and git's index, which changes on commits, checkouts and status.
var lastUsedMarkers = []string{
	"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb", "bun.lock",
	filepath.Join(nodeModules, ".package-lock.json"),
	filepath.Join(nodeModules, ".modules.yaml"),
	filepath.Join(nodeModules, ".yarn-state.yml"),
	filepath.Join(nodeModules, ".yarn-integrity"),
	filepath.Join(".git", "index"),
}

// projectLastUsed is the most recent change to any of the project's lastUsedMarkers
func projectLastUsed(project string) string {
	latest := time.Time{}
	for _, name := range lastUsedMarkers {
		if info, err := os.Stat(filepath.Join(project, name)); err == nil && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.Format("2006-01-02")
}

// Delete removes the given folders. Only known deletable caches and folders named
// node_modules are accepted; anything else is refused without touching the disk.
// progress, if set, is called a few times a second while files are removed.
func Delete(ctx context.Context, paths []string, progress func(DeleteProgress)) *DeleteResult {
	return deletePaths(ctx, paths, knownCaches(), progress)
}

func deletePaths(ctx context.Context, paths []string, caches []knownCache, progress func(DeleteProgress)) *DeleteResult {
	allowed := map[string]bool{}
	for _, cache := range caches {
		if cache.deletable {
			allowed[strings.ToLower(filepath.Clean(cache.path))] = true
		}
	}

	result := &DeleteResult{Deleted: []string{}, Failed: []string{}}
	state := DeleteProgress{Folders: len(paths)}
	lastReport := time.Time{}
	report := func(force bool) {
		if progress != nil && (force || time.Since(lastReport) > 100*time.Millisecond) {
			lastReport = time.Now()
			progress(state)
		}
	}

	for i, raw := range paths {
		if ctx.Err() != nil {
			result.Failed = append(result.Failed, raw+": cancelled")
			continue
		}

		path := filepath.Clean(raw)
		switch {
		case !filepath.IsAbs(path):
			result.Failed = append(result.Failed, raw+": not an absolute path")
			continue
		case !allowed[strings.ToLower(path)] && filepath.Base(path) != nodeModules:
			result.Failed = append(result.Failed, raw+": not a known cache or node_modules folder")
			continue
		case !isDir(path):
			result.Failed = append(result.Failed, raw+": folder not found")
			continue
		}

		state.Current, state.Folder = path, i+1
		report(true)
		err := removeTree(ctx, path, func(size int64) {
			state.FreedBytes += size
			state.FilesDeleted++
			report(false)
		})
		if err != nil {
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", raw, err))
			continue
		}
		result.Deleted = append(result.Deleted, path)
	}

	// a partial delete still frees space, so count every file that went
	result.FreedBytes = state.FreedBytes
	report(true)
	return result
}

// removeTree deletes a folder file by file so progress can be shown, calling
// onFile with the size of each removed file. Links and junctions are removed
// themselves, never followed. Read-only files (the Go module cache is written
// read-only on purpose) are made writable first.
func removeTree(ctx context.Context, root string, onFile func(size int64)) error {
	var dirs []string
	var firstErr error
	remember := func(err error) {
		if firstErr == nil {
			firstErr = err
		}
	}

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil {
			remember(err)
			if d != nil && d.IsDir() {
				dirs = append(dirs, path)
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}

		var size int64
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() {
			size = info.Size()
		}
		if err := removeOne(path); err != nil {
			remember(err)
			return nil
		}
		onFile(size)
		return nil
	})

	// deepest folders first, so each is empty by the time it is removed
	for i := len(dirs) - 1; i >= 0; i-- {
		if ctx.Err() != nil {
			break
		}
		if err := removeOne(dirs[i]); err != nil {
			remember(err)
		}
	}

	if ctx.Err() != nil {
		return fmt.Errorf("stopped")
	}
	if _, err := os.Lstat(root); err == nil {
		if firstErr == nil {
			firstErr = fmt.Errorf("some files could not be removed")
		}
		return firstErr
	}
	return nil
}

func removeOne(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	if os.Chmod(path, 0o666) == nil {
		if err2 := os.Remove(path); err2 == nil || os.IsNotExist(err2) {
			return nil
		}
	}
	return err
}
