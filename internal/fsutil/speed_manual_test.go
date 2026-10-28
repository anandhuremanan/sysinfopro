//go:build manual

package fsutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// go test -tags manual -run TestRemoveSpeed ./internal/fsutil/ -v
// A Go-module-cache-like tree: many small read-only files in nested folders.
func TestRemoveSpeed(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mod")
	const dirs, perDir = 400, 100
	for d := 0; d < dirs; d++ {
		dir := filepath.Join(root, fmt.Sprintf("example.com/m%03d@v1.0.0/pkg", d))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for f := 0; f < perDir; f++ {
			p := filepath.Join(dir, fmt.Sprintf("f%03d.go", f))
			if err := os.WriteFile(p, make([]byte, 2048), 0o444); err != nil {
				t.Fatal(err)
			}
		}
	}
	start := time.Now()
	files := 0
	_, err := RemoveTree(context.Background(), root, func(int64) { files++ })
	t.Logf("removed %d read-only files in %v (err %v)", files, time.Since(start).Round(time.Millisecond), err)
	if _, statErr := os.Stat(root); statErr == nil {
		t.Fatal("root still exists")
	}
}
