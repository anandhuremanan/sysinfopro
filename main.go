package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/build
var assets embed.FS

// webviewDataPath keeps the embedded browser's cache in the app's own folder.
// Left unset, Wails names the folder after the exe, so every renamed copy of
// the exe would start a new cache of 30 MB or more.
func webviewDataPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "SysInfoPro", "WebView2")
}

func main() {
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
			WebviewUserDataPath: webviewDataPath(),
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
