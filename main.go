package main

import (
	"embed"

	"sysinfopro/internal/appdata"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/build
var assets embed.FS

func main() {
	webviewDir := ""
	if !generatingBindings {
		// after "Restart now", wait for the old window to close first: it holds
		// the single-instance lock and the files a pending move has to copy
		waitForRestart()
		// find the data folder, and finish a move the user asked for, before
		// anything opens the app's files
		appdata.Init()
		webviewDir = appdata.WebViewDir()
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "SysInfo Pro",
		Width:  768,
		Height: 460,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               singleInstanceID,
			OnSecondInstanceLaunch: app.onSecondInstance,
		},
		Bind: []interface{}{
			app,
		},
		Frameless:       true,
		CSSDragProperty: "widows",
		CSSDragValue:    "1",
		Fullscreen:      false,
		DisableResize:   true,
		Windows: &windows.Options{
			EnableSwipeGestures: false,
			// with the app's other files. Left unset, Wails names the folder after the
			// exe, so every renamed copy would start a new cache of 30 MB or more.
			WebviewUserDataPath: webviewDir,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
