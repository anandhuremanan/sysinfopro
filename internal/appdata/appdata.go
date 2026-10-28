// Package appdata decides where SysInfo Pro keeps its own files, and moves
// them to another folder when the user asks.
//
// By default settings, history and the embedded browser's data live in
// %APPDATA%\SysInfoPro and the logs in %LOCALAPPDATA%\SysInfoPro\logs. After a
// move everything lives in one folder the user chose (<folder>\SysInfoPro),
// with the browser data in WebView2 and the logs in logs below it. A small
// pointer file, location.json, always stays in %APPDATA%\SysInfoPro so the app
// can find the data again.
//
// A move cannot happen while the app runs, because the embedded browser holds
// its files open. It is recorded as pending and carried out by Init the next
// time the app starts, before the window opens.
package appdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	folderName  = "SysInfoPro"
	pointerName = "location.json"
	webviewName = "WebView2"
	logsName    = "logs"
)

// the embedded browser's caches; it rebuilds them, so a move skips them
var webviewCaches = map[string]bool{
	"cache": true, "code cache": true, "gpucache": true, "grshadercache": true,
	"shadercache": true, "dawncache": true, "dawngraphitecache": true,
	"dawnwebgpucache": true, "crashpad": true,
}

// where the default folders are; tests point these at a temporary folder
var (
	defaultRoot = func() string {
		if dir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(dir, folderName)
		}
		return filepath.Join(os.TempDir(), folderName)
	}
	defaultLogs = func() string {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			if dir, err := os.UserCacheDir(); err == nil {
				base = dir
			} else {
				base = os.TempDir()
			}
		}
		return filepath.Join(base, folderName, logsName)
	}
)

// pointer is the content of location.json
type pointer struct {
	// Dir is the chosen folder; "" means the default places
	Dir string `json:"dir,omitempty"`
	// Pending is set when a move waits for the next start
	Pending *pendingMove `json:"pending,omitempty"`
}

type pendingMove struct {
	// To is the folder to move to; "" means back to the default places
	To string `json:"to"`
}

// Status describes the data folder for the Settings screen
type Status struct {
	Dir        string `json:"dir"`
	IsDefault  bool   `json:"is_default"`
	DefaultDir string `json:"default_dir"`
	SizeBytes  int64  `json:"size_bytes"`
	// a move waiting for the next start, and where to ("" = the default folder)
	Pending   bool   `json:"pending"`
	PendingTo string `json:"pending_to"`
	// Notice reports what happened at this start: a finished move, a failed one,
	// or a chosen folder that could not be found
	Notice      string `json:"notice"`
	NoticeIsErr bool   `json:"notice_is_error"`
}

var (
	mu       sync.Mutex
	ready    bool
	active   pointer // what this run uses
	notice   string
	noticeOK bool
	// the data folder was empty when this run started: the app is newly installed
	fresh bool
)

// Init reads location.json and carries out a pending move. Call it once at
// start, before anything reads or writes the app's files.
func Init() {
	mu.Lock()
	defer mu.Unlock()
	ready = true
	defer func() { fresh = isEmpty(rootOf(active.Dir)) }()
	p, err := readPointer()
	if err != nil {
		setNotice(false, "Could not read where SysInfo Pro keeps its files, so the default folder is used: "+err.Error())
		active = pointer{}
		return
	}
	if p.Pending != nil {
		p = applyMove(p)
	}
	if p.Dir != "" {
		if info, err := os.Stat(p.Dir); err != nil || !info.IsDir() {
			setNotice(false, fmt.Sprintf("The data folder %s could not be found (is its drive connected?), so the default folder is used for now.", p.Dir))
			active = pointer{}
			return
		}
	}
	active = pointer{Dir: p.Dir}
}

func setNotice(ok bool, text string) {
	notice, noticeOK = text, ok
}

func ensure() {
	if !ready {
		mu.Unlock()
		Init()
		mu.Lock()
	}
}

// Dir is the folder for settings, history and other small files
func Dir() string {
	mu.Lock()
	defer mu.Unlock()
	ensure()
	return rootOf(active.Dir)
}

// LogDir is the folder for the log files
func LogDir() string {
	mu.Lock()
	defer mu.Unlock()
	ensure()
	return logsOf(active.Dir)
}

// WebViewDir is the embedded browser's data folder
func WebViewDir() string {
	return filepath.Join(Dir(), webviewName)
}

func rootOf(dir string) string {
	if dir == "" {
		return defaultRoot()
	}
	return dir
}

func logsOf(dir string) string {
	if dir == "" {
		return defaultLogs()
	}
	return filepath.Join(dir, logsName)
}

// FreshInstall is true when the data folder held nothing (but the pointer)
// before this run: the app has never run on this PC under this user
func FreshInstall() bool {
	mu.Lock()
	defer mu.Unlock()
	ensure()
	return fresh
}

func isEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.Is(err, fs.ErrNotExist)
	}
	for _, e := range entries {
		if !strings.EqualFold(e.Name(), pointerName) {
			return false
		}
	}
	return true
}

// GetStatus reports the data folder, its size and any pending move
func GetStatus() Status {
	mu.Lock()
	ensure()
	dir := active.Dir
	text, ok := notice, noticeOK
	mu.Unlock()

	s := Status{
		Dir:         rootOf(dir),
		IsDefault:   dir == "",
		DefaultDir:  defaultRoot(),
		SizeBytes:   dataSize(dir),
		Notice:      text,
		NoticeIsErr: text != "" && !ok,
	}
	if p, err := readPointer(); err == nil && p.Pending != nil {
		s.Pending = true
		s.PendingTo = p.Pending.To
		if s.PendingTo == "" {
			s.PendingTo = defaultRoot()
		}
	}
	return s
}

// RequestMove checks the folder the user picked and records a move to a
// SysInfoPro folder inside it, carried out at the next start
func RequestMove(picked string) (string, error) {
	target, err := checkTarget(picked)
	if err != nil {
		return "", err
	}
	if err := setPending(&pendingMove{To: target}); err != nil {
		return "", err
	}
	return target, nil
}

// RequestDefault records a move back to the default folders
func RequestDefault() error {
	mu.Lock()
	ensure()
	dir := active.Dir
	mu.Unlock()
	if dir == "" {
		// already on the default folder (perhaps because the chosen one went
		// missing): forget the chosen folder and any pending move
		return writePointer(pointer{})
	}
	return setPending(&pendingMove{To: ""})
}

// CancelMove forgets a pending move
func CancelMove() error {
	return setPending(nil)
}

func setPending(move *pendingMove) error {
	p, err := readPointer()
	if err != nil {
		return err
	}
	mu.Lock()
	// the pointer may name a folder this run could not find; keep it
	if p.Dir == "" {
		p.Dir = active.Dir
	}
	mu.Unlock()
	p.Pending = move
	return writePointer(p)
}

// checkTarget turns the picked folder into the folder the data would move to,
// and refuses places where it would be lost, unreadable or in the way
func checkTarget(picked string) (string, error) {
	picked = strings.TrimSpace(picked)
	if picked == "" {
		return "", errors.New("no folder was chosen")
	}
	if strings.HasPrefix(picked, `\\`) || strings.HasPrefix(picked, "//") {
		return "", errors.New("network folders are not supported: the data would be lost whenever the network is unavailable")
	}
	if !filepath.IsAbs(picked) {
		return "", errors.New("choose a folder on a drive, such as D:\\")
	}
	picked = filepath.Clean(picked)
	if !isFixedDrive(picked) {
		return "", errors.New("choose a folder on a drive that is always connected; USB sticks, memory cards and network drives are not supported")
	}
	for _, env := range []string{"WINDIR", "ProgramFiles", "ProgramFiles(x86)"} {
		if root := os.Getenv(env); root != "" && within(picked, root) {
			return "", errors.New("Windows and program folders cannot hold app data; choose another folder")
		}
	}

	target := picked
	if !strings.EqualFold(filepath.Base(picked), folderName) {
		target = filepath.Join(picked, folderName)
	}

	mu.Lock()
	ensure()
	current := rootOf(active.Dir)
	mu.Unlock()
	if strings.EqualFold(target, current) {
		return "", errors.New("SysInfo Pro already keeps its files there")
	}
	if within(target, current) || within(current, target) {
		return "", errors.New("choose a folder outside the current data folder")
	}
	if strings.EqualFold(target, defaultRoot()) {
		return "", errors.New("that is the default folder; use \"Use default\" instead")
	}

	created := false
	if entries, err := os.ReadDir(target); err == nil {
		if len(entries) > 0 {
			return "", fmt.Errorf("%s already has files in it; choose an empty folder or another place", target)
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(target, 0o755); err != nil {
			return "", fmt.Errorf("could not create %s: %w", target, err)
		}
		created = true
	} else {
		return "", fmt.Errorf("could not open %s: %w", target, err)
	}
	// try a write, then leave the folder as it was
	probe := filepath.Join(target, ".write-test")
	werr := os.WriteFile(probe, []byte("ok"), 0o644)
	os.Remove(probe)
	if created {
		os.Remove(target)
	}
	if werr != nil {
		return "", fmt.Errorf("SysInfo Pro cannot write to %s: %w", target, werr)
	}

	if free, ok := freeSpace(picked); ok {
		mu.Lock()
		dir := active.Dir
		mu.Unlock()
		if need := dataSize(dir) + 64<<20; free < uint64(need) {
			return "", fmt.Errorf("that drive does not have enough free space (needs about %d MB)", need>>20)
		}
	}
	return target, nil
}

// within is true when path is root or inside it
func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

// a copy job: the folder to copy, where to, and what to leave behind
type part struct {
	from, to string
	skip     func(rel string, isDir bool) bool
}

// partsFor lists what to copy for a move from dir to newDir
func partsFor(dir, newDir string) []part {
	fromRoot, toRoot := rootOf(dir), rootOf(newDir)
	fromLogs, toLogs := logsOf(dir), logsOf(newDir)
	return []part{
		{from: fromRoot, to: toRoot, skip: func(rel string, isDir bool) bool {
			lower := strings.ToLower(rel)
			if lower == pointerName || lower == logsName {
				return true // the pointer stays put; logs are their own part
			}
			// the browser's caches are rebuilt; WebView2\<profile>\<cache> or WebView2\<cache>
			if isDir && strings.HasPrefix(lower, strings.ToLower(webviewName)+string(filepath.Separator)) {
				return webviewCaches[strings.ToLower(filepath.Base(rel))]
			}
			return false
		}},
		{from: fromLogs, to: toLogs},
	}
}

// how long to wait before each new try at copying. Right after a restart the
// old window's browser processes can hold its files open for a few seconds.
var retryWaits = []time.Duration{time.Second, 2 * time.Second, 3 * time.Second}

// applyMove copies the data to the pending folder, switches to it and removes
// the old copy. Anything that fails leaves the data where it was.
func applyMove(p pointer) pointer {
	from, to := p.Dir, p.Pending.To
	p.Pending = nil
	parts := partsFor(from, to)

	err := copyParts(parts)
	for _, wait := range retryWaits {
		if err == nil {
			break
		}
		time.Sleep(wait)
		err = copyParts(parts)
	}
	if err != nil {
		if to != "" {
			removeEmptyDirs(rootOf(to))
		}
		writePointer(p)
		setNotice(false, fmt.Sprintf("Moving the data to %s did not work, so it stays in %s: %v", rootOf(to), rootOf(from), err))
		return p
	}

	p.Dir = to
	if err := writePointer(p); err != nil {
		setNotice(false, "The data was copied but SysInfo Pro could not switch to it, so it stays where it was: "+err.Error())
		p.Dir = from
		return p
	}

	// the new copy is in use; clear out the old one
	for _, pt := range parts {
		removeOld(pt.from, pt.skip)
	}
	if from != "" {
		removeEmptyDirs(rootOf(from))
	} else {
		os.Remove(defaultLogs())
		os.Remove(filepath.Dir(defaultLogs()))
	}
	setNotice(true, "SysInfo Pro's files were moved to "+rootOf(to)+".")
	return p
}

// copyParts copies every part, or, if anything fails, removes what it copied
// and returns the error; the old folders are never touched
func copyParts(parts []part) error {
	var files, dirs []string
	for _, pt := range parts {
		f, d, err := copyTree(pt.from, pt.to, pt.skip)
		files, dirs = append(files, f...), append(dirs, d...)
		if err != nil {
			for _, name := range files {
				os.Remove(name)
			}
			// newest first, so inner folders go before the folders holding them
			for i := len(dirs) - 1; i >= 0; i-- {
				os.Remove(dirs[i]) // only removes it if empty
			}
			return err
		}
	}
	return nil
}

// copyTree copies from into to, file by file, and checks each copy's size. It
// returns the files it wrote and the folders it created. A missing source
// folder is not an error: there is nothing to copy yet.
func copyTree(from, to string, skip func(string, bool) bool) (written, created []string, err error) {
	mkdir := func(dir string) error {
		if _, err := os.Stat(dir); err == nil {
			return nil
		}
		// record every level that does not exist yet
		var missing []string
		for d := dir; ; d = filepath.Dir(d) {
			if _, err := os.Stat(d); err == nil || filepath.Dir(d) == d {
				break
			}
			missing = append(missing, d)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for i := len(missing) - 1; i >= 0; i-- {
			created = append(created, missing[i])
		}
		return nil
	}
	err = filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == from && errors.Is(err, fs.ErrNotExist) {
				return filepath.SkipDir
			}
			return err
		}
		rel, _ := filepath.Rel(from, path)
		if rel == "." {
			return mkdir(to)
		}
		if skip != nil && skip(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return mkdir(target)
		}
		if !d.Type().IsRegular() {
			return nil // links and devices are not app data
		}
		if err := copyFile(path, target); err != nil {
			return fmt.Errorf("copying %s: %w", rel, err)
		}
		written = append(written, target)
		return nil
	})
	return written, created, err
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n != info.Size() {
		return fmt.Errorf("copied %d of %d bytes", n, info.Size())
	}
	os.Chtimes(to, info.ModTime(), info.ModTime())
	return nil
}

// removeOld deletes the old copy: everything the move took, and the browser
// caches it left behind
func removeOld(from string, skip func(string, bool) bool) {
	entries, err := os.ReadDir(from)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		lower := strings.ToLower(name)
		if lower == pointerName {
			continue
		}
		if skip != nil && lower == logsName && skip(name, true) {
			continue // the logs part removes these
		}
		os.RemoveAll(filepath.Join(from, name))
	}
	if skip == nil {
		os.Remove(from)
	}
}

// removeEmptyDirs removes dir if nothing is left in it
func removeEmptyDirs(dir string) {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		os.Remove(dir)
	}
}

// dataSize adds up what a move would copy from the folders for dir
func dataSize(dir string) int64 {
	var total int64
	for _, pt := range partsFor(dir, dir) {
		filepath.WalkDir(pt.from, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			rel, _ := filepath.Rel(pt.from, path)
			if rel != "." && pt.skip != nil && pt.skip(rel, d.IsDir()) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.IsDir() {
				if info, err := d.Info(); err == nil {
					total += info.Size()
				}
			}
			return nil
		})
	}
	return total
}

func pointerPath() string {
	return filepath.Join(defaultRoot(), pointerName)
}

func readPointer() (pointer, error) {
	var p pointer
	data, err := os.ReadFile(pointerPath())
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return pointer{}, fmt.Errorf("%s is not valid: %w", pointerName, err)
	}
	return p, nil
}

// writePointer saves location.json atomically, or removes it when it says
// nothing beyond the defaults
func writePointer(p pointer) error {
	path := pointerPath()
	if p.Dir == "" && p.Pending == nil {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), pointerName+".*.tmp")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp.Name())
		return errors.Join(werr, cerr)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// reset is for tests
func reset() {
	mu.Lock()
	ready, active, notice, noticeOK, fresh = false, pointer{}, "", false, false
	mu.Unlock()
}
