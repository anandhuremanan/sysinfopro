package hotkey

import "testing"

func TestParseCombo(t *testing.T) {
	for _, c := range Combos {
		if _, _, err := parseCombo(c); err != nil {
			t.Errorf("%s: %v", c, err)
		}
	}
	mods, vk, err := parseCombo("Ctrl+Alt+W")
	if err != nil || mods != modControl|modAlt || vk != 'W' {
		t.Errorf("Ctrl+Alt+W = %x, %x, %v", mods, vk, err)
	}
	for _, bad := range []string{"W", "Ctrl+", "Ctrl+F12", "Hyper+W", ""} {
		if _, _, err := parseCombo(bad); err == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
}
