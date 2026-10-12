package storage

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
)

const (
	scanWorkers      = 6
	largestFilesKept = 25
	progressInterval = 250 * time.Millisecond
)

type DriveInfo struct {
	Path       string  `json:"path"`
	FSType     string  `json:"fs_type"`
	TotalBytes uint64  `json:"total_bytes"`
	FreeBytes  uint64  `json:"free_bytes"`
	UsedBytes  uint64  `json:"used_bytes"`
	UsedPct    float64 `json:"used_pct"`
}

type UsageEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsDir     bool   `json:"is_dir"`
	SizeBytes int64  `json:"size_bytes"`
	Files     int64  `json:"files"`
}

type FolderUsage struct {
	Path         string       `json:"path"`
	TotalBytes   int64        `json:"total_bytes"`
	Files        int64        `json:"files"`
	Skipped      int64        `json:"skipped"` // items that could not be read (e.g. access denied)
	Entries      []UsageEntry `json:"entries"`
	LargestFiles []UsageEntry `json:"largest_files"`
	DurationMs   int64        `json:"duration_ms"`
	Cancelled    bool         `json:"cancelled"`
}

type ScanProgress struct {
	Files   int64  `json:"files"`
	Bytes   int64  `json:"bytes"`
	Current string `json:"current"`
}

type ScanProgressFunc func(ScanProgress)

// RootPath turns a bare drive letter ("C:") into its root ("C:\").
// A bare letter means "the current directory on that drive", which Windows can
// resolve to a different drive entirely, so sizes must always be asked of the root.
func RootPath(mount string) string {
	if len(mount) == 2 && mount[1] == ':' {
		return mount + string(filepath.Separator)
	}
	return mount
}

// ListDrives returns every mounted partition with its capacity
func ListDrives() ([]DriveInfo, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	drives := []DriveInfo{}
	for _, p := range partitions {
		usage, err := disk.Usage(RootPath(p.Mountpoint))
		if err != nil || usage.Total == 0 {
			continue
		}
		drives = append(drives, DriveInfo{
			Path:       p.Mountpoint,
			FSType:     p.Fstype,
			TotalBytes: usage.Total,
			FreeBytes:  usage.Free,
			UsedBytes:  usage.Used,
			UsedPct:    usage.UsedPercent,
		})
	}
	return drives, nil
}

// largestFiles keeps the biggest files seen during a scan
type largestFiles struct {
	mu    sync.Mutex
	items []UsageEntry
	floor int64 // smallest size currently kept once the list is full
}

func (l *largestFiles) offer(name, path string, size int64) {
	if size <= atomic.LoadInt64(&l.floor) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	l.items = append(l.items, UsageEntry{Name: name, Path: path, SizeBytes: size, Files: 1})
	if len(l.items) > largestFilesKept*2 {
		l.trim()
	}
}

func (l *largestFiles) trim() {
	sort.Slice(l.items, func(i, j int) bool { return l.items[i].SizeBytes > l.items[j].SizeBytes })
	if len(l.items) > largestFilesKept {
		l.items = l.items[:largestFilesKept]
		atomic.StoreInt64(&l.floor, l.items[len(l.items)-1].SizeBytes)
	}
}

// AnalyzeFolder sizes every item directly inside root, walking sub-folders recursively.
// Links and junctions are not followed, so nothing is counted twice.
func AnalyzeFolder(ctx context.Context, root string, progress ScanProgressFunc) (*FolderUsage, error) {
	start := time.Now()
	root = RootPath(filepath.Clean(root))

	children, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", root, err)
	}

	var files, bytes, skipped int64
	var current atomic.Value
	current.Store(root)
	largest := &largestFiles{}

	// Live progress while the walk runs
	stopProgress := make(chan struct{})
	var progressWG sync.WaitGroup
	if progress != nil {
		progressWG.Add(1)
		go func() {
			defer progressWG.Done()
			ticker := time.NewTicker(progressInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					progress(ScanProgress{
						Files:   atomic.LoadInt64(&files),
						Bytes:   atomic.LoadInt64(&bytes),
						Current: current.Load().(string),
					})
				case <-stopProgress:
					return
				}
			}
		}()
	}

	entries := make([]UsageEntry, len(children))
	sem := make(chan struct{}, scanWorkers)
	var wg sync.WaitGroup

	for i, child := range children {
		path := filepath.Join(root, child.Name())
		entry := &entries[i]
		entry.Name = child.Name()
		entry.Path = path

		if !child.IsDir() {
			// file, link or junction directly in root
			info, err := child.Info()
			if err != nil {
				atomic.AddInt64(&skipped, 1)
				continue
			}
			if info.Mode().IsRegular() {
				entry.SizeBytes = info.Size()
				entry.Files = 1
				atomic.AddInt64(&files, 1)
				atomic.AddInt64(&bytes, info.Size())
				largest.offer(entry.Name, path, info.Size())
			}
			continue
		}

		entry.IsDir = true
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
				if ctx.Err() != nil {
					return filepath.SkipAll
				}
				if walkErr != nil {
					atomic.AddInt64(&skipped, 1)
					if d != nil && d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if d.IsDir() {
					current.Store(p)
					return nil
				}
				if !d.Type().IsRegular() {
					return nil
				}
				info, err := d.Info()
				if err != nil {
					atomic.AddInt64(&skipped, 1)
					return nil
				}
				size := info.Size()
				entry.SizeBytes += size
				entry.Files++
				atomic.AddInt64(&files, 1)
				atomic.AddInt64(&bytes, size)
				largest.offer(d.Name(), p, size)
				return nil
			})
		}()
	}

	wg.Wait()
	close(stopProgress)
	progressWG.Wait()

	sort.Slice(entries, func(i, j int) bool { return entries[i].SizeBytes > entries[j].SizeBytes })

	largest.mu.Lock()
	largest.trim()
	largestItems := largest.items
	largest.mu.Unlock()
	if largestItems == nil {
		largestItems = []UsageEntry{}
	}

	return &FolderUsage{
		Path:         root,
		TotalBytes:   atomic.LoadInt64(&bytes),
		Files:        atomic.LoadInt64(&files),
		Skipped:      atomic.LoadInt64(&skipped),
		Entries:      entries,
		LargestFiles: largestItems,
		DurationMs:   time.Since(start).Milliseconds(),
		Cancelled:    ctx.Err() != nil,
	}, nil
}

// RevealInFileManager opens the system file manager with the item selected
func RevealInFileManager(path string) error {
	path = filepath.Clean(path)
	if _, err := os.Lstat(path); err != nil {
		return fmt.Errorf("cannot find %s: %w", path, err)
	}

	// explorer returns a non-zero exit code even on success, so only Start is checked
	return revealCommand(path).Start()
}
