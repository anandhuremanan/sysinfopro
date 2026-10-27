package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// wails.json holds the product version that the Store package and installer
// use, so the app reads its version from the same place
//
//go:embed wails.json
var wailsJSON []byte

var versionOnce = sync.OnceValue(func() string {
	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if json.Unmarshal(wailsJSON, &cfg) != nil || cfg.Info.ProductVersion == "" {
		return "unknown"
	}
	return cfg.Info.ProductVersion
})

// appVersion is the product version, e.g. "3.2.0"
func appVersion() string { return versionOnce() }

// windowsVersion describes the Windows release and build, e.g.
// "Windows 11 Pro 23H2 (build 22631.4317)"
func windowsVersion() string {
	name, display, build, ubr := windowsRelease()
	if name == "" {
		return runtime.GOOS
	}
	parts := []string{name}
	if display != "" {
		parts = append(parts, display)
	}
	if build != "" {
		b := build
		if ubr != 0 {
			b = fmt.Sprintf("%s.%d", build, ubr)
		}
		parts = append(parts, "(build "+b+")")
	}
	return strings.Join(parts, " ")
}
