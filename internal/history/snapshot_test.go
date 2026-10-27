package history

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func category(t *testing.T, changes *Changes, id string) CategoryDiff {
	t.Helper()
	for _, c := range changes.Categories {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("category %q missing", id)
	return CategoryDiff{}
}

func TestCompare(t *testing.T) {
	baseline := Snapshot{
		TakenAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
		Startup: []string{"OneDrive", "Teams"},
		Ports:   []string{"135 svchost.exe", "5432 postgres.exe"},
		Path:    []string{`C:\Windows`, `C:\old-sdk\bin`},
		Apps:    map[string]string{"Node.js": "20.1.0", "Git": "2.44", "OldTool": "1.0"},
	}
	current := Snapshot{
		TakenAt: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC),
		Startup: []string{"OneDrive", "Updater", "Updater"},
		Ports:   []string{"135 svchost.exe", "3000 node.exe"},
		Path:    []string{`C:\Windows`, `C:\new-sdk\bin`},
		Apps:    map[string]string{"Node.js": "22.3.0", "Git": "2.44", "Docker Desktop": "4.30"},
	}

	changes := Compare(baseline, current)

	if !changes.HasBaseline || changes.Since != "2026-10-01T09:00:00Z" {
		t.Errorf("baseline info = %v %q", changes.HasBaseline, changes.Since)
	}

	startup := category(t, changes, "startup")
	if !reflect.DeepEqual(startup.Added, []string{"Updater"}) || !reflect.DeepEqual(startup.Removed, []string{"Teams"}) {
		t.Errorf("startup = +%v -%v, want +[Updater] once and -[Teams]", startup.Added, startup.Removed)
	}

	apps := category(t, changes, "apps")
	if !reflect.DeepEqual(apps.Added, []string{"Docker Desktop 4.30"}) ||
		!reflect.DeepEqual(apps.Removed, []string{"OldTool 1.0"}) ||
		!reflect.DeepEqual(apps.Changed, []string{"Node.js: 20.1.0 → 22.3.0"}) {
		t.Errorf("apps = +%v -%v ~%v", apps.Added, apps.Removed, apps.Changed)
	}

	ports := category(t, changes, "ports")
	if !reflect.DeepEqual(ports.Added, []string{"3000 node.exe"}) || !reflect.DeepEqual(ports.Removed, []string{"5432 postgres.exe"}) {
		t.Errorf("ports = +%v -%v", ports.Added, ports.Removed)
	}

	// 2 startup + 3 apps + 2 ports + 2 path
	if changes.Total != 9 {
		t.Errorf("total = %d, want 9", changes.Total)
	}
}

func TestCompareIdenticalIsEmpty(t *testing.T) {
	snapshot := Snapshot{Startup: []string{"A"}, Ports: []string{"80 x"}, Path: []string{"p"}, Apps: map[string]string{"App": "1"}}
	changes := Compare(snapshot, snapshot)
	if changes.Total != 0 {
		t.Errorf("identical snapshots reported %d changes", changes.Total)
	}
	for _, c := range changes.Categories {
		// the UI iterates these, so they must be empty lists rather than null
		if c.Added == nil || c.Removed == nil || c.Changed == nil {
			t.Errorf("%s has a nil list", c.ID)
		}
	}
}

func TestStoreLatestAndPruning(t *testing.T) {
	store := &Store{Dir: filepath.Join(t.TempDir(), "snapshots")}

	if latest, err := store.Latest(); err != nil || latest != nil {
		t.Fatalf("empty store returned %+v, %v", latest, err)
	}

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < snapshotsKept+5; i++ {
		snapshot := Snapshot{TakenAt: start.Add(time.Duration(i) * time.Hour), Startup: []string{"run"}}
		if err := store.Save(snapshot); err != nil {
			t.Fatal(err)
		}
	}

	if got := len(store.files()); got != snapshotsKept {
		t.Errorf("%d snapshot files kept, want %d", got, snapshotsKept)
	}
	latest, err := store.Latest()
	if err != nil || latest == nil {
		t.Fatalf("latest = %+v, %v", latest, err)
	}
	if want := start.Add(time.Duration(snapshotsKept+4) * time.Hour); !latest.TakenAt.Equal(want) {
		t.Errorf("latest taken at %v, want the newest %v", latest.TakenAt, want)
	}
}

func TestStoreSkipsCorruptFile(t *testing.T) {
	store := &Store{Dir: t.TempDir()}
	good := Snapshot{TakenAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Path: []string{"ok"}}
	if err := store.Save(good); err != nil {
		t.Fatal(err)
	}
	// a newer, half-written file must not hide the good snapshot
	newer := filepath.Join(store.Dir, "snapshot-99999999999999999999.json")
	if err := os.WriteFile(newer, []byte("{ truncated"), 0o644); err != nil {
		t.Fatal(err)
	}

	latest, err := store.Latest()
	if err != nil || latest == nil || len(latest.Path) != 1 {
		t.Errorf("latest = %+v, %v; want the readable snapshot", latest, err)
	}
}

func TestUnreadableListsAreNotCompared(t *testing.T) {
	baseline := Snapshot{
		Startup: []string{"OneDrive (Registry)"},
		Path:    []string{`C:\Tools\`},
		Apps:    map[string]string{"Git": "2.45"},
	}
	// startup could not be read now (nil), and no apps were read at all
	current := Snapshot{Path: []string{`c:\tools`}, Apps: map[string]string{}}
	c := Compare(baseline, current)
	if c.Total != 0 {
		t.Errorf("unreadable lists and case or slash differences are not changes: %+v", c.Categories)
	}
}
