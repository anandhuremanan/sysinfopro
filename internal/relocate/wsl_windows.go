//go:build windows

package relocate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sysinfopro/internal/vdisk"
	"sysinfopro/internal/winexec"
)

const dockerGuide = "Docker Desktop manages this disk itself. Move it in Docker Desktop: Settings > Resources > Advanced > Disk image location."

// wsl runs wsl.exe with UTF-8 output (it prints UTF-16 otherwise)
func wsl(ctx context.Context, args ...string) (string, error) {
	cmd, cancel := winexec.Command(ctx, "wsl.exe", args...)
	defer cancel()
	cmd.Env = append(os.Environ(), "WSL_UTF8=1")
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(strings.ReplaceAll(string(out), "\x00", "")), err
}

// wslCanMove is true when this WSL has `wsl --manage <name> --move`, added in
// 2024; older versions only offer export and import, which is not done here
func wslCanMove(ctx context.Context) bool {
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, _ := wsl(c, "--help")
	return strings.Contains(out, "--move")
}

func wslItems(ctx context.Context, records map[string]record, system string) []Item {
	disks := vdisk.ListDisks()
	if len(disks) == 0 {
		return nil
	}
	canMove := wslCanMove(ctx)
	var items []Item
	for _, d := range disks {
		if d.Missing {
			continue
		}
		dir := filepath.Dir(d.Path)
		size := d.OnDiskBytes
		if size == 0 {
			size = d.SizeBytes
		}
		item := Item{ID: "wsl:" + d.Name, Name: d.Name, Group: "wsl", Method: "wsl", Path: dir, SizeBytes: size, OnSystemDrive: driveOf(dir) == system}
		switch {
		case d.Kind == "docker" || strings.HasPrefix(strings.ToLower(d.Name), "docker-desktop"):
			item.ID, item.Guide = "docker:"+d.Name, dockerGuide
		case d.Version != 2:
			item.Blocker = "WSL 1 distributions cannot be moved this way"
		default:
			if rec, ok := records[item.ID]; ok && samePath(rec.To, dir) {
				item.Moved, item.MovedFrom = true, rec.From
			}
			if !canMove {
				item.Blocker = "This version of WSL cannot move distributions. Update it first: run wsl --update"
			} else {
				item.Note = "Linux " + d.Name + " is shut down while it moves; any programs running in it stop."
			}
		}
		items = append(items, item)
	}
	return items
}

func findDisk(name string) (vdisk.Disk, bool) {
	for _, d := range vdisk.ListDisks() {
		if d.Kind == "wsl" && d.Name == name && !d.Missing {
			return d, true
		}
	}
	return vdisk.Disk{}, false
}

func moveWSL(ctx context.Context, name, base string) (*Result, error) {
	d, ok := findDisk(name)
	if !ok {
		return nil, errors.New("that WSL distribution was not found")
	}
	if !wslCanMove(ctx) {
		return nil, errors.New("this version of WSL cannot move distributions; run wsl --update first")
	}
	from := filepath.Dir(d.Path)
	to := filepath.Join(base, "WSL", name)
	if err := checkTarget(from, to, d.OnDiskBytes); err != nil {
		return nil, err
	}
	if err := wslMove(ctx, name, to); err != nil {
		return nil, err
	}
	saveRecord("wsl:"+name, &record{From: from, To: to, Method: "wsl", When: time.Now()})
	slog.Info("moved WSL distribution", "name", name, "to", to)
	return &Result{ID: "wsl:" + name, To: to}, nil
}

func moveWSLBack(ctx context.Context, name string, rec record) (*Result, error) {
	if _, ok := findDisk(name); !ok {
		return nil, errors.New("that WSL distribution was not found")
	}
	if err := wslMove(ctx, name, rec.From); err != nil {
		return nil, err
	}
	saveRecord("wsl:"+name, nil)
	removeIfEmpty(rec.To)
	removeIfEmpty(filepath.Dir(rec.To))
	return &Result{ID: "wsl:" + name, To: rec.From}, nil
}

// wslMove shuts the distribution down and lets WSL move its disk, then checks
// WSL now reports the new place
func wslMove(ctx context.Context, name, to string) error {
	c, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()
	wsl(c, "--terminate", name)
	if out, err := wsl(c, "--manage", name, "--move", to); err != nil {
		return fmt.Errorf("WSL could not move it: %s", lastLine(out))
	}
	if d, ok := findDisk(name); !ok || !samePath(filepath.Dir(d.Path), to) {
		return errors.New("WSL finished, but does not report the new place; check it with wsl --list --verbose")
	}
	return nil
}
