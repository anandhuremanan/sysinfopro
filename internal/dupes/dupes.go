// Package dupes finds files with exactly the same contents. Files are first
// grouped by size, then by a hash of their first and last 64 KB, and only the
// files still alike are read in full, so most files are never read at all.
package dupes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type File struct {
	Path     string `json:"path"`
	Modified int64  `json:"modified"` // Unix milliseconds
}

type Group struct {
	ID    string `json:"id"` // the contents' hash
	Size  int64  `json:"size"`
	Files []File `json:"files"` // oldest first
	// space the extra copies take: size x (copies - 1)
	Wasted int64 `json:"wasted"`
	// program parts (libraries, drivers): copies are often needed where they are
	Risky bool `json:"risky"`
}

// parts of programs; a copy next to each program that loads it is normal
var programExts = map[string]bool{
	".dll": true, ".so": true, ".dylib": true, ".sys": true, ".pyd": true, ".node": true,
	".ocx": true, ".drv": true, ".mui": true, ".jar": true, ".pdb": true, ".lib": true, ".a": true,
}

func isProgramPart(path string) bool {
	return programExts[strings.ToLower(filepath.Ext(path))]
}

type Result struct {
	Root   string  `json:"root"`
	Groups []Group `json:"groups"`
	// files looked at, and bytes that could be freed by keeping one of each
	Files  int   `json:"files"`
	Wasted int64 `json:"wasted"`
	// online-only cloud files (OneDrive and the like) are skipped: reading one downloads it
	CloudSkipped int   `json:"cloud_skipped"`
	Unreadable   int   `json:"unreadable"`
	Truncated    bool  `json:"truncated"` // only the biggest groups are listed
	DurationMs   int64 `json:"duration_ms"`
}

// Place is a suggested folder to search
type Place struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type Progress struct {
	Phase   string `json:"phase"` // listing | comparing
	Files   int    `json:"files"` // files found so far
	Done    int64  `json:"done"`  // bytes compared so far
	Total   int64  `json:"total"` // bytes that need comparing
	Current string `json:"current"`
}

const (
	sampleSize = 64 * 1024
	maxGroups  = 2000
	workers    = 4
)

// folders never searched: deleting a "duplicate" in them breaks a program or a project
var skipDirs = map[string]bool{
	"windows": true, "program files": true, "program files (x86)": true, "programdata": true,
	"$recycle.bin": true, "system volume information": true, "recovery": true, "$windows.~bt": true,
	"appdata": true, "node_modules": true, ".git": true, ".svn": true, ".hg": true, ".venv": true,
	"venv": true, "__pycache__": true, "site-packages": true, ".gradle": true, ".m2": true,
	".nuget": true, ".cargo": true, ".rustup": true, "target": true, "vendor": true, ".idea": true,
	".vs": true, "obj": true, ".pnpm-store": true, ".npm": true, ".cache": true,
	"windowsapps": true, "steamapps": true,
}

func skipDir(name string) bool {
	return skipDirs[strings.ToLower(name)]
}

type candidate struct {
	path     string
	size     int64
	modified int64
	info     fs.FileInfo
}

// Find looks for duplicate files of at least minSize bytes under root
func Find(ctx context.Context, root string, minSize int64, onProgress func(Progress)) (*Result, error) {
	began := time.Now()
	if minSize < 1 {
		minSize = 1
	}
	res := &Result{Root: root, Groups: []Group{}}

	// 1. list files, by size
	bySize := map[int64][]candidate{}
	last := time.Now()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != root && (skipDir(d.Name()) || d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0) {
				return fs.SkipDir
			}
			if onProgress != nil && time.Since(last) > 200*time.Millisecond {
				last = time.Now()
				onProgress(Progress{Phase: "listing", Files: res.Files, Current: path})
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		res.Files++
		if info.Size() < minSize {
			return nil
		}
		if isCloudOnly(info) {
			res.CloudSkipped++
			return nil
		}
		bySize[info.Size()] = append(bySize[info.Size()], candidate{path, info.Size(), info.ModTime().UnixMilli(), info})
		return nil
	})
	if err != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// 2. sizes shared by two or more files; hard links to one file are one file
	var sets [][]candidate
	var total int64
	for _, files := range bySize {
		if len(files) < 2 {
			continue
		}
		files = dropHardLinks(files)
		if len(files) < 2 {
			continue
		}
		sets = append(sets, files)
		for _, f := range files {
			total += min(f.size, 2*sampleSize)
		}
	}

	var done atomic.Int64
	var unreadable atomic.Int64
	report := func(current string) {
		if onProgress != nil && time.Since(last) > 200*time.Millisecond {
			last = time.Now()
			onProgress(Progress{Phase: "comparing", Files: res.Files, Done: done.Load(), Total: total, Current: current})
		}
	}

	// 3. split each set by a quick sample, then by the full contents
	var partial [][]candidate
	for _, set := range sets {
		groups := splitBy(ctx, set, func(c candidate) (string, error) {
			defer done.Add(min(c.size, 2*sampleSize))
			return hashSample(c.path, c.size)
		}, &unreadable, report)
		partial = append(partial, groups...)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// the full read is only needed when the sample did not already cover the whole file
	for _, set := range partial {
		if set[0].size > 2*sampleSize {
			total += set[0].size * int64(len(set))
		}
	}
	for _, set := range partial {
		var groups [][]candidate
		if set[0].size <= 2*sampleSize {
			groups = [][]candidate{set}
		} else {
			groups = splitBy(ctx, set, func(c candidate) (string, error) {
				h, err := hashFull(ctx, c.path, &done)
				return h, err
			}, &unreadable, report)
		}
		for _, g := range groups {
			res.Groups = append(res.Groups, makeGroup(g))
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	sort.Slice(res.Groups, func(i, j int) bool { return res.Groups[i].Wasted > res.Groups[j].Wasted })
	for _, g := range res.Groups {
		res.Wasted += g.Wasted
	}
	if len(res.Groups) > maxGroups {
		res.Groups = res.Groups[:maxGroups]
		res.Truncated = true
	}
	res.Unreadable = int(unreadable.Load())
	res.DurationMs = time.Since(began).Milliseconds()
	return res, nil
}

// splitBy hashes files in parallel and keeps the hashes shared by two or more
func splitBy(ctx context.Context, set []candidate, hash func(candidate) (string, error), unreadable *atomic.Int64, report func(string)) [][]candidate {
	hashes := make([]string, len(set))
	var wg sync.WaitGroup
	next := atomic.Int64{}
	var mu sync.Mutex
	for w := 0; w < min(workers, len(set)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1) - 1)
				if i >= len(set) || ctx.Err() != nil {
					return
				}
				h, err := hash(set[i])
				if err != nil {
					unreadable.Add(1)
					continue
				}
				hashes[i] = h
				mu.Lock()
				report(set[i].path)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	byHash := map[string][]candidate{}
	var order []string
	for i, h := range hashes {
		if h == "" {
			continue
		}
		if _, ok := byHash[h]; !ok {
			order = append(order, h)
		}
		byHash[h] = append(byHash[h], set[i])
	}
	var out [][]candidate
	for _, h := range order {
		if len(byHash[h]) > 1 {
			out = append(out, byHash[h])
		}
	}
	return out
}

func makeGroup(set []candidate) Group {
	sort.Slice(set, func(i, j int) bool { return set[i].modified < set[j].modified })
	g := Group{Size: set[0].size, Wasted: set[0].size * int64(len(set)-1)}
	for _, c := range set {
		g.Files = append(g.Files, File{Path: c.path, Modified: c.modified})
		if isProgramPart(c.path) {
			g.Risky = true
		}
	}
	sum := sha256.Sum256([]byte(fmt.Sprint(set[0].size, set[0].path)))
	g.ID = hex.EncodeToString(sum[:8])
	return g
}

// dropHardLinks keeps one name for each file that has several (hard links take no extra space)
func dropHardLinks(files []candidate) []candidate {
	if len(files) > 64 {
		return files // checking every pair would be slow; a rare case not worth it
	}
	kept := files[:0:0]
outer:
	for _, f := range files {
		for _, k := range kept {
			if os.SameFile(f.info, k.info) {
				continue outer
			}
		}
		kept = append(kept, f)
	}
	return kept
}

func hashSample(path string, size int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.CopyN(h, f, min(size, sampleSize)); err != nil {
		return "", err
	}
	if size > 2*sampleSize {
		if _, err := f.Seek(-sampleSize, io.SeekEnd); err != nil {
			return "", err
		}
		if _, err := io.CopyN(h, f, sampleSize); err != nil {
			return "", err
		}
	} else if size > sampleSize {
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashFull(ctx context.Context, path string, done *atomic.Int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 1<<20)
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		n, err := f.Read(buf)
		h.Write(buf[:n])
		done.Add(int64(n))
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ErrWouldRemoveAll refuses a removal that leaves no copy of some file
var ErrWouldRemoveAll = errors.New("that would remove every copy of a file; at least one copy is always kept")

// CheckKeepsOne makes sure removing paths leaves at least one copy in every group
func CheckKeepsOne(groups []Group, paths []string) error {
	removing := map[string]bool{}
	for _, p := range paths {
		removing[strings.ToLower(p)] = true
	}
	known := 0
	for _, g := range groups {
		touched, kept := false, false
		for _, f := range g.Files {
			if removing[strings.ToLower(f.Path)] {
				known++
				touched = true
				continue
			}
			// the copy that stays must still be there, unchanged; it may have
			// been moved or deleted since the search
			if info, err := os.Stat(f.Path); err == nil && info.Size() == g.Size {
				kept = true
			}
		}
		if touched && !kept {
			return ErrWouldRemoveAll
		}
	}
	if known != len(removing) {
		return errors.New("some of those files were not part of the last search; search again")
	}
	return nil
}
