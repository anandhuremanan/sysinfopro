//go:build windows

package shred

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sysinfopro/internal/fsutil"
	"sysinfopro/internal/storage"
	"sysinfopro/internal/winexec"
	"sysinfopro/internal/winps"
)

// Drive describes what shredding can promise on one drive
type Drive struct {
	Drive string `json:"drive"` // "C:"
	// Kind is "SSD", "NVMe SSD", "HDD" or "" when Windows does not say
	Kind string `json:"kind"`
	// BitLocker is "on", "off" or "" when it could not be read
	BitLocker string `json:"bitlocker"`
}

// DriveOf returns "C:" for a path on drive C, or "" for other paths
func DriveOf(path string) string {
	vol := filepath.VolumeName(path)
	if len(vol) == 2 && vol[1] == ':' {
		return strings.ToUpper(vol)
	}
	return ""
}

func validDrive(drive string) bool {
	return len(drive) == 2 && drive[1] == ':' && strings.ToUpper(drive)[0] >= 'A' && strings.ToUpper(drive)[0] <= 'Z'
}

// Describe reports the kind of drive and whether BitLocker protects it
func Describe(ctx context.Context, drive string) Drive {
	d := Drive{Drive: strings.ToUpper(drive)}
	if !validDrive(drive) {
		return d
	}
	d.Kind = storage.DiskType(d.Drive)
	// the same read-only property Explorer shows; no administrator rights needed.
	// The drive letter was checked above, so it is safe inside the script.
	script := `$ErrorActionPreference='SilentlyContinue'
$v = (New-Object -ComObject Shell.Application).NameSpace('` + d.Drive + `\').Self.ExtendedProperty('System.Volume.BitLockerProtection')
if ($v -eq $null) { 'none' } else { [string]$v }`
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if out, err := winps.Run(c, script); err == nil {
		// 1 on, 2 off, 3 encrypting, 4 decrypting, 5 suspended, 6 on (locked)
		switch v, _ := strconv.Atoi(strings.TrimSpace(string(out))); v {
		case 1, 3, 6:
			d.BitLocker = "on"
		case 2, 4, 5:
			d.BitLocker = "off"
		}
	}
	return d
}

// WipeProgress reports which of free-space wiping's three passes is running
type WipeProgress struct {
	Drive string `json:"drive"`
	Pass  int    `json:"pass"` // 1 zeros, 2 ones, 3 random
}

// WipeFreeSpace overwrites a drive's free space with Windows' own cipher /w,
// three passes (zeros, ones, random), so files deleted earlier cannot be
// recovered from it. It fills the drive while it runs and frees the space
// again at the end, or when stopped.
func WipeFreeSpace(ctx context.Context, drive string, progress func(WipeProgress)) error {
	if !validDrive(drive) {
		return errors.New("choose a drive")
	}
	drive = strings.ToUpper(drive)
	root := drive + `\`
	cmd, cancel := winexec.Command(ctx, "cipher.exe", "/w:"+root)
	defer cancel()
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return errors.New("could not start Windows' cipher tool")
	}

	// cipher's messages are translated, but the 0x00 and 0xFF passes are named
	// the same in every language; anything else after them is the random pass
	pass := 0
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		next := pass
		switch {
		case strings.Contains(line, "0x00"):
			next = 1
		case strings.Contains(strings.ToUpper(line), "0XFF"):
			next = 2
		case pass == 2 && line != "" && strings.Trim(line, ".") != "":
			next = 3
		}
		if next != pass {
			pass = next
			if progress != nil {
				progress(WipeProgress{Drive: drive, Pass: pass})
			}
		}
	}
	err = cmd.Wait()

	if ctx.Err() != nil {
		// stopped: give back the space its fill file was holding
		fsutil.RemoveTree(context.Background(), filepath.Join(root, "EFSTMPWP"), func(int64) {})
		return context.Canceled
	}
	if err != nil {
		os.Remove(filepath.Join(root, "EFSTMPWP"))
		return errors.New("Windows' cipher tool stopped with an error; the drive may be read-only or not allow it")
	}
	return nil
}
