//go:build windows

package system

import (
	"errors"
	"regexp"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	systemEnvKey = `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
	userEnvKey   = `Environment`
)

// pathKey is where the user PATH is stored; tests point it elsewhere
var pathKey = struct {
	root registry.Key
	path string
}{registry.CURRENT_USER, userEnvKey}

func readValue(root registry.Key, path, name string) (string, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()
	value, _, err := key.GetStringValue(name)
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	return value, err
}

// readStoredPath returns the system and user PATH as stored, before %VARIABLES% are expanded
func readStoredPath() (system, user string, err error) {
	system, err = readValue(registry.LOCAL_MACHINE, systemEnvKey, "Path")
	if err != nil {
		return "", "", err
	}
	user, err = readValue(pathKey.root, pathKey.path, "Path")
	return system, user, err
}

var envVarRef = regexp.MustCompile(`%([^%]+)%`)

// expandEnv expands %VARIABLES%. This app's own environment is used first; a
// variable it lacks (one set after the app started, such as an installer's
// NVM_HOME) is read from the stored user and system variables, which is what a
// new terminal would see.
func expandEnv(value string) string {
	if !strings.Contains(value, "%") {
		return value
	}
	if expanded, err := registry.ExpandString(value); err == nil {
		value = expanded
	}
	for i := 0; i < 3 && strings.Contains(value, "%"); i++ {
		value = envVarRef.ReplaceAllStringFunc(value, func(ref string) string {
			name := ref[1 : len(ref)-1]
			if v, err := readValue(registry.CURRENT_USER, userEnvKey, name); err == nil && v != "" {
				return v
			}
			if v, err := readValue(registry.LOCAL_MACHINE, systemEnvKey, name); err == nil && v != "" {
				return v
			}
			return ref
		})
	}
	return value
}

// writeUserPath stores a new user PATH, keeping %VARIABLES% unexpanded, and tells
// running programs that the environment changed (new terminals then see it)
func writeUserPath(value string) error {
	key, err := registry.OpenKey(pathKey.root, pathKey.path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if value == "" {
		err = key.DeleteValue("Path")
	} else {
		// REG_EXPAND_SZ, as Windows itself stores PATH, so %USERPROFILE% and the
		// like keep working
		err = key.SetExpandStringValue("Path", value)
	}
	if err != nil {
		return err
	}
	broadcastEnvironmentChange()
	return nil
}

var procSendMessageTimeout = windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")

func broadcastEnvironmentChange() {
	const (
		hwndBroadcast       = 0xFFFF
		wmSettingChange     = 0x001A
		smtoAbortIfHung     = 0x0002
		timeoutMilliseconds = 3000
	)
	env, _ := windows.UTF16PtrFromString("Environment")
	var result uintptr
	procSendMessageTimeout.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(env)),
		smtoAbortIfHung, timeoutMilliseconds, uintptr(unsafe.Pointer(&result)))
}
