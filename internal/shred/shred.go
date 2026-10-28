// Package shred overwrites files before deleting them, so ordinary recovery
// tools cannot bring them back, and wipes a drive's free space.
//
// What overwriting can promise depends on the drive. On a spinning hard drive
// one pass of random data replaces the old contents for good. On an SSD the
// drive decides where data really goes and keeps spare space the system cannot
// reach, so an overwrite may land somewhere new and leave the old blocks until
// the drive reuses them. The screens say so plainly; Reach describes a drive.
package shred

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Methods are the pass counts offered: one random pass is enough for any
// drive; three and seven passes are for people who want them on old hard drives
var Methods = map[int][]pattern{
	1: {random},
	3: {zeros, ones, random},
	7: {zeros, ones, random, zeros, ones, random, random},
}

type pattern int

const (
	random pattern = iota
	zeros
	ones
)

const chunk = 1 << 20

// Progress is reported a few times a second while files are overwritten
type Progress struct {
	File       string `json:"file"`
	FileIndex  int    `json:"file_index"`
	Files      int    `json:"files"`
	Pass       int    `json:"pass"`
	Passes     int    `json:"passes"`
	DoneBytes  int64  `json:"done_bytes"`
	TotalBytes int64  `json:"total_bytes"`
}

// Skip is a file that was left alone, and why
type Skip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Result says what was shredded
type Result struct {
	Shredded int    `json:"shredded"`
	Bytes    int64  `json:"bytes"`
	Skipped  []Skip `json:"skipped"`
	// Notes are things the user should know about files that were shredded
	Notes   []string `json:"notes"`
	Stopped bool     `json:"stopped"`
}

// a file found under the chosen paths
type target struct {
	path string
	size int64
}

// Shred overwrites every file in paths (folders are walked, links are never
// followed) with the given number of passes, then renames and deletes it.
// Folders left empty afterwards are removed too.
func Shred(ctx context.Context, paths []string, passes int, progress func(Progress)) *Result {
	result := &Result{Skipped: []Skip{}, Notes: []string{}}
	method, ok := Methods[passes]
	if !ok {
		result.Skipped = append(result.Skipped, Skip{"", fmt.Sprintf("%d passes is not one of the offered methods", passes)})
		return result
	}

	var files []target
	var dirs []string
	var total int64
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			result.Skipped = append(result.Skipped, Skip{p, "not found"})
			continue
		}
		if !info.IsDir() {
			if !info.Mode().IsRegular() {
				result.Skipped = append(result.Skipped, Skip{p, "a link or special file, which is never followed"})
				continue
			}
			files = append(files, target{p, info.Size()})
			total += info.Size()
			continue
		}
		filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				result.Skipped = append(result.Skipped, Skip{path, "could not be read"})
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				dirs = append(dirs, path)
				return nil
			}
			if !d.Type().IsRegular() {
				result.Skipped = append(result.Skipped, Skip{path, "a link or special file, which is never followed"})
				return nil
			}
			if info, err := d.Info(); err == nil {
				files = append(files, target{path, info.Size()})
				total += info.Size()
			}
			return nil
		})
	}

	state := Progress{Files: len(files), Passes: len(method), TotalBytes: total * int64(len(method))}
	report := throttle(progress)
	for i, f := range files {
		if ctx.Err() != nil {
			result.Stopped = true
			break
		}
		state.File, state.FileIndex = f.path, i+1
		note, err := shredFile(ctx, f.path, method, func(pass int, wrote int64) {
			state.Pass = pass
			state.DoneBytes += wrote
			report(state, false)
		})
		if errors.Is(err, context.Canceled) {
			result.Stopped = true
			result.Skipped = append(result.Skipped, Skip{f.path, "stopped part way; its contents are already partly overwritten"})
			break
		}
		if err != nil {
			result.Skipped = append(result.Skipped, Skip{f.path, err.Error()})
			continue
		}
		if note != "" {
			result.Notes = append(result.Notes, filepath.Base(f.path)+": "+note)
		}
		result.Shredded++
		result.Bytes += f.size
	}
	report(state, true)

	// deepest folders first; a folder still holding a skipped file stays
	for i := len(dirs) - 1; i >= 0; i-- {
		os.Remove(dirs[i])
	}
	return result
}

// fill writes one pass's pattern into buf
func fill(buf []byte, p pattern) error {
	switch p {
	case zeros:
		clear(buf)
	case ones:
		for i := range buf {
			buf[i] = 0xFF
		}
	default:
		if _, err := rand.Read(buf); err != nil {
			return err
		}
	}
	return nil
}

func throttle(progress func(Progress)) func(Progress, bool) {
	if progress == nil {
		return func(Progress, bool) {}
	}
	var last int64
	return func(p Progress, force bool) {
		step := p.TotalBytes / 200
		if force || p.DoneBytes-last >= step || step == 0 {
			last = p.DoneBytes
			progress(p)
		}
	}
}

// randomName is a name that says nothing about the file, of the same length
// as name, so the old name does not survive in the folder's records
func randomName(name string) string {
	n := len(name)
	if n < 8 {
		n = 8
	}
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

// Protected reports why a path must not be shredded, or "" when it may be:
// whole drives, Windows and program folders, and the top of the user's
// profile are refused outright
func Protected(path string) string {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return "not a full path"
	}
	if filepath.Dir(clean) == clean {
		return "a whole drive cannot be shredded; choose files or a folder"
	}
	within := func(root string) bool {
		if root == "" {
			return false
		}
		rel, err := filepath.Rel(root, clean)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	for _, env := range []string{"WINDIR", "ProgramFiles", "ProgramFiles(x86)", "ProgramData"} {
		if within(os.Getenv(env)) {
			return "Windows and program folders cannot be shredded"
		}
	}
	for _, env := range []string{"USERPROFILE", "APPDATA", "LOCALAPPDATA", "OneDrive"} {
		if root := os.Getenv(env); root != "" && strings.EqualFold(filepath.Clean(root), clean) {
			return "this whole folder is too broad to shred; choose the files inside it"
		}
	}
	return ""
}
