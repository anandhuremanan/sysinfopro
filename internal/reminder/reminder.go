// Package reminder decides when the monthly cleanup reminder is due and what it says.
package reminder

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// how often the reminder may appear
	Interval = 30 * 24 * time.Hour
	// below this, there is nothing worth mentioning
	MinWorthBytes = 1 << 30
	// project leftovers count once their project has been untouched this long
	StaleAfter = 182 * 24 * time.Hour
)

type Settings struct {
	Enabled bool `json:"enabled"`
	// the folder last searched in Dev caches, checked for old project leftovers
	ProjectsRoot string `json:"projects_root"`
	// unix seconds of the last check; 0 means never
	LastCheck int64 `json:"last_check"`
}

// Due reports whether a check should run now
func Due(s Settings, now time.Time) bool {
	return s.Enabled && now.Sub(time.Unix(s.LastCheck, 0)) >= Interval
}

// Findings are what a check measured
type Findings struct {
	CacheBytes    int64
	JunkBytes     int64
	LeftoverBytes int64
	LeftoverCount int
}

func (f Findings) Total() int64 { return f.CacheBytes + f.JunkBytes + f.LeftoverBytes }

// Message is the reminder's title and text, or ok=false when there is too little to mention
func Message(f Findings) (title, body string, ok bool) {
	if f.Total() < MinWorthBytes {
		return "", "", false
	}
	var parts []string
	if f.LeftoverBytes > 0 {
		noun := "folders"
		if f.LeftoverCount == 1 {
			noun = "folder"
		}
		parts = append(parts, fmt.Sprintf("%s in %d old project %s", formatGB(f.LeftoverBytes), f.LeftoverCount, noun))
	}
	if f.CacheBytes > 0 {
		parts = append(parts, fmt.Sprintf("%s of package caches", formatGB(f.CacheBytes)))
	}
	if f.JunkBytes > 0 {
		parts = append(parts, fmt.Sprintf("%s of temporary files", formatGB(f.JunkBytes)))
	}
	return fmt.Sprintf("You could free %s", formatGB(f.Total())), capitalise(joinList(parts)) + ".", true
}

func formatGB(bytes int64) string {
	v := float64(bytes) / (1 << 30)
	if v < 10 {
		return fmt.Sprintf("%.1f GB", v)
	}
	// halves round up, as people expect (14.5 GB reads as 15 GB)
	return fmt.Sprintf("%.0f GB", math.Round(v))
}

func joinList(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// --- settings file ---

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SysInfoPro", "reminder.json"), nil
}

func LoadSettings() Settings {
	var s Settings
	if path, err := settingsPath(); err == nil {
		if data, err := os.ReadFile(path); err == nil {
			json.Unmarshal(data, &s)
		}
	}
	return s
}

func SaveSettings(s Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(path, data, 0o644)
}
