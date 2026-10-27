// Package hotkey registers one system-wide keyboard shortcut and reports each press.
package hotkey

import (
	"fmt"
	"strings"

	"sysinfopro/internal/settings"
)

// Combos the user can pick from; each is tried with Windows, which refuses one
// that another program already owns
var Combos = []string{"Ctrl+Alt+W", "Ctrl+Shift+W", "Alt+Shift+W"}

const (
	modAlt     = 0x0001
	modControl = 0x0002
	modShift   = 0x0004
	modWin     = 0x0008
)

// parseCombo turns "Ctrl+Alt+W" into Windows modifier flags and a virtual key code
func parseCombo(combo string) (mods, vk uint32, err error) {
	parts := strings.Split(combo, "+")
	for _, part := range parts[:len(parts)-1] {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "ctrl", "control":
			mods |= modControl
		case "alt":
			mods |= modAlt
		case "shift":
			mods |= modShift
		case "win":
			mods |= modWin
		default:
			return 0, 0, fmt.Errorf("unknown modifier %q in %q", part, combo)
		}
	}
	key := strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
	if len(key) != 1 || !(key[0] >= 'A' && key[0] <= 'Z' || key[0] >= '0' && key[0] <= '9') {
		return 0, 0, fmt.Errorf("unsupported key %q in %q", key, combo)
	}
	if mods == 0 {
		return 0, 0, fmt.Errorf("%q needs at least one of Ctrl, Alt, Shift or Win", combo)
	}
	// letters and digits use their ASCII codes as virtual key codes
	return mods, uint32(key[0]), nil
}

// Settings are saved between runs
type Settings struct {
	Enabled bool   `json:"enabled"`
	Combo   string `json:"combo"`
	// set when Windows refused the shortcut, usually because another app owns it
	Error string `json:"error,omitempty"`
}

func DefaultSettings() Settings {
	return Settings{Enabled: true, Combo: Combos[0]}
}

func LoadSettings() Settings {
	s := settings.Load("hotkey.json", DefaultSettings())
	s.Error = ""
	if _, _, err := parseCombo(s.Combo); err != nil {
		s.Combo = Combos[0]
	}
	return s
}

func SaveSettings(s Settings) error {
	s.Error = ""
	return settings.Save("hotkey.json", s)
}
