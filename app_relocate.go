package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sysinfopro/internal/fsutil"
	"sysinfopro/internal/relocate"
	"sysinfopro/internal/storage"
)

// MoveDestination is a drive things can be moved to
type MoveDestination struct {
	Drive     string `json:"drive"` // "D:"
	FreeBytes uint64 `json:"free_bytes"`
	// Folder is the suggested folder on it, e.g. D:\Moved from C
	Folder string `json:"folder"`
}

// ListMovable lists caches, WSL disks and browser data that can be moved off
// the system drive. Sizes come separately from MeasureFolder.
func (a *App) ListMovable() []relocate.Item {
	return relocate.Detect(a.ctx)
}

// MeasureFolder adds up the size of a folder
func (a *App) MeasureFolder(path string) int64 {
	if !a.picked.allowed(path) {
		return -1
	}
	return relocate.FolderSize(path)
}

// MoveDestinations lists the drives things can be moved to: internal drives
// other than the system drive, most free space first
func (a *App) MoveDestinations() []MoveDestination {
	system := strings.TrimSuffix(strings.ToUpper(os.Getenv("SystemDrive")), `\`)
	drives, _ := storage.ListDrives()
	var out []MoveDestination
	for _, d := range drives {
		drive := strings.ToUpper(strings.TrimRight(d.Path, `\/`))
		if d.IsSystem || drive == system || !fsutil.IsFixedDrive(drive+`\`) {
			continue
		}
		out = append(out, MoveDestination{
			Drive:     drive,
			FreeBytes: d.FreeBytes,
			Folder:    filepath.Join(drive+`\`, "Moved from "+strings.TrimSuffix(system, ":")),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FreeBytes > out[j].FreeBytes })
	return out
}

// PickMoveFolder asks for a folder to move things into
func (a *App) PickMoveFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Choose where to move it"})
}

// MoveItem moves one item into a folder inside base. Progress is sent as
// "move-progress".
func (a *App) MoveItem(id, base string) (*relocate.Result, error) {
	ctx, done := a.moveJob.start(a.ctx)
	defer done()
	return relocate.Move(ctx, id, base, func(p relocate.Progress) {
		runtime.EventsEmit(a.ctx, "move-progress", p)
	})
}

// MoveItemBack returns a moved item to where it was
func (a *App) MoveItemBack(id string) (*relocate.Result, error) {
	ctx, done := a.moveJob.start(a.ctx)
	defer done()
	return relocate.MoveBack(ctx, id, func(p relocate.Progress) {
		runtime.EventsEmit(a.ctx, "move-progress", p)
	})
}

// CancelMove stops a move while it is copying; nothing changes
func (a *App) CancelMove() {
	a.moveJob.stop()
}
