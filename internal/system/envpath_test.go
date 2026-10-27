package system

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func fakeSource(system, user string, dirs ...string) pathSource {
	exists := map[string]bool{}
	for _, d := range dirs {
		exists[strings.ToLower(d)] = true
	}
	return pathSource{
		system: system,
		user:   user,
		expand: func(s string) string { return strings.ReplaceAll(s, "%USERPROFILE%", `C:\Users\dev`) },
		// like os.Stat, a trailing backslash does not matter
		isDir: func(p string) bool { return exists[strings.ToLower(strings.TrimRight(p, `\`))] },
	}
}

func TestAnalysePath(t *testing.T) {
	src := fakeSource(
		`C:\Windows;C:\Windows\System32;C:\Old SDK\bin;;C:\Go\bin`,
		`%USERPROFILE%\go\bin;C:\Go\bin\;C:\Users\dev\gone;c:\windows`,
		`C:\Windows`, `C:\Windows\System32`, `C:\Go\bin`, `C:\Users\dev\go\bin`,
	)
	a := analysePath(src)

	if a.TotalEntries != 8 || a.InvalidEntries != 2 || a.DuplicatesCount != 2 {
		t.Fatalf("total %d, invalid %d, duplicates %d; want 8, 2, 2", a.TotalEntries, a.InvalidEntries, a.DuplicatesCount)
	}
	byRaw := map[string]PathEntry{}
	for _, e := range a.PathEntries {
		byRaw[e.Scope+" "+e.Raw] = e
	}

	if e := byRaw[`system C:\Old SDK\bin`]; e.IsValid || e.Fixable {
		t.Errorf("a broken system entry needs admin rights, so it is not fixable here: %+v", e)
	}
	if e := byRaw[`user %USERPROFILE%\go\bin`]; !e.IsValid || e.Fixable || e.Path != `C:\Users\dev\go\bin` {
		t.Errorf("a valid user entry is expanded and left alone: %+v", e)
	}
	// a trailing slash or different case still repeats the system entry
	for _, raw := range []string{`C:\Go\bin\`, `c:\windows`} {
		if e := byRaw["user "+raw]; !e.IsDuplicate || !e.Fixable {
			t.Errorf("%s repeats a system entry and can be removed: %+v", raw, e)
		}
	}
	if e := byRaw[`user C:\Users\dev\gone`]; e.IsValid || !e.Fixable || e.Index != 2 {
		t.Errorf("a missing user folder can be removed: %+v", e)
	}
}

func TestRemoveEntries(t *testing.T) {
	got := removeEntries(`A;B;;C;D`, []int{1, 4})
	if got != `A;C` {
		t.Errorf("removeEntries = %q, want A;C (empty entries dropped too)", got)
	}
}

// RemoveUserPathEntries and UndoPathChange against a scratch registry key, never
// the real PATH
func TestRemoveAndUndo(t *testing.T) {
	const testKey = `Software\SysInfoProTest\Environment`
	key, _, err := registry.CreateKey(registry.CURRENT_USER, testKey, registry.ALL_ACCESS)
	if err != nil {
		t.Skipf("cannot create a scratch registry key: %v", err)
	}
	key.Close()
	t.Cleanup(func() {
		registry.DeleteKey(registry.CURRENT_USER, testKey)
		registry.DeleteKey(registry.CURRENT_USER, `Software\SysInfoProTest`)
	})
	saved := pathKey
	pathKey.path = testKey
	t.Cleanup(func() { pathKey = saved })
	t.Setenv("AppData", t.TempDir()) // the undo backup goes here

	dir := t.TempDir()
	original := dir + `;C:\definitely\missing;` + dir
	if err := writeUserPath(original); err != nil {
		t.Fatal(err)
	}

	before, _ := GetEnvPathAnalysis()
	if before.UserPathRaw != original || before.CanUndo {
		t.Fatalf("before: raw %q, can undo %v", before.UserPathRaw, before.CanUndo)
	}

	if _, err := RemoveUserPathEntries(dir+";something else", []int{1}); err != ErrPathChanged {
		t.Errorf("a stale original must be refused, got %v", err)
	}

	after, err := RemoveUserPathEntries(original, []int{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if after.UserPathRaw != dir || !after.CanUndo {
		t.Errorf("after removal: raw %q, can undo %v; want %q and true", after.UserPathRaw, after.CanUndo, dir)
	}
	if _, kind, _ := readKind(t, testKey); kind != registry.EXPAND_SZ {
		t.Errorf("PATH must stay REG_EXPAND_SZ, got type %d", kind)
	}

	undone, err := UndoPathChange()
	if err != nil {
		t.Fatal(err)
	}
	if undone.UserPathRaw != original || undone.CanUndo {
		t.Errorf("after undo: raw %q, can undo %v", undone.UserPathRaw, undone.CanUndo)
	}
}

func readKind(t *testing.T, path string) (string, uint32, error) {
	t.Helper()
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		return "", 0, err
	}
	defer key.Close()
	return key.GetStringValue("Path")
}

func TestUndefinedVariableIsNotMissing(t *testing.T) {
	src := fakeSource(``, `%NOT_DEFINED_ANYWHERE%\\bin;C:\\gone`)
	a := analysePath(src)
	if len(a.PathEntries) != 2 {
		t.Fatalf("entries = %+v", a.PathEntries)
	}
	undefined, gone := a.PathEntries[0], a.PathEntries[1]
	if undefined.Fixable || undefined.StatusText == "Missing folder" {
		t.Errorf("an unexpandable entry must not be called missing or offered for removal: %+v", undefined)
	}
	if !gone.Fixable || gone.StatusText != "Missing folder" {
		t.Errorf("a real missing folder is still reported: %+v", gone)
	}
	if a.InvalidEntries != 1 {
		t.Errorf("invalid = %d, want only the real missing folder", a.InvalidEntries)
	}
}

func TestOnlyFixableIndicesAreRemoved(t *testing.T) {
	a := &EnvPathAnalysis{PathEntries: []PathEntry{
		{Scope: "user", Index: 0, Fixable: false},  // a working folder
		{Scope: "user", Index: 1, Fixable: true},   // missing
		{Scope: "system", Index: 2, Fixable: true}, // never the system list
		{Scope: "user", Index: 3, Fixable: true},   // duplicate
	}}
	got := onlyFixable(a, []int{0, 1, 2, 3, 7})
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Errorf("got %v, want [1 3]", got)
	}
}
