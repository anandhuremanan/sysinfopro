// Package relocate moves big caches and app data off the system drive, and
// back again.
//
// Each item moves the way its own tool expects, so nothing breaks afterwards:
//   - "setting": copy the folder, point the tool at the new place with its own
//     setting (npm config, go env -w, a user environment variable), check the
//     tool now reports the new place, then delete the old copy.
//   - "junction": for apps with no such setting (browsers, Maven, Cargo), copy
//     the folder and leave a folder link in the old place, so the app still
//     finds it there. Junctions need no administrator rights.
//   - "wsl": WSL's own `wsl --manage <name> --move <folder>`.
//
// Every move refuses to start while the program is running, leaves the
// original untouched if anything fails, and is remembered so it can be moved
// back. Programs installed in Program Files are not moved: that needs
// administrator rights and breaks updates; reinstalling to another drive is
// the safe way.
package relocate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"sysinfopro/internal/fsutil"
	"sysinfopro/internal/settings"
)

// Item is one thing that can be moved, as the screen shows it
type Item struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"` // dev | wsl | browser
	// Method is how it moves: setting | junction | wsl
	Method string `json:"method"`
	// Path is where it is now (for a moved item, where it was moved to)
	Path string `json:"path"`
	// SizeBytes is -1 until measured with FolderSize (WSL disks come with theirs)
	SizeBytes     int64 `json:"size_bytes"`
	OnSystemDrive bool  `json:"on_system_drive"`
	// Moved is true when SysInfo Pro moved it; it can be moved back to MovedFrom
	Moved     bool   `json:"moved"`
	MovedFrom string `json:"moved_from"`
	// Blocker says why it cannot be moved (or moved back) right now
	Blocker string `json:"blocker"`
	// Note is something to know before moving it
	Note string `json:"note"`
	// Guide is set for items that are only explained here, not moved
	Guide string `json:"guide"`
}

// Progress is reported while an item is copied
type Progress struct {
	ID    string `json:"id"`
	Phase string `json:"phase"` // copying | switching | cleaning
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
}

// Result describes a finished move
type Result struct {
	ID   string `json:"id"`
	To   string `json:"to"`
	Note string `json:"note"`
}

// record remembers a move, so it can be undone
type record struct {
	From   string    `json:"from"`
	To     string    `json:"to"`
	Method string    `json:"method"`
	When   time.Time `json:"when"`
}

const recordsFile = "relocations.json"

var recordsMu sync.Mutex

// where moves are remembered; tests keep them in memory instead
var (
	readRecords  = func() map[string]record { return settings.Load(recordsFile, map[string]record{}) }
	writeRecords = func(all map[string]record) error { return settings.Save(recordsFile, all) }
	// tests move between folders on one drive
	allowSameDrive = false
)

func loadRecords() map[string]record {
	recordsMu.Lock()
	defer recordsMu.Unlock()
	return readRecords()
}

func saveRecord(id string, r *record) error {
	recordsMu.Lock()
	defer recordsMu.Unlock()
	all := readRecords()
	if r == nil {
		delete(all, id)
	} else {
		all[id] = *r
	}
	return writeRecords(all)
}

// kind is how one item is found and moved
type kind struct {
	id, name, group string
	method          string // setting | junction | wsl
	// folder is the name of its folder at the destination
	folder string
	// locate returns where it is now; ok is false when it is not on this PC
	locate func(ctx context.Context) (path string, ok bool, err error)
	// point tells the tool to use dir (setting method only)
	point func(ctx context.Context, dir string) error
	// programs that must be closed first, as file name and the name people know
	procs [][2]string
	note  string
}

// Detect lists what can be moved. It runs each tool's own command to ask where
// its cache is, so it takes a second or two; sizes are measured separately
// with FolderSize, since a big cache can take a while to add up.
func Detect(ctx context.Context) []Item {
	records := loadRecords()
	running := runningPrograms()
	system := strings.ToUpper(os.Getenv("SystemDrive"))

	var mu sync.Mutex
	var items []Item
	var wg sync.WaitGroup
	for _, k := range kinds() {
		wg.Add(1)
		go func(k kind) {
			defer wg.Done()
			c, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			item, ok := describe(c, k, records, running, system)
			if ok {
				mu.Lock()
				items = append(items, item)
				mu.Unlock()
			}
		}(k)
	}
	wg.Wait()
	items = append(items, wslItems(ctx, records, system)...)

	order := map[string]int{"dev": 0, "wsl": 1, "browser": 2}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Group != items[j].Group {
			return order[items[i].Group] < order[items[j].Group]
		}
		return items[i].Name < items[j].Name
	})
	return items
}

func describe(ctx context.Context, k kind, records map[string]record, running map[string]bool, system string) (Item, bool) {
	item := Item{ID: k.id, Name: k.name, Group: k.group, Method: k.method, Note: k.note}
	path, ok, err := k.locate(ctx)
	rec, moved := records[k.id]
	if err != nil && !moved {
		item.Blocker = err.Error()
	}
	if !ok && !moved {
		return item, false
	}
	if path == "" && moved {
		path = rec.To
	}
	item.Path = path

	switch k.method {
	case "junction":
		if target, isLink := junctionTarget(path); isLink {
			if moved && samePath(target, rec.To) {
				item.Path, item.Moved, item.MovedFrom = rec.To, true, rec.From
			} else {
				item.Path = target
				item.Blocker = "Already a link to " + target + ", set up outside SysInfo Pro"
			}
		} else if moved {
			// the link is gone (the app or the user replaced it); forget the move
			saveRecord(k.id, nil)
		}
	case "setting":
		if moved && samePath(path, rec.To) {
			item.Moved, item.MovedFrom = true, rec.From
		}
	}

	if !isDir(item.Path) {
		return item, false
	}
	item.SizeBytes = -1
	item.OnSystemDrive = driveOf(item.Path) == system
	if item.Blocker == "" {
		item.Blocker = closeFirst(k.procs, running)
	}
	return item, true
}

// closeFirst names the programs that must be closed, or ""
func closeFirst(procs [][2]string, running map[string]bool) string {
	var names []string
	seen := map[string]bool{}
	for _, p := range procs {
		if running[strings.ToLower(p[0])] && !seen[p[1]] {
			seen[p[1]] = true
			names = append(names, p[1])
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "Close " + strings.Join(names, " and ") + " first"
}

func findKind(id string) (kind, bool) {
	for _, k := range kinds() {
		if k.id == id {
			return k, true
		}
	}
	return kind{}, false
}

// Move moves one item into a folder inside base (for example D:\Moved from C)
func Move(ctx context.Context, id, base string, progress func(Progress)) (*Result, error) {
	if strings.HasPrefix(id, "wsl:") {
		return moveWSL(ctx, strings.TrimPrefix(id, "wsl:"), base)
	}
	k, ok := findKind(id)
	if !ok {
		return nil, errors.New("unknown item")
	}
	item, ok := describe(ctx, k, loadRecords(), runningPrograms(), strings.ToUpper(os.Getenv("SystemDrive")))
	switch {
	case !ok:
		return nil, errors.New(k.name + " was not found on this PC")
	case item.Moved:
		return nil, errors.New(k.name + " has already been moved")
	case item.Blocker != "":
		return nil, errors.New(item.Blocker)
	}
	return moveKind(ctx, k, item.Path, filepath.Join(base, k.folder), dirSize(item.Path), progress)
}

// moveKind does the move once the item is known to be movable
func moveKind(ctx context.Context, k kind, src, dst string, size int64, progress func(Progress)) (*Result, error) {
	if err := checkTarget(src, dst, size); err != nil {
		return nil, err
	}
	report := func(phase string, done, total int64) {
		if progress != nil {
			progress(Progress{ID: k.id, Phase: phase, Done: done, Total: total})
		}
	}

	if err := copyTree(ctx, src, dst, func(done int64) { report("copying", done, size) }); err != nil {
		os.RemoveAll(dst)
		return nil, err
	}
	report("switching", size, size)

	result := &Result{ID: k.id, To: dst}
	switch k.method {
	case "setting":
		if err := k.point(ctx, dst); err != nil {
			os.RemoveAll(dst)
			return nil, fmt.Errorf("could not point %s at the new folder, so nothing changed: %w", k.name, err)
		}
		if now, _, _ := k.locate(ctx); !samePath(now, dst) {
			k.point(ctx, src)
			os.RemoveAll(dst)
			return nil, fmt.Errorf("%s still reports %s after the change, so it was put back and nothing changed", k.name, now)
		}
	case "junction":
		old := src + ".sysinfopro-old"
		if err := os.Rename(src, old); err != nil {
			os.RemoveAll(dst)
			return nil, fmt.Errorf("a program is still using %s, so nothing changed. Close it and try again", k.name)
		}
		if err := createJunction(src, dst); err != nil {
			os.Rename(old, src)
			os.RemoveAll(dst)
			return nil, fmt.Errorf("could not create the link, so nothing changed: %w", err)
		}
		src = old
	}

	if err := saveRecord(k.id, &record{From: strings.TrimSuffix(src, ".sysinfopro-old"), To: dst, Method: k.method, When: time.Now()}); err != nil {
		slog.Warn("could not remember a move", "id", k.id, "err", err)
	}
	report("cleaning", size, size)
	if skipped, _ := fsutil.RemoveTree(ctx, src, func(int64) {}); skipped > 0 {
		result.Note = fmt.Sprintf("%d old files on the system drive could not be removed (probably in use); they can be deleted from %s later", skipped, src)
	}
	slog.Info("moved", "id", k.id, "to", dst)
	return result, nil
}

// MoveBack returns a moved item to where it was
func MoveBack(ctx context.Context, id string, progress func(Progress)) (*Result, error) {
	rec, ok := loadRecords()[id]
	if !ok {
		return nil, errors.New("SysInfo Pro has no record of moving this")
	}
	if strings.HasPrefix(id, "wsl:") {
		return moveWSLBack(ctx, strings.TrimPrefix(id, "wsl:"), rec)
	}
	k, ok := findKind(id)
	if !ok {
		return nil, errors.New("unknown item")
	}
	if why := closeFirst(k.procs, runningPrograms()); why != "" {
		return nil, errors.New(why)
	}
	return moveKindBack(ctx, k, rec, progress)
}

func moveKindBack(ctx context.Context, k kind, rec record, progress func(Progress)) (*Result, error) {
	size := dirSize(rec.To)
	if free, ok := fsutil.FreeSpace(rec.From); ok && free < uint64(size)+256<<20 {
		return nil, fmt.Errorf("%s does not have enough free space to take it back (needs about %d MB)", driveOf(rec.From), (size+256<<20)>>20)
	}
	report := func(phase string, done int64) {
		if progress != nil {
			progress(Progress{ID: k.id, Phase: phase, Done: done, Total: size})
		}
	}

	switch k.method {
	case "junction":
		target, isLink := junctionTarget(rec.From)
		if !isLink || !samePath(target, rec.To) {
			return nil, errors.New("the link SysInfo Pro made is no longer there, so nothing was changed")
		}
		if err := os.Remove(rec.From); err != nil {
			return nil, fmt.Errorf("could not remove the link: %w", err)
		}
		if err := copyTree(ctx, rec.To, rec.From, func(done int64) { report("copying", done) }); err != nil {
			os.RemoveAll(rec.From)
			createJunction(rec.From, rec.To)
			return nil, err
		}
	case "setting":
		if err := copyTree(ctx, rec.To, rec.From, func(done int64) { report("copying", done) }); err != nil {
			return nil, err
		}
		report("switching", size)
		if err := k.point(ctx, rec.From); err != nil {
			return nil, fmt.Errorf("could not point %s back: %w", k.name, err)
		}
	}

	saveRecord(k.id, nil)
	report("cleaning", size)
	result := &Result{ID: k.id, To: rec.From}
	if skipped, _ := fsutil.RemoveTree(ctx, rec.To, func(int64) {}); skipped > 0 {
		result.Note = fmt.Sprintf("%d files could not be removed from %s; they can be deleted later", skipped, rec.To)
	}
	removeIfEmpty(filepath.Dir(rec.To))
	return result, nil
}

// checkTarget refuses destinations where the data would be lost, or that
// would not free anything
func checkTarget(src, dst string, size int64) error {
	if !filepath.IsAbs(dst) || strings.HasPrefix(dst, `\\`) {
		return errors.New("choose a folder on a drive, such as D:\\")
	}
	if !allowSameDrive && !fsutil.IsFixedDrive(dst) {
		return errors.New("choose a drive that is always connected; USB sticks, memory cards and network drives are not supported")
	}
	if !allowSameDrive && strings.EqualFold(driveOf(src), driveOf(dst)) {
		return errors.New("that folder is on the same drive, so moving there would not free any space")
	}
	if entries, err := os.ReadDir(dst); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s already has files in it", dst)
	}
	if free, ok := fsutil.FreeSpace(dst); ok && free < uint64(size+size/10)+256<<20 {
		return fmt.Errorf("%s does not have enough free space (needs about %d MB)", driveOf(dst), (size+size/10+256<<20)>>20)
	}
	return nil
}

// copyTree copies src into dst with each file's size checked. Links inside
// are not followed or copied: a folder holding one is refused instead.
func copyTree(ctx context.Context, src, dst string, onBytes func(done int64)) error {
	var done int64
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("could not read %s: %w", path, err)
		}
		if ctx.Err() != nil {
			return errors.New("stopped; nothing was changed")
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s contains a link (%s), which cannot be moved safely, so nothing was changed", src, rel)
		}
		n, err := copyFile(path, target)
		if err != nil {
			return fmt.Errorf("could not copy %s: %w", rel, err)
		}
		done += n
		if onBytes != nil {
			onBytes(done)
		}
		return nil
	})
}

func copyFile(from, to string) (int64, error) {
	in, err := os.Open(from)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return 0, err
	}
	out, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, err
	}
	n, err := out.ReadFrom(in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return n, err
	}
	if n != info.Size() {
		return n, fmt.Errorf("copied %d of %d bytes", n, info.Size())
	}
	os.Chtimes(to, info.ModTime(), info.ModTime())
	if info.Mode().Perm()&0o200 == 0 {
		os.Chmod(to, 0o444) // keep read-only files read-only, as Go's module cache expects
	}
	return n, nil
}

// FolderSize adds up the files in a folder, without following links
func FolderSize(root string) int64 {
	return dirSize(root)
}

func dirSize(root string) int64 {
	var total int64
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func removeIfEmpty(dir string) {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		os.Remove(dir)
	}
}

func driveOf(path string) string {
	return strings.ToUpper(filepath.VolumeName(path))
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
