package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRotationKeepsAFewSmallFiles(t *testing.T) {
	dir := t.TempDir()
	r, err := openRotating(dir)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Repeat("x", 1000) + "\n"
	for i := 0; i < 4000; i++ { // about 4 MB
		if _, err := r.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	r.Close()
	matches, _ := filepath.Glob(filepath.Join(dir, "app.log*"))
	if len(matches) != keep+1 {
		t.Errorf("files = %v", matches)
	}
	for _, m := range matches {
		if info, _ := os.Stat(m); info.Size() > maxBytes {
			t.Errorf("%s is %d bytes", m, info.Size())
		}
	}
}

func TestRecoverLogsPanic(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	var wg sync.WaitGroup
	wg.Add(1)
	Go("test task", func() {
		defer wg.Done()
		panic("boom")
	})
	wg.Wait()
	// the panic is logged as the goroutine unwinds, just after wg.Done
	var text string
	for i := 0; i < 100; i++ {
		text = strings.Join(Tail(50), "\n")
		if strings.Contains(text, "boom") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(text, "background task crashed") || !strings.Contains(text, "boom") || !strings.Contains(text, "test task") {
		t.Errorf("log does not record the panic:\n%s", text)
	}
}

func TestTailOrderAndLimit(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	for i := 0; i < 20; i++ {
		slog.Info(fmt.Sprintf("line %02d", i))
	}
	tail := Tail(5)
	if len(tail) != 5 || !strings.Contains(tail[4], "line 19") || !strings.Contains(tail[0], "line 15") {
		t.Errorf("tail = %v", tail)
	}
	if text := Text("3.2.0", "Windows 11", 3); !strings.Contains(text, "SysInfo Pro 3.2.0") || !strings.Contains(text, "line 19") {
		t.Errorf("text = %s", text)
	}
}
