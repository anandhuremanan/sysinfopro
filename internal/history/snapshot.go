// Package history records what is installed and configured on the machine,
// so a later run can show what changed.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const snapshotsKept = 20

// Snapshot is the state of the machine at one moment
type Snapshot struct {
	TakenAt time.Time `json:"taken_at"`
	Startup []string  `json:"startup"`
	Ports   []string  `json:"ports"`
	Path    []string  `json:"path"`
	// installed application name -> version
	Apps map[string]string `json:"apps"`
}

type CategoryDiff struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Changed []string `json:"changed"`
}

type Changes struct {
	// false on the first run, when there was nothing to compare with
	HasBaseline bool           `json:"has_baseline"`
	Since       string         `json:"since"` // RFC 3339 time of the baseline
	Total       int            `json:"total"`
	Categories  []CategoryDiff `json:"categories"`
}

// Compare describes how current differs from baseline
func Compare(baseline, current Snapshot) *Changes {
	apps := CategoryDiff{ID: "apps", Title: "Installed software", Added: []string{}, Removed: []string{}, Changed: []string{}}
	for name, version := range current.Apps {
		old, existed := baseline.Apps[name]
		switch {
		case !existed:
			apps.Added = append(apps.Added, strings.TrimSpace(name+" "+version))
		case old != version:
			apps.Changed = append(apps.Changed, fmt.Sprintf("%s: %s → %s", name, old, version))
		}
	}
	for name, version := range baseline.Apps {
		if _, still := current.Apps[name]; !still {
			apps.Removed = append(apps.Removed, strings.TrimSpace(name+" "+version))
		}
	}
	sort.Strings(apps.Added)
	sort.Strings(apps.Removed)
	sort.Strings(apps.Changed)

	changes := &Changes{
		HasBaseline: true,
		Since:       baseline.TakenAt.Format(time.RFC3339),
		Categories: []CategoryDiff{
			listDiff("startup", "Startup programs", baseline.Startup, current.Startup),
			apps,
			listDiff("ports", "Listening ports", baseline.Ports, current.Ports),
			listDiff("path", "PATH entries", baseline.Path, current.Path),
		},
	}
	for _, c := range changes.Categories {
		changes.Total += len(c.Added) + len(c.Removed) + len(c.Changed)
	}
	return changes
}

func listDiff(id, title string, before, after []string) CategoryDiff {
	diff := CategoryDiff{ID: id, Title: title, Added: []string{}, Removed: []string{}, Changed: []string{}}
	had := map[string]bool{}
	for _, item := range before {
		had[item] = true
	}
	has := map[string]bool{}
	for _, item := range after {
		if !has[item] && !had[item] {
			diff.Added = append(diff.Added, item)
		}
		has[item] = true
	}
	for _, item := range before {
		if !has[item] {
			diff.Removed = append(diff.Removed, item)
			has[item] = true // list each removal once
		}
	}
	return diff
}

// Store keeps snapshots as JSON files in one folder
type Store struct {
	Dir string
}

// DefaultStore keeps snapshots in the user's app-settings folder
func DefaultStore() (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &Store{Dir: filepath.Join(base, "SysInfoPro", "snapshots")}, nil
}

func (s *Store) files() []string {
	matches, _ := filepath.Glob(filepath.Join(s.Dir, "snapshot-*.json"))
	sort.Strings(matches) // names carry a zero-padded timestamp, so this is oldest first
	return matches
}

// Latest returns the most recent snapshot, or nil when none has been saved
func (s *Store) Latest() (*Snapshot, error) {
	files := s.files()
	for i := len(files) - 1; i >= 0; i-- {
		data, err := os.ReadFile(files[i])
		if err != nil {
			continue
		}
		snapshot := &Snapshot{}
		if json.Unmarshal(data, snapshot) == nil {
			return snapshot, nil
		}
		// an unreadable file is skipped so one bad write cannot hide older snapshots
	}
	return nil, nil
}

// Save writes snapshot and drops the oldest ones beyond the keep limit
func (s *Store) Save(snapshot Snapshot) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	name := fmt.Sprintf("snapshot-%020d.json", snapshot.TakenAt.UnixNano())
	if err := os.WriteFile(filepath.Join(s.Dir, name), data, 0o644); err != nil {
		return err
	}

	files := s.files()
	for len(files) > snapshotsKept {
		os.Remove(files[0])
		files = files[1:]
	}
	return nil
}
