//go:build manual

package relocate

import (
	"context"
	"testing"
	"time"
)

// go test -tags manual -run TestDetectHere -v ./internal/relocate/   (read-only)
func TestDetectHere(t *testing.T) {
	start := time.Now()
	for _, it := range Detect(context.Background()) {
		t.Logf("%-8s %-18s %8.1f MB  sys=%-5v moved=%-5v %s | blocker=%q guide=%v", it.Group, it.Name, float64(it.SizeBytes)/1e6, it.OnSystemDrive, it.Moved, it.Path, it.Blocker, it.Guide != "")
	}
	t.Logf("took %v", time.Since(start).Round(time.Millisecond))
}
