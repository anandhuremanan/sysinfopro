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

	"sysinfopro/internal/fsutil"
)

const nodeModules = "node_modules"

type Item struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"` // cache | leftover
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Files     int64  `json:"files"`
	Note      string `json:"note"`
	Deletable bool   `json:"deletable"`
	// For leftovers: what kind of folder it is ("Rust build") and its folder name
	// ("target"); empty for caches
	Label  string `json:"label"`
	Folder string `json:"folder"`
	// When the owning project was last touched (leftovers only), YYYY-MM-DD
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
		{"pnpm-store", "pnpm store", filepath.Join(local, "pnpm", "store"), "Its files are shared with your projects' node_modules, so space only comes back for packages no project uses. Projects re-link on next install", true},
		{"pnpm-cache", "pnpm cache", filepath.Join(local, "pnpm-cache"), redownload, true},
		{"pip", "pip cache", filepath.Join(local, "pip", "Cache"), redownload, true},
		{"nuget", "NuGet packages", filepath.Join(home, ".nuget", "packages"), "Restored on the next build", true},
		{"gradle", "Gradle caches", filepath.Join(home, ".gradle", "caches"), "Rebuilt on the next Gradle build, which will be slower once", true},
		{"maven", "Maven repository", filepath.Join(home, ".m2", "repository"), "Dependencies are downloaded again on the next build", true},
		{"cargo", "Cargo registry", filepath.Join(home, ".cargo", "registry"), "Crates are downloaded again on the next build", true},
		goCache("go-build", "Go build cache", goBuildCache, "Rebuilt on the next go build, which will be slower once"),
		goCache("go-mod", "Go module cache", goModCache, "Modules are downloaded again when needed"),
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

// FindLeftovers lists every project leftover (node_modules, Rust target, Python
// virtual environments and the rest of leftoverRules) under root, largest first.
// progress receives the folder currently being searched.
func FindLeftovers(ctx context.Context, root string, progress func(current string)) ([]Item, error) {
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
		if !d.IsDir() || path == root {
			return nil
		}
		if progress != nil && time.Since(lastReport) > 200*time.Millisecond {
			lastReport = time.Now()
			progress(path)
		}
		if d.Name() == ".git" {
			return filepath.SkipDir
		}
		rule, ok := matchLeftover(path)
		if !ok {
			return nil
		}

		project := filepath.Dir(path)
		size, files := dirSize(ctx, path)
		items = append(items, Item{
			ID: path, Name: filepath.Base(project), Kind: "leftover", Path: path,
			SizeBytes: size, Files: files, Deletable: true,
			Label: rule.label, Folder: d.Name(), Note: rule.note,
			LastUsed: projectLastUsed(project, path),
		})
		// anything nested inside is already counted in this one
		return filepath.SkipDir
	})

	sort.Slice(items, func(i, j int) bool { return items[i].SizeBytes > items[j].SizeBytes })
	if ctx.Err() != nil {
		return items, ctx.Err()
	}
	return items, nil
}

// projectLastUsed is the most recent change to the project's lastUsedMarkers, its
// .NET project files, or the leftover folder itself
func projectLastUsed(project, leftover string) string {
	paths := []string{leftover}
	for _, name := range lastUsedMarkers {
		paths = append(paths, filepath.Join(project, name))
	}
	for _, pattern := range []string{"*.csproj", "*.fsproj", "*.vbproj"} {
		matches, _ := filepath.Glob(filepath.Join(project, pattern))
		paths = append(paths, matches...)
	}

	latest := time.Time{}
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.Format("2006-01-02")
}

// Delete removes the given folders. Only known deletable caches and project
// leftovers (checked again here against leftoverRules) are accepted; anything
// else is refused without touching the disk.
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
		case !allowed[strings.ToLower(path)] && !isLeftover(path):
			result.Failed = append(result.Failed, raw+": not a known cache or project leftover")
			continue
		case !isDir(path):
			result.Failed = append(result.Failed, raw+": folder not found")
			continue
		}

		state.Current, state.Folder = path, i+1
		report(true)
		_, err := fsutil.RemoveTree(ctx, path, func(size int64) {
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

func isLeftover(path string) bool {
	_, ok := matchLeftover(path)
	return ok
}

// goCache lists a Go cache, but only offers it for deletion when the folder
// really looks like one. Its location comes from the Go settings, and a
// mistaken GOMODCACHE, GOCACHE or GOPATH could otherwise point "delete the
// cache" at a folder full of other files.
func goCache(id, name, path, note string) knownCache {
	if !goCacheLooksRight(id, path) {
		return knownCache{id, name, path, "This folder does not look like Go's own cache, so it is not offered for deletion", false}
	}
	return knownCache{id, name, path, note, true}
}

func goCacheLooksRight(id, path string) bool {
	clean := strings.TrimRight(filepath.Clean(path), `\/`)
	if clean == "" || filepath.Dir(clean) == clean {
		return false // empty or a whole drive
	}
	if home, err := os.UserHomeDir(); err == nil && strings.EqualFold(clean, strings.TrimRight(filepath.Clean(home), `\/`)) {
		return false
	}
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(clean, rel))
		return err == nil
	}
	switch id {
	case "go-mod":
		// .../pkg/mod, holding Go's download cache
		return strings.EqualFold(filepath.Base(clean), "mod") || exists(filepath.Join("cache", "download"))
	case "go-build":
		// go-build, or a folder with the marker files Go writes into its build cache
		return strings.EqualFold(filepath.Base(clean), "go-build") || exists("trim.txt") || exists("README")
	}
	return false
}
