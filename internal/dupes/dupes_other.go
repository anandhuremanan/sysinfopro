//go:build !windows

package dupes

import (
	"errors"
	"io/fs"
)

func UserFolders() []Place { return []Place{} }

func isCloudOnly(info fs.FileInfo) bool { return false }

// ToRecycleBin is only available on Windows
func ToRecycleBin(path string) error { return errors.New("not supported on this platform") }
