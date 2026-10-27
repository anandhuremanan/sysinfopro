package main

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
)

// Network (UNC) paths make Windows open a connection to another computer, and
// sign in to it with the user's credentials. The screens only ever pass paths
// the user chose or that a scan found, so a network path that did not come from
// the folder picker is refused here rather than trusted.

var errNetworkPath = errors.New("network locations can only be used after choosing them with \"Choose a folder\"")

// pickedRoots remembers network folders the user chose in a dialog, so scans of
// them, and the files they find, are allowed
type pickedRoots struct {
	mu    sync.Mutex
	roots []string
}

func (p *pickedRoots) add(path string) {
	if path == "" || !isNetworkPath(path) {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.roots = append(p.roots, filepath.Clean(path))
}

// allowed is true for local paths, and for network paths inside a folder the user picked
func (p *pickedRoots) allowed(path string) bool {
	if isDevicePath(path) {
		return false
	}
	if !isNetworkPath(path) {
		return true
	}
	clean := strings.ToLower(filepath.Clean(path))
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, root := range p.roots {
		r := strings.ToLower(root)
		if clean == r || strings.HasPrefix(clean, strings.TrimRight(r, `\`)+`\`) {
			return true
		}
	}
	return false
}

// check returns errNetworkPath when any path is a network path that was not picked
func (p *pickedRoots) check(paths ...string) error {
	for _, path := range paths {
		if !p.allowed(path) {
			return errNetworkPath
		}
	}
	return nil
}

// isNetworkPath is true for \\server\share, //server/share and \\?\UNC\server\share
func isNetworkPath(path string) bool {
	p := strings.ReplaceAll(strings.TrimSpace(path), "/", `\`)
	if rest, ok := strings.CutPrefix(p, `\\?\`); ok {
		return strings.HasPrefix(strings.ToUpper(rest), `UNC\`)
	}
	return strings.HasPrefix(p, `\\`) && !strings.HasPrefix(p, `\\.\`)
}

// isDevicePath is true for \\.\ device paths, and \\?\ paths that are neither a
// drive nor a network share; none of them are folders a user picks
func isDevicePath(path string) bool {
	p := strings.ReplaceAll(strings.TrimSpace(path), "/", `\`)
	if strings.HasPrefix(p, `\\.\`) {
		return true
	}
	if rest, ok := strings.CutPrefix(p, `\\?\`); ok {
		if strings.HasPrefix(strings.ToUpper(rest), `UNC\`) {
			return false
		}
		return len(rest) < 2 || rest[1] != ':'
	}
	return false
}
