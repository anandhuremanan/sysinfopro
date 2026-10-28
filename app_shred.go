package main

import (
	"errors"
	"log/slog"
	"sort"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sysinfopro/internal/shred"
)

// PickFiles asks for one or more files
func (a *App) PickFiles(title string) ([]string, error) {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	for _, p := range paths {
		a.picked.add(p)
	}
	return paths, err
}

// DescribeShredDrives reports, for each drive the chosen paths are on, whether
// it is an SSD or a hard drive and whether BitLocker protects it, so the screen
// can say what shredding can promise there
func (a *App) DescribeShredDrives(paths []string) []shred.Drive {
	seen := map[string]bool{}
	var drives []shred.Drive
	for _, p := range paths {
		d := shred.DriveOf(p)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		drives = append(drives, shred.Describe(a.ctx, d))
	}
	sort.Slice(drives, func(i, j int) bool { return drives[i].Drive < drives[j].Drive })
	return drives
}

// ShredFiles overwrites the given files (and the files in the given folders)
// with 1, 3 or 7 passes and deletes them. Progress is sent as "shred-progress".
func (a *App) ShredFiles(paths []string, passes int) *shred.Result {
	var allowed []string
	var refused []shred.Skip
	for _, p := range paths {
		if why := shred.Protected(p); why != "" {
			refused = append(refused, shred.Skip{Path: p, Reason: why})
			continue
		}
		if !a.picked.allowed(p) {
			refused = append(refused, shred.Skip{Path: p, Reason: errNetworkPath.Error()})
			continue
		}
		allowed = append(allowed, p)
	}
	ctx, done := a.shredJob.start(a.ctx)
	defer done()
	result := shred.Shred(ctx, allowed, passes, func(p shred.Progress) {
		runtime.EventsEmit(a.ctx, "shred-progress", p)
	})
	result.Skipped = append(refused, result.Skipped...)
	slog.Info("shred finished", "files", result.Shredded, "skipped", len(result.Skipped), "passes", passes)
	return result
}

// CancelShred stops shredding after the current chunk
func (a *App) CancelShred() {
	a.shredJob.stop()
}

// WipeFreeSpace overwrites a drive's free space so files deleted earlier
// cannot be recovered from it. The drive's low-space alerts pause meanwhile,
// since it fills up on purpose. Progress is sent as "wipe-progress".
func (a *App) WipeFreeSpace(drive string) error {
	ctx, done := a.wipeJob.start(a.ctx)
	defer done()
	a.wipingDrive.Store(drive)
	defer a.wipingDrive.Store("")
	err := shred.WipeFreeSpace(ctx, drive, func(p shred.WipeProgress) {
		runtime.EventsEmit(a.ctx, "wipe-progress", p)
	})
	if ctx.Err() != nil {
		return errors.New("stopped; the space it was using has been given back")
	}
	return err
}

// CancelWipe stops wiping free space and gives the space back
func (a *App) CancelWipe() {
	a.wipeJob.stop()
}
