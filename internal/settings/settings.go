// Package settings stores the app's small settings files (JSON) in one folder,
// %APPDATA%\SysInfoPro, or the folder the user moved the data to (see appdata).
// Writes are atomic (a temporary file renamed over the old one), so a crash or power cut mid-write never leaves a half-written file, and a
// file that cannot be read is logged rather than silently replaced by defaults.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"sysinfopro/internal/appdata"
)

// SchemaVersion is written into every settings file. Files written before it
// existed read as version 0 and are loaded the same way: so far every change has
// only added fields, which older files simply lack.
const SchemaVersion = 1

const versionKey = "schema_version"

// dirOverride lets tests keep their files in a temporary folder
var dirOverride string

// Dir is the folder that holds every settings file
func Dir() (string, error) {
	if dirOverride != "" {
		return dirOverride, nil
	}
	return appdata.Dir(), nil
}

// Path is where the settings file called name lives
func Path(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

// LoadExisting reads name into a copy of defaults (fields missing from the file
// keep their default). found is false when there is no file.
func LoadExisting[T any](name string, defaults T) (value T, found bool, err error) {
	value = defaults
	path, err := Path(name)
	if err != nil {
		return value, false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return defaults, true, fmt.Errorf("%s is not valid: %w", name, err)
	}
	return value, true, nil
}

// Load reads name, falling back to defaults when there is no file. A file that
// exists but cannot be read is logged, and the defaults are used.
func Load[T any](name string, defaults T) T {
	value, _, err := LoadExisting(name, defaults)
	if err != nil {
		slog.Warn("settings file could not be read; using defaults", "file", name, "err", err)
	}
	return value
}

// Save writes value to name atomically, with the schema version alongside it
func Save[T any](name string, value T) error {
	path, err := Path(name)
	if err != nil {
		return err
	}
	data, err := withVersion(value)
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, data)
}

// Remove deletes the settings file called name, if there is one
func Remove(name string) {
	if path, err := Path(name); err == nil {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			slog.Warn("settings file could not be removed", "file", name, "err", err)
		}
	}
}

// withVersion marshals value as a JSON object with the schema version added
func withVersion[T any](value T) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		// not an object (a list, a number): store it as is
		return json.MarshalIndent(value, "", "  ")
	}
	fields[versionKey] = json.RawMessage(fmt.Sprint(SchemaVersion))
	return json.MarshalIndent(fields, "", "  ")
}

// WriteFileAtomic writes data to path through a temporary file in the same
// folder, then renames it over path, so readers see the old or the new file,
// never half of one
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	_, werr := tmp.Write(data)
	serr := tmp.Sync()
	cerr := tmp.Close()
	if err := errors.Join(werr, serr, cerr); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}
