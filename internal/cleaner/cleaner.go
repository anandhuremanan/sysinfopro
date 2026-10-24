// Package cleaner finds and removes temporary files and browser caches in the
// user's own folders. Nothing here needs administrator rights.
package cleaner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"myproject/internal/fsutil"
)

// Files changed this recently are left alone: a program is probably still using them
const recentWindow = time.Hour

// A Location is one folder whose contents can be cleared
type Location struct {
	Category string // shown to the user, e.g. "Temporary files"
	Path     string
}

type Category struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	Items     int    `json:"items"` // top-level files and folders
}

type ScanResult struct {
	Categories []Category `json:"categories"`
	TotalBytes int64      `json:"total_bytes"`
}

type Progress struct {
	Category     string `json:"category"`
	FreedBytes   int64  `json:"freed_bytes"`
	TotalBytes   int64  `json:"total_bytes"`
	FilesDeleted int64  `json:"files_deleted"`
}

type Summary struct {
	FreedBytes   int64 `json:"freed_bytes"`
	FilesDeleted int64 `json:"files_deleted"`
	// files that could not be removed, nearly always because a program has them open
	Skipped int  `json:"skipped"`
	Stopped bool `json:"stopped"`
}

// DefaultLocations are the user's temp folder and the caches of the common
// browsers. Only cache folders are listed, never profiles, so history,
// passwords and settings are never touched.
func DefaultLocations() []Location {
	locations := []Location{{"Temporary files", os.TempDir()}}
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return locations
	}
	for _, browser := range []struct{ name, userData string }{
		{"Chrome cache", filepath.Join(local, "Google", "Chrome", "User Data")},
		{"Edge cache", filepath.Join(local, "Microsoft", "Edge", "User Data")},
		{"Brave cache", filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data")},
	} {
		// every profile ("Default", "Profile 1", ...) has its own caches
		caches, _ := filepath.Glob(filepath.Join(browser.userData, "*", "Cache"))
		for _, cache := range caches {
			locations = append(locations,
				Location{browser.name, cache},
				Location{browser.name, filepath.Join(filepath.Dir(cache), "Code Cache")})
		}
	}
	firefox, _ := filepath.Glob(filepath.Join(local, "Mozilla", "Firefox", "Profiles", "*", "cache2"))
	for _, cache := range firefox {
		locations = append(locations, Location{"Firefox cache", cache})
	}
	return locations
}

// an entry is one file or folder directly inside a location
type entry struct {
	category string
	path     string
	size     int64
}

// entries lists what can be cleared: everything directly inside each location
// except items changed within recentWindow
func entries(ctx context.Context, locations []Location) []entry {
	cutoff := time.Now().Add(-recentWindow)
	seen := map[string]bool{}
	var found []entry
	for _, loc := range locations {
		dir := filepath.Clean(loc.Path)
		if seen[dir] {
			continue // the same folder reached two ways is listed once
		}
		seen[dir] = true

		items, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, item := range items {
			if ctx.Err() != nil {
				return found
			}
			info, err := item.Info()
			if err != nil || info.ModTime().After(cutoff) {
				continue
			}
			path := filepath.Join(dir, item.Name())
			size := info.Size()
			if item.IsDir() {
				size = treeSize(ctx, path, cutoff)
				if size < 0 {
					continue // something inside is in use right now
				}
			}
			found = append(found, entry{loc.Category, path, size})
		}
	}
	return found
}

// treeSize totals a folder, or returns -1 if anything inside changed after cutoff
func treeSize(ctx context.Context, root string, cutoff time.Time) int64 {
	var size int64
	recent := false
	filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || ctx.Err() != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			if info.ModTime().After(cutoff) {
				recent = true
				return filepath.SkipAll
			}
			if info.Mode().IsRegular() {
				size += info.Size()
			}
		}
		return nil
	})
	if recent {
		return -1
	}
	return size
}

// Scan totals what Clean would remove, by category, in the order of locations
func Scan(ctx context.Context, locations []Location) ScanResult {
	result := ScanResult{Categories: []Category{}}
	index := map[string]int{}
	for _, e := range entries(ctx, locations) {
		i, ok := index[e.category]
		if !ok {
			i = len(result.Categories)
			index[e.category] = i
			result.Categories = append(result.Categories, Category{Name: e.category})
		}
		result.Categories[i].SizeBytes += e.size
		result.Categories[i].Items++
		result.TotalBytes += e.size
	}
	return result
}

// Clean removes what Scan finds, reporting progress a few times a second
func Clean(ctx context.Context, locations []Location, progress func(Progress)) Summary {
	found := entries(ctx, locations)
	state := Progress{}
	for _, e := range found {
		state.TotalBytes += e.size
	}

	summary := Summary{}
	lastReport := time.Time{}
	report := func(force bool) {
		if progress != nil && (force || time.Since(lastReport) > 100*time.Millisecond) {
			lastReport = time.Now()
			progress(state)
		}
	}
	onFile := func(size int64) {
		state.FreedBytes += size
		state.FilesDeleted++
		report(false)
	}

	for _, e := range found {
		if ctx.Err() != nil {
			break
		}
		if state.Category != e.category {
			state.Category = e.category
			report(true)
		}

		info, err := os.Lstat(e.path)
		if err != nil {
			continue // already gone
		}
		if !info.IsDir() {
			if fsutil.RemoveOne(e.path) != nil {
				summary.Skipped++
				continue
			}
			onFile(e.size)
			continue
		}
		skipped, _ := fsutil.RemoveTree(ctx, e.path, onFile)
		summary.Skipped += skipped
	}

	summary.FreedBytes = state.FreedBytes
	summary.FilesDeleted = state.FilesDeleted
	summary.Stopped = ctx.Err() != nil
	report(true)
	return summary
}
