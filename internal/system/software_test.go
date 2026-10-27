package system

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.22.10", "1.22.9", 1},
		{"9.4.0", "10.0.0", -1},
		{"2.45.1", "2.45.1", 0},
		{"v20.11.0", "20.11", 0},
		{"3.13.7", "3.13.10", -1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
	if isVersionOlder("1.22.10", "1.22.9") || !isVersionOlder("9.4.0", "10.0.0") || isVersionOlder("1.0.0", "") {
		t.Error("isVersionOlder")
	}
}

func TestPickSameLine(t *testing.T) {
	node := []byte(`[{"version":"v25.1.0","date":"2026-10-01"},{"version":"v24.11.0","date":"2026-09-20"},{"version":"v22.21.0","date":"2026-09-10"},{"version":"v22.20.0","date":"2026-08-01"}]`)
	if v, d := pickNode(node, "22.14.0"); v != "22.21.0" || d != "2026-09-10" {
		t.Errorf("node = %s %s, want the newest 22.x", v, d)
	}
	py := []byte(`[{"cycle":"3.14","latest":"3.14.0","latestReleaseDate":"2026-10-07"},{"cycle":"3.13","latest":"3.13.9","latestReleaseDate":"2026-10-14"}]`)
	if v, _ := pickPython(py, "3.13.7"); v != "3.13.9" {
		t.Errorf("python = %s, want 3.13.9", v)
	}
	if v, _ := pickPython(py, "3.8.10"); v != "3.14.0" {
		t.Errorf("an old line is compared with the newest: %s", v)
	}
	gol := []byte(`[{"version":"go1.27.3"},{"version":"go1.26.8"}]`)
	if v, d := pickGo(gol, "1.26.2"); v != "1.26.8" || d != "" {
		t.Errorf("go = %s %q", v, d)
	}
	if v, d := pickNode([]byte(`not json`), "22.1.0"); v != "" || d != "" {
		t.Errorf("a bad reply must be unknown, not a made-up version: %s %s", v, d)
	}
}

func TestSameProduct(t *testing.T) {
	yes := [][2]string{{"go programming language amd64 go1.27.3", "Go"}, {"git", "Git"}, {"python 3.13.7 (64-bit)", "Python"}, {"microsoft visual studio code (user)", "Visual Studio Code"}}
	no := [][2]string{{"google chrome", "Go"}, {"github desktop", "Git"}, {"mongodb compass", "Go"}}
	for _, c := range yes {
		if !sameProduct(c[0], c[1]) {
			t.Errorf("%q should be %s", c[0], c[1])
		}
	}
	for _, c := range no {
		if sameProduct(c[0], c[1]) {
			t.Errorf("%q is not %s", c[0], c[1])
		}
	}
}
