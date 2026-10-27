//go:build windows

// Package notify shows Windows notifications under the app's own name.
package notify

import (
	"context"
	_ "embed"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"sysinfopro/internal/winps"

	"git.sr.ht/~jackmordaunt/go-toast/v2/wintoast"
)

//go:embed icon.png
var icon []byte

// the name notifications are shown under when the app is not installed from the Store
const displayName = "SysInfo Pro"

var (
	setup     sync.Once
	appID     string
	setupErr  error
	pushMutex sync.Mutex
)

// packageAppID returns the Store package's app ID, or "" for a plain exe.
// A packaged app must notify under this ID; Windows already knows its name and icon.
func packageAppID() string {
	// from the Windows system folder only, never from the app's folder
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentApplicationUserModelId")
	if proc.Find() != nil {
		return ""
	}
	var length uint32 = 256
	buf := make([]uint16, length)
	if ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&length)), uintptr(unsafe.Pointer(&buf[0]))); ret != 0 {
		return "" // APPMODEL_ERROR_NO_APPLICATION: not packaged
	}
	return syscall.UTF16ToString(buf)
}

func initialise() {
	if id := packageAppID(); id != "" {
		appID = id
		return
	}
	// A plain exe registers a display name and icon for itself (in the current
	// user's registry, no administrator rights needed)
	appID = displayName
	iconPath := ""
	if dir, err := os.UserConfigDir(); err == nil {
		iconPath = filepath.Join(dir, "SysInfoPro", "notification-icon.png")
		if os.MkdirAll(filepath.Dir(iconPath), 0o755) != nil || os.WriteFile(iconPath, icon, 0o644) != nil {
			iconPath = ""
		}
	}
	setupErr = wintoast.SetAppData(wintoast.AppData{AppID: displayName, IconPath: iconPath})
}

// Send shows a notification with a title and a line of text
func Send(title, body string) error {
	setup.Do(initialise)
	if setupErr != nil {
		return setupErr
	}
	pushMutex.Lock()
	defer pushMutex.Unlock()
	return wintoast.Push(appID, toastXML(title, body))
}

func toastXML(title, body string) string {
	var b strings.Builder
	b.WriteString(`<toast activationType="foreground" launch="sysinfopro"><visual><binding template="ToastGeneric"><text>`)
	xml.EscapeText(&b, []byte(title))
	b.WriteString(`</text><text>`)
	xml.EscapeText(&b, []byte(body))
	b.WriteString(`</text></binding></visual></toast>`)
	return b.String()
}

// Status reports whether Windows will show this app's notifications: "enabled",
// "off_for_app" (turned off in Settings > Notifications), "off" (all
// notifications are off), "blocked_by_policy" (an organisation has turned them
// off) or "unknown"
func Status(ctx context.Context) string {
	setup.Do(initialise)
	script := `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[string][Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('` + strings.ReplaceAll(appID, "'", "''") + `').Setting`
	out, err := winps.Run(ctx, script)
	if err != nil {
		return "unknown"
	}
	switch strings.TrimSpace(string(out)) {
	case "Enabled":
		return "enabled"
	case "DisabledForApplication":
		return "off_for_app"
	case "DisabledForUser":
		return "off"
	case "DisabledByGroupPolicy", "DisabledByManifest":
		return "blocked_by_policy"
	}
	return "unknown"
}
