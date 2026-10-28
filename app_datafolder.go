package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sysinfopro/internal/appdata"
)

// GetDataFolder reports where the app keeps its files, their size and any
// move waiting for a restart
func (a *App) GetDataFolder() appdata.Status {
	return appdata.GetStatus()
}

// ChooseDataFolder asks for a folder and, if it is suitable, sets up a move to
// a SysInfoPro folder inside it at the next start. Cancelling the dialog
// changes nothing.
func (a *App) ChooseDataFolder() (appdata.Status, error) {
	picked, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose where SysInfo Pro keeps its files",
	})
	if err != nil {
		return appdata.GetStatus(), err
	}
	if picked == "" {
		return appdata.GetStatus(), nil
	}
	target, err := appdata.RequestMove(picked)
	if err != nil {
		return appdata.GetStatus(), err
	}
	slog.Info("data folder move set up", "to", target)
	return appdata.GetStatus(), nil
}

// UseDefaultDataFolder sets up a move back to the default folder at the next start
func (a *App) UseDefaultDataFolder() (appdata.Status, error) {
	err := appdata.RequestDefault()
	return appdata.GetStatus(), err
}

// CancelDataFolderMove forgets a move that has not happened yet
func (a *App) CancelDataFolderMove() (appdata.Status, error) {
	err := appdata.CancelMove()
	return appdata.GetStatus(), err
}

// RestartApp starts a new copy of the app and closes this one. The new copy
// waits for this one to exit, then carries out any pending data move.
func (a *App) RestartApp() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not find the app's program file: %w", err)
	}
	cmd := exec.Command(exe, restartFlag, strconv.Itoa(os.Getpid()))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start SysInfo Pro again; close it and open it yourself: %w", err)
	}
	cmd.Process.Release()
	slog.Info("restarting")
	runtime.Quit(a.ctx)
	return nil
}
