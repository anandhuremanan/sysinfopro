//go:build !windows

package shred

import (
	"context"
	"errors"
)

func shredFile(ctx context.Context, path string, method []pattern, wrote func(pass int, n int64)) (string, error) {
	return "", errors.New("shredding is only available on Windows")
}
