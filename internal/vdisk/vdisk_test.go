package vdisk

import "testing"

func TestParseDockerDF(t *testing.T) {
	out := []byte(`{"Active":"2","Reclaimable":"3.1GB (62%)","Size":"5.02GB","TotalCount":"14","Type":"Images"}
{"Active":"1","Reclaimable":"120MB (98%)","Size":"122MB","TotalCount":"3","Type":"Containers"}
not json
{"Active":"0","Reclaimable":"812MB","Size":"812MB","TotalCount":"41","Type":"Build Cache"}
`)
	rows := parseDockerDF(out)
	if len(rows) != 3 {
		t.Fatalf("rows = %+v, want 3", rows)
	}
	if rows[0] != (DockerRow{"Images", "14", "5.02GB", "3.1GB (62%)"}) {
		t.Errorf("first row = %+v", rows[0])
	}
	if rows[2].Type != "Build Cache" || rows[2].Reclaimable != "812MB" {
		t.Errorf("last row = %+v", rows[2])
	}
}

func TestReclaimedPattern(t *testing.T) {
	out := "Deleted build cache objects:\nabc\n\nTotal reclaimed space: 1.284GB\n"
	if m := reclaimedPattern.FindStringSubmatch(out); m == nil || m[1] != "1.284GB" {
		t.Errorf("match = %v", m)
	}
}

func TestDockerNamesAreDocker(t *testing.T) {
	for name, want := range map[string]bool{"docker-desktop": true, "docker-desktop-data": true, "Ubuntu-22.04": false} {
		if isDocker(name) != want {
			t.Errorf("isDocker(%q) = %v", name, !want)
		}
	}
}
