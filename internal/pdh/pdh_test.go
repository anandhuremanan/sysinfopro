package pdh

import "testing"

// a nil query (counters unavailable) reads as unknown and never panics
func TestNilQuery(t *testing.T) {
	var q *Query
	q.Collect()
	if q.Value(`\System\Processes`) != -1 || q.Has(`\System\Processes`) {
		t.Error("nil query should read as unknown")
	}
	q.Close()
}
