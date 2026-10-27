package system

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PathEntry struct {
	Path        string `json:"path"`  // expanded, as programs see it
	Raw         string `json:"raw"`   // as stored, e.g. %USERPROFILE%\bin
	Scope       string `json:"scope"` // system | user | process
	Index       int    `json:"index"` // position in its scope's stored list
	IsValid     bool   `json:"is_valid"`
	IsDuplicate bool   `json:"is_duplicate"`
	StatusText  string `json:"status_text"`
	// a broken or duplicate entry in the user's own PATH, which can be removed
	// without administrator rights
	Fixable bool `json:"fixable"`
}

type EnvVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EnvPathAnalysis struct {
	TotalEntries    int           `json:"total_entries"`
	ValidEntries    int           `json:"valid_entries"`
	InvalidEntries  int           `json:"invalid_entries"`
	DuplicatesCount int           `json:"duplicates_count"`
	PathEntries     []PathEntry   `json:"path_entries"`
	KeyVariables    []EnvVariable `json:"key_variables"`
	// the user PATH exactly as stored; sent back with a removal so a PATH that
	// changed in the meantime is never overwritten
	UserPathRaw string `json:"user_path_raw"`
	// the last change made here can still be undone
	CanUndo bool `json:"can_undo"`
	// false when the stored lists could not be read and the app's own PATH was used
	Editable bool `json:"editable"`
}

// the stored PATH lists, as read from the registry (or faked in tests)
type pathSource struct {
	system, user string
	expand       func(string) string
	isDir        func(string) bool
}

func splitPath(raw string) []string {
	return strings.Split(raw, ";")
}

// analysePath walks the system PATH then the user PATH, the order Windows joins
// them in, and marks entries that point nowhere or repeat an earlier one
func analysePath(src pathSource) *EnvPathAnalysis {
	seen := map[string]bool{}
	analysis := &EnvPathAnalysis{PathEntries: []PathEntry{}, UserPathRaw: src.user, Editable: true}

	for _, scope := range []struct{ name, raw string }{{"system", src.system}, {"user", src.user}} {
		for i, raw := range splitPath(scope.raw) {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			expanded := src.expand(raw)
			key := strings.ToLower(strings.TrimRight(filepath.Clean(expanded), `\/`))

			entry := PathEntry{Path: expanded, Raw: raw, Scope: scope.name, Index: i}
			entry.IsDuplicate = seen[key]
			seen[key] = true
			entry.IsValid = src.isDir(expanded)
			// a %VARIABLE% nobody defines cannot be checked: it is not known to be
			// missing, and must never be offered for removal
			unknown := !entry.IsValid && strings.Contains(expanded, "%")

			switch {
			case entry.IsDuplicate:
				entry.StatusText = "Duplicate"
				analysis.DuplicatesCount++
			case unknown:
				entry.StatusText = "Uses an undefined variable"
			case !entry.IsValid:
				entry.StatusText = "Missing folder"
			default:
				entry.StatusText = "Valid"
			}
			switch {
			case entry.IsValid:
				analysis.ValidEntries++
			case !unknown:
				analysis.InvalidEntries++
			}
			entry.Fixable = scope.name == "user" && (entry.IsDuplicate || (!entry.IsValid && !unknown))
			analysis.PathEntries = append(analysis.PathEntries, entry)
		}
	}
	analysis.TotalEntries = len(analysis.PathEntries)
	return analysis
}

// removeEntries drops the entries at the given positions of a stored PATH list
func removeEntries(raw string, indices []int) string {
	drop := map[int]bool{}
	for _, i := range indices {
		drop[i] = true
	}
	var kept []string
	for i, entry := range splitPath(raw) {
		if !drop[i] && strings.TrimSpace(entry) != "" {
			kept = append(kept, entry)
		}
	}
	return strings.Join(kept, ";")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func keyVariables() []EnvVariable {
	vars := []EnvVariable{}
	for _, key := range []string{"JAVA_HOME", "PYTHONPATH", "GOROOT", "GOPATH", "CUDA_PATH", "NODE_ENV", "TEMP", "TMP"} {
		if val := os.Getenv(key); val != "" {
			if len(val) > 60 {
				val = val[:57] + "..."
			}
			vars = append(vars, EnvVariable{Key: key, Value: val})
		}
	}
	return vars
}

// GetEnvPathAnalysis checks every entry of the system and user PATH
func GetEnvPathAnalysis() (*EnvPathAnalysis, error) {
	system, user, err := readStoredPath()
	var analysis *EnvPathAnalysis
	if err != nil {
		// fall back to the PATH this app was started with; it cannot be edited
		analysis = analysePath(pathSource{system: os.Getenv("PATH"), expand: os.ExpandEnv, isDir: isDir})
		analysis.Editable = false
		for i := range analysis.PathEntries {
			analysis.PathEntries[i].Scope = "process"
			analysis.PathEntries[i].Fixable = false
		}
	} else {
		analysis = analysePath(pathSource{system: system, user: user, expand: expandEnv, isDir: isDir})
		if b, err := loadBackup(); err == nil && b.After == user {
			analysis.CanUndo = true
		}
	}
	analysis.KeyVariables = keyVariables()
	return analysis, nil
}

// ErrPathChanged means the user PATH changed after it was shown
var ErrPathChanged = errors.New("your PATH changed since it was checked; refresh and try again")

// RemoveUserPathEntries removes entries from the user's own PATH. original must
// be the stored value the entries were chosen from (UserPathRaw).
func RemoveUserPathEntries(original string, indices []int) (*EnvPathAnalysis, error) {
	_, current, err := readStoredPath()
	if err != nil {
		return nil, err
	}
	if current != original {
		return nil, ErrPathChanged
	}
	updated := removeEntries(current, indices)
	if updated == current {
		return GetEnvPathAnalysis()
	}
	if err := saveBackup(pathBackup{Before: current, After: updated, At: time.Now().Unix()}); err != nil {
		return nil, err
	}
	if err := writeUserPath(updated); err != nil {
		return nil, err
	}
	return GetEnvPathAnalysis()
}

// UndoPathChange puts back the user PATH as it was before the last removal,
// provided nothing else has changed it since
func UndoPathChange() (*EnvPathAnalysis, error) {
	b, err := loadBackup()
	if err != nil {
		return nil, errors.New("there is no change to undo")
	}
	_, current, err := readStoredPath()
	if err != nil {
		return nil, err
	}
	if current != b.After {
		return nil, errors.New("your PATH changed again after the last fix, so it was not restored")
	}
	if err := writeUserPath(b.Before); err != nil {
		return nil, err
	}
	removeBackup()
	return GetEnvPathAnalysis()
}

// --- backup of the last change, for undo ---

type pathBackup struct {
	Before string `json:"before"`
	After  string `json:"after"`
	At     int64  `json:"at"`
}

func backupPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SysInfoPro", "path-backup.json"), nil
}

func saveBackup(b pathBackup) error {
	path, err := backupPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(b, "", "  ")
	return os.WriteFile(path, data, 0o644)
}

func loadBackup() (pathBackup, error) {
	var b pathBackup
	path, err := backupPath()
	if err != nil {
		return b, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return b, err
	}
	return b, json.Unmarshal(data, &b)
}

func removeBackup() {
	if path, err := backupPath(); err == nil {
		os.Remove(path)
	}
}
