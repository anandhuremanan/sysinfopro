// Package logging writes the app's log to a small rotating file, so problems on
// a user's PC can be diagnosed, and keeps panics in background work from
// silently killing the app.
//
// The log lives in %LOCALAPPDATA%\SysInfoPro\logs: app.log, plus up to two older
// files (app.log.1, app.log.2), each capped at 1 MB. It holds what went wrong and
// when, never file contents or personal data beyond paths the app was asked about.
package logging

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
)

const (
	fileName = "app.log"
	maxBytes = 1 << 20 // per file
	keep     = 2       // older files kept besides the current one
)

var (
	mu      sync.Mutex
	logDir  string
	current *rotatingFile
)

// Dir is where the log files are kept
func Dir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		if d, err := os.UserCacheDir(); err == nil {
			base = d
		} else {
			base = os.TempDir()
		}
	}
	return filepath.Join(base, "SysInfoPro", "logs")
}

// Init sends slog's default logger to the log file in dir ("" for Dir()). If the
// file cannot be opened, logging stays on standard error and the app carries on.
func Init(dir string) error {
	if dir == "" {
		dir = Dir()
	}
	f, err := openRotating(dir)
	if err != nil {
		return err
	}
	mu.Lock()
	if current != nil {
		current.Close()
	}
	logDir, current = dir, f
	mu.Unlock()
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo})))
	return nil
}

// Go runs fn in a goroutine whose panics are logged instead of ending the app
func Go(name string, fn func()) {
	go func() {
		defer Recover(name)
		fn()
	}()
}

// Recover, deferred at the top of a goroutine, logs a panic with its stack
// instead of letting it end the app
func Recover(name string) {
	if r := recover(); r != nil {
		slog.Error("background task crashed", "task", name, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
	}
}

// Tail returns up to n of the most recent log lines, oldest first
func Tail(n int) []string {
	mu.Lock()
	dir := logDir
	if current != nil {
		current.Sync()
	}
	mu.Unlock()
	if dir == "" {
		dir = Dir()
	}
	var lines []string
	// newest file last, so the result stays in time order
	for i := keep; i >= 0; i-- {
		name := filepath.Join(dir, fileName)
		if i > 0 {
			name = fmt.Sprintf("%s.%d", name, i)
		}
		f, err := os.Open(name)
		if err != nil {
			continue
		}
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 64*1024), 1<<20)
		for s.Scan() {
			lines = append(lines, s.Text())
		}
		f.Close()
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// rotatingFile appends to app.log and starts a new file when it reaches maxBytes
type rotatingFile struct {
	mu   sync.Mutex
	dir  string
	f    *os.File
	size int64
}

func openRotating(dir string) (*rotatingFile, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	r := &rotatingFile{dir: dir}
	return r, r.open()
}

func (r *rotatingFile) path() string { return filepath.Join(r.dir, fileName) }

func (r *rotatingFile) open() error {
	f, err := os.OpenFile(r.path(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	r.f, r.size = f, info.Size()
	return nil
}

func (r *rotatingFile) rotate() error {
	r.f.Close()
	for i := keep; i >= 1; i-- {
		older := fmt.Sprintf("%s.%d", r.path(), i)
		newer := r.path()
		if i > 1 {
			newer = fmt.Sprintf("%s.%d", r.path(), i-1)
		}
		os.Remove(older)
		os.Rename(newer, older)
	}
	return r.open()
}

func (r *rotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return 0, io.ErrClosedPipe
	}
	if r.size+int64(len(p)) > maxBytes && r.size > 0 {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

func (r *rotatingFile) Sync() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f != nil {
		r.f.Sync()
	}
}

func (r *rotatingFile) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f != nil {
		r.f.Close()
		r.f = nil
	}
}

// Text is a short, shareable diagnostics report: version, Windows build and the
// most recent log lines
func Text(version, windows string, lines int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "SysInfo Pro %s\nWindows: %s\nLog folder: %s\n\n", version, windows, Dir())
	tail := Tail(lines)
	if len(tail) == 0 {
		b.WriteString("(the log is empty)\n")
	}
	for _, l := range tail {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return b.String()
}

// Close closes the log file; later log lines go to standard error
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		current.Close()
		current = nil
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
}
