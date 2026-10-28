//go:build !windows

package shred

import (
	"context"
	"errors"
)

type Drive struct {
	Drive     string `json:"drive"`
	Kind      string `json:"kind"`
	BitLocker string `json:"bitlocker"`
}

type WipeProgress struct {
	Drive string `json:"drive"`
	Pass  int    `json:"pass"`
}

func DriveOf(path string) string { return "" }

func Describe(ctx context.Context, drive string) Drive { return Drive{Drive: drive} }

func WipeFreeSpace(ctx context.Context, drive string, progress func(WipeProgress)) error {
	return errors.New("only available on Windows")
}
