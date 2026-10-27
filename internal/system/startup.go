package system

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type StartupItem struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	Location string `json:"location"`
	// not measured: left empty. (They used to be guessed from the length of the
	// command line, which says nothing about how much a program slows startup.)
	Impact      string `json:"impact"`
	ImpactScore int    `json:"impact_score"`
	IsEnabled   bool   `json:"is_enabled"`
}

func GetStartupItems() ([]StartupItem, error) {
	items := []StartupItem{}

	// 1. Scan HKCU Registry Run
	hkcuItems, _ := scanRegistryKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, "HKCU Registry")
	items = append(items, hkcuItems...)

	// 2. Scan HKLM Registry Run
	hklmItems, _ := scanRegistryKey(registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Run`, "HKLM Registry")
	items = append(items, hklmItems...)

	// 3. Scan User Startup Folder
	folderItems, _ := scanStartupFolder()
	items = append(items, folderItems...)

	return items, nil
}

func scanRegistryKey(rootKey registry.Key, path string, locationName string) ([]StartupItem, error) {
	key, err := registry.OpenKey(rootKey, path, registry.READ)
	if err != nil {
		return nil, err
	}
	defer key.Close()

	names, err := key.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}

	results := []StartupItem{}
	for _, n := range names {
		val, _, err := key.GetStringValue(n)
		if err != nil || val == "" {
			continue
		}

		results = append(results, StartupItem{
			Name:      n,
			Command:   val,
			Location:  locationName,
			IsEnabled: true,
		})
	}

	return results, nil
}

func scanStartupFolder() ([]StartupItem, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	startupPath := filepath.Join(homeDir, `AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup`)
	entries, err := os.ReadDir(startupPath)
	if err != nil {
		return nil, err
	}

	results := []StartupItem{}
	for _, e := range entries {
		// desktop.ini is Explorer's folder settings file, not a program
		if e.IsDir() || strings.EqualFold(e.Name(), "desktop.ini") {
			continue
		}
		results = append(results, StartupItem{
			Name:      e.Name(),
			Command:   filepath.Join(startupPath, e.Name()),
			Location:  "Startup Folder",
			IsEnabled: true,
		})
	}

	return results, nil
}
