//go:build !windows

package hardware

import (
	"context"
	"errors"
)

var errNotWindows = errors.New("hardware inventory needs Windows")

func readDiskRows(ctx context.Context) ([]diskRow, error) { return nil, errNotWindows }

func readSpecs(ctx context.Context) (*HardwareSpecs, error) { return nil, errNotWindows }
