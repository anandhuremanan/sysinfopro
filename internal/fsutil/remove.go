// Package fsutil holds file system helpers shared by the cleaning tools.
package fsutil

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrStopped is returned when ctx is cancelled part way through a removal
var ErrStopped = errors.New("stopped")

// RemoveTree deletes a folder file by file so progress can be shown, calling
// onFile with the size of each removed file. Files that cannot be removed (most
// often because a program has them open) are skipped and counted; everything
// else still goes. Links and junctions are removed themselves, never followed.
// Read-only files (the Go module cache is written read-only on purpose) are
// removed in one step where Windows allows it (see removeFile), which is what
// keeps caches of many small read-only files from taking minutes.
func RemoveTree(ctx context.Context, root string, onFile func(size int64)) (skipped int, err error) {
	var dirs []string
	var firstErr error
	remember := func(err error) {
		skipped++
		if firstErr == nil {
			firstErr = err
		}
	}

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil {
			remember(err)
			if d != nil && d.IsDir() {
				dirs = append(dirs, path)
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}

		var size int64
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() {
			size = info.Size()
		}
		if err := removeFile(path); err != nil {
			remember(err)
			return nil
		}
		onFile(size)
		return nil
	})

	// deepest folders first, so each is empty by the time it is removed; a folder
	// still holding a skipped file fails here, which is expected and not counted again
	for i := len(dirs) - 1; i >= 0; i-- {
		if ctx.Err() != nil {
			break
		}
		if err := RemoveOne(dirs[i]); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if ctx.Err() != nil {
		return skipped, ErrStopped
	}
	if _, err := os.Lstat(root); err == nil {
		if firstErr == nil {
			firstErr = fmt.Errorf("some files could not be removed")
		}
		return skipped, firstErr
	}
	return skipped, nil
}

// RemoveOne removes a single file, link or empty folder, clearing a read-only
// flag if that is what stands in the way. A path that is already gone is not an error.
func RemoveOne(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	if os.Chmod(path, 0o666) == nil {
		if err2 := os.Remove(path); err2 == nil || os.IsNotExist(err2) {
			return nil
		}
	}
	return err
}
