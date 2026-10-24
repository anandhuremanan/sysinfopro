// Package alerts watches free space on each drive and raises an alert when a
// drive runs low or starts filling up fast.
package alerts

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const gb = 1 << 30

// Settings are what the user chooses in Settings > Alerts
type Settings struct {
	Enabled bool `json:"enabled"`
	// alert when a drive has less than this much free
	MinFreeGB float64 `json:"min_free_gb"`
	// alert when a drive loses this much within DropWindow; 0 turns it off
	DropGB float64 `json:"drop_gb"`
}

func DefaultSettings() Settings {
	return Settings{Enabled: true, MinFreeGB: 5, DropGB: 2}
}

// An Alert is one notification's worth of news about a drive
type Alert struct {
	Drive     string `json:"drive"`
	Kind      string `json:"kind"` // low | dropping
	Title     string `json:"title"`
	Body      string `json:"body"`
	FreeBytes uint64 `json:"free_bytes"`
	At        int64  `json:"at"` // unix milliseconds
}

const (
	// how far back a fast drop is measured
	DropWindow = 10 * time.Minute
	// a drive that stays low is mentioned again after this long
	lowRepeat = 6 * time.Hour
	// after a fast-drop alert, the same drive stays quiet this long
	dropCooldown = 30 * time.Minute
	// a low drive must climb this far above the limit to count as recovered
	recoverMargin = 1.1
)

type sample struct {
	at   time.Time
	free uint64
}

type driveState struct {
	samples   []sample
	lowSince  time.Time // zero while the drive is above the limit
	lowAlert  time.Time // last low alert
	dropAlert time.Time // last fast-drop alert
}

// Monitor turns a stream of free-space readings into alerts. It holds no
// timers of its own; the caller feeds it readings, which keeps it easy to test.
type Monitor struct {
	mu       sync.Mutex
	settings Settings
	drives   map[string]*driveState
}

func NewMonitor(settings Settings) *Monitor {
	return &Monitor{settings: settings, drives: map[string]*driveState{}}
}

func (m *Monitor) Settings() Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settings
}

func (m *Monitor) SetSettings(s Settings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings = s
	// start fresh so a changed limit is judged from now on
	m.drives = map[string]*driveState{}
}

// Observe records one reading of a drive and returns any alerts it causes
func (m *Monitor) Observe(drive string, free, total uint64, now time.Time) []Alert {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.settings.Enabled || total == 0 {
		return nil
	}

	st := m.drives[drive]
	if st == nil {
		st = &driveState{}
		m.drives[drive] = st
	}
	st.samples = append(st.samples, sample{now, free})
	for len(st.samples) > 1 && now.Sub(st.samples[0].at) > DropWindow {
		st.samples = st.samples[1:]
	}

	var alerts []Alert
	limit := uint64(m.settings.MinFreeGB * gb)
	// a limit bigger than the drive itself would always be "low"; ignore it there
	if limit > 0 && limit < total {
		switch {
		case free < limit:
			repeat := !st.lowAlert.IsZero() && now.Sub(st.lowAlert) >= lowRepeat
			if st.lowSince.IsZero() || repeat {
				if st.lowSince.IsZero() {
					st.lowSince = now
				}
				st.lowAlert = now
				alerts = append(alerts, Alert{
					Drive: drive, Kind: "low", FreeBytes: free, At: now.UnixMilli(),
					Title: fmt.Sprintf("%s is almost full", drive),
					Body:  fmt.Sprintf("Only %s free.", formatGB(free)),
				})
			}
		case float64(free) > float64(limit)*recoverMargin:
			st.lowSince, st.lowAlert = time.Time{}, time.Time{}
		}
	}

	drop := uint64(m.settings.DropGB * gb)
	if drop > 0 && (st.dropAlert.IsZero() || now.Sub(st.dropAlert) >= dropCooldown) {
		highest, highestAt := free, now
		for _, s := range st.samples {
			if s.free > highest {
				highest, highestAt = s.free, s.at
			}
		}
		if highest-free >= drop {
			st.dropAlert = now
			minutes := int(math.Max(1, math.Round(now.Sub(highestAt).Minutes())))
			alerts = append(alerts, Alert{
				Drive: drive, Kind: "dropping", FreeBytes: free, At: now.UnixMilli(),
				Title: fmt.Sprintf("%s is filling up fast", drive),
				Body: fmt.Sprintf("%s used in the last %d %s. %s free.",
					formatGB(highest-free), minutes, plural(minutes, "minute"), formatGB(free)),
			})
		}
	}
	return alerts
}

func formatGB(bytes uint64) string {
	v := float64(bytes) / gb
	if v < 10 {
		return fmt.Sprintf("%.1f GB", v)
	}
	// halves round up, as people expect (14.5 GB reads as 15 GB)
	return fmt.Sprintf("%.0f GB", math.Round(v))
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// --- settings file ---

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SysInfoPro", "alerts.json"), nil
}

// LoadSettings reads the saved settings, or the defaults if there are none
func LoadSettings() Settings {
	s := DefaultSettings()
	path, err := settingsPath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	json.Unmarshal(data, &s)
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
