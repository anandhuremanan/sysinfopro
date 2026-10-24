package reminder

import (
	"testing"
	"time"
)

const gb = 1 << 30

func TestDue(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		s    Settings
		want bool
	}{
		{Settings{Enabled: false}, false},
		{Settings{Enabled: true}, true}, // never checked
		{Settings{Enabled: true, LastCheck: now.Add(-29 * 24 * time.Hour).Unix()}, false},
		{Settings{Enabled: true, LastCheck: now.Add(-31 * 24 * time.Hour).Unix()}, true},
	}
	for _, c := range cases {
		if got := Due(c.s, now); got != c.want {
			t.Errorf("Due(%+v) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestMessage(t *testing.T) {
	title, body, ok := Message(Findings{CacheBytes: 5 * gb, JunkBytes: gb / 2, LeftoverBytes: 9 * gb, LeftoverCount: 4})
	if !ok || title != "You could free 15 GB" ||
		body != "9.0 GB in 4 old project folders, 5.0 GB of package caches and 0.5 GB of temporary files." {
		t.Errorf("got %q / %q / %v", title, body, ok)
	}

	_, body, _ = Message(Findings{LeftoverBytes: 2 * gb, LeftoverCount: 1})
	if body != "2.0 GB in 1 old project folder." {
		t.Errorf("single folder: %q", body)
	}

	if _, _, ok := Message(Findings{CacheBytes: gb / 2}); ok {
		t.Error("half a gigabyte is not worth a reminder")
	}
}
