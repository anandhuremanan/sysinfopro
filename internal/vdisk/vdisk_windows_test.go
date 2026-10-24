//go:build windows

package vdisk

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

const fsctlSetSparse = 0x000900C4

func TestAllocatedSeesSparseFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "disk.vhdx")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	var returned uint32
	if err := windows.DeviceIoControl(windows.Handle(f.Fd()), fsctlSetSparse, nil, 0, nil, 0, &returned, nil); err != nil {
		f.Close()
		t.Skipf("this file system cannot make sparse files: %v", err)
	}
	// a 1 GB file that holds 64 KB of real data
	if err := f.Truncate(1 << 30); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 64<<10)
	for i := range data {
		data[i] = byte(i)
	}
	if _, err := f.WriteAt(data, 0); err != nil {
		t.Fatal(err)
	}
	f.Sync()
	f.Close()

	onDisk, sparse := allocated(path, 1<<30)
	if !sparse {
		t.Error("file should be reported as sparse")
	}
	if onDisk <= 0 || onDisk >= 1<<20 {
		t.Errorf("on disk = %d bytes, want a few KB for a mostly empty sparse file", onDisk)
	}

	disk := Disk{Path: path}
	measure(&disk)
	if disk.Missing || disk.SizeBytes != 1<<30 || disk.OnDiskBytes != onDisk {
		t.Errorf("measure = %+v", disk)
	}
}
