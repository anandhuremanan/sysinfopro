//go:build !windows

package storage

import "io/fs"

func onDisk(info fs.FileInfo) int64 { return info.Size() }
