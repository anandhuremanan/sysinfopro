package main

import "testing"

func TestNetworkPathsNeedPicking(t *testing.T) {
	var p pickedRoots
	local := []string{`C:\Users\dev\Downloads`, `D:\`, `\\?\C:\very\long\path`, `relative\path`}
	for _, path := range local {
		if !p.allowed(path) {
			t.Errorf("local path refused: %s", path)
		}
	}
	network := []string{`\\server\share\x`, `//server/share/x`, `\\?\UNC\server\share\x`}
	for _, path := range network {
		if p.allowed(path) {
			t.Errorf("unpicked network path allowed: %s", path)
		}
	}
	device := []string{`\\.\PhysicalDrive0`, `\\?\GLOBALROOT\Device\x`}
	for _, path := range device {
		if p.allowed(path) {
			t.Errorf("device path allowed: %s", path)
		}
	}

	p.add(`\\server\share\photos`)
	if !p.allowed(`\\server\share\photos`) || !p.allowed(`\\SERVER\share\Photos\2024\a.jpg`) {
		t.Error("a picked network folder and what is inside it should be allowed")
	}
	if p.allowed(`\\server\share\photos-other`) || p.allowed(`\\server\share`) {
		t.Error("only the picked folder and its contents are allowed")
	}
	if err := p.check(`C:\a`, `\\other\x`); err == nil {
		t.Error("check should fail when any path is refused")
	}
}
