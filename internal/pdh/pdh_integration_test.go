//go:build integration && windows

package pdh

import (
	"testing"
	"time"
)

func TestReadsRealCounters(t *testing.T) {
	q := Open(`\System\Processes`, `\Processor Information(_Total)\% Processor Utility`, `\No Such\Counter`)
	if q == nil {
		t.Skip("performance counters unavailable")
	}
	defer q.Close()
	time.Sleep(500 * time.Millisecond)
	q.Collect()
	if v := q.Value(`\System\Processes`); v <= 0 {
		t.Errorf("processes = %v", v)
	}
	if q.Has(`\No Such\Counter`) || q.Value(`\No Such\Counter`) != -1 {
		t.Error("an unknown counter should be left out")
	}
}
