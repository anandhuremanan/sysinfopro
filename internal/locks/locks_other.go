//go:build !windows

package locks

import "errors"

// Find is only implemented on Windows
func Find(path string) (*Result, error) {
	return nil, errors.New("finding locking processes is only supported on Windows")
}
