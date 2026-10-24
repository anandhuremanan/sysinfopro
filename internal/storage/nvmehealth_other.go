//go:build !windows

package storage

import "errors"

// ReadNVMeHealth is only implemented on Windows
func ReadNVMeHealth(mount string) (*NVMeHealth, error) {
	return nil, errors.New("not supported on this platform")
}
