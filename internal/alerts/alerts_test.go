package alerts

import (
	"strings"
	"testing"
	"time"
)

const total = 500 * gb

var start = time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

// gbs is a size in gigabytes, for readable test data
func gbs(v float64) uint64 { return uint64(v * gb) }

func at(minutes float64) time.Time {
	return start.Add(time.Duration(minutes * float64(time.Minute)))
}

func kinds(alerts []Alert) string {
	var k []string
	for _, a := range alerts {
		k = append(k, a.Kind)
	}
	return strings.Join(k, ",")
}

func TestLowSpaceAlertsOncePerCrossing(t *testing.T) {
	m := NewMonitor(Settings{Enabled: true, MinFreeGB: 5})

	if got := m.Observe("C:", 6*gb, total, at(0)); len(got) != 0 {
		t.Fatalf("6 GB free with a 5 GB limit: got %v", got)
	}
	got := m.Observe("C:", 4*gb, total, at(1))
	if kinds(got) != "low" || got[0].Title != "C: is almost full" || !strings.Contains(got[0].Body, "4.0 GB free") {
		t.Fatalf("crossing below the limit: got %+v", got)
	}
	if got := m.Observe("C:", 3*gb, total, at(2)); len(got) != 0 {
		t.Errorf("still low a minute later should stay quiet, got %v", got)
	}
	// hovering just above the limit is not a recovery
	m.Observe("C:", gbs(5.2), total, at(3))
	if got := m.Observe("C:", gbs(4.9), total, at(4)); len(got) != 0 {
		t.Errorf("dipping again without a real recovery should stay quiet, got %v", got)
	}
	// a real recovery re-arms the alert
	m.Observe("C:", 8*gb, total, at(5))
	if got := m.Observe("C:", 4*gb, total, at(6)); kinds(got) != "low" {
		t.Errorf("after recovering, falling below again should alert, got %v", got)
	}
}

func TestLowSpaceRepeatsAfterHours(t *testing.T) {
	m := NewMonitor(Settings{Enabled: true, MinFreeGB: 5})
	m.Observe("C:", 3*gb, total, at(0))
	if got := m.Observe("C:", 3*gb, total, at(5*60)); len(got) != 0 {
		t.Errorf("five hours later: got %v", got)
	}
	if got := m.Observe("C:", 3*gb, total, at(6*60+1)); kinds(got) != "low" {
		t.Errorf("six hours later the reminder is due, got %v", got)
	}
}

func TestFastDrop(t *testing.T) {
	m := NewMonitor(Settings{Enabled: true, MinFreeGB: 1, DropGB: 2})
	m.Observe("D:", 50*gb, total, at(0))
	m.Observe("D:", 49*gb, total, at(3))
	got := m.Observe("D:", gbs(47.5), total, at(6))
	if kinds(got) != "dropping" || got[0].Title != "D: is filling up fast" ||
		!strings.Contains(got[0].Body, "2.5 GB used in the last 6 minutes") {
		t.Fatalf("2.5 GB lost in 6 minutes: got %+v", got)
	}
	if got := m.Observe("D:", 44*gb, total, at(8)); len(got) != 0 {
		t.Errorf("within the cool-down: got %v", got)
	}
}

func TestSlowDropIsNotAlerted(t *testing.T) {
	m := NewMonitor(Settings{Enabled: true, MinFreeGB: 1, DropGB: 2})
	// 3 GB over half an hour is never 2 GB inside one ten-minute window
	for i := 0; i <= 30; i++ {
		if got := m.Observe("D:", uint64((50-0.1*float64(i))*gb), total, at(float64(i))); len(got) != 0 {
			t.Fatalf("minute %d: got %v", i, got)
		}
	}
}

func TestDisabledAndOddLimits(t *testing.T) {
	m := NewMonitor(Settings{Enabled: false, MinFreeGB: 5, DropGB: 2})
	if got := m.Observe("C:", 1*gb, total, at(0)); len(got) != 0 {
		t.Errorf("alerts off: got %v", got)
	}
	// a 5 GB limit on a 4 GB USB stick would always be low; ignore it there
	m.SetSettings(Settings{Enabled: true, MinFreeGB: 5})
	if got := m.Observe("E:", 1*gb, 4*gb, at(1)); len(got) != 0 {
		t.Errorf("limit larger than the drive: got %v", got)
	}
}
