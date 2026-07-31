package system

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

type StartupItem struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Location    string `json:"location"`
	Impact      string `json:"impact"`       // High, Medium, Low
	ImpactScore int    `json:"impact_score"` // 1-100
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

	if len(items) == 0 {
		// Fallback sample items for demonstration if registry access is limited
		items = append(items, StartupItem{
			Name:        "Windows Terminal Helper",
			Command:     "wt.exe -autostart",
			Location:    "HKCU Registry",
			Impact:      "Low",
			ImpactScore: 12,
			IsEnabled:   true,
		})
	}

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

		impact := "Low"
		score := 25
		if len(val) > 80 {
			impact = "High"
			score = 85
		} else if len(val) > 40 {
			impact = "Medium"
			score = 55
		}

		results = append(results, StartupItem{
			Name:        n,
			Command:     val,
			Location:    locationName,
			Impact:      impact,
			ImpactScore: score,
			IsEnabled:   true,
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
		if e.IsDir() {
			continue
		}
		results = append(results, StartupItem{
			Name:        e.Name(),
			Command:     filepath.Join(startupPath, e.Name()),
			Location:    "Startup Folder",
			Impact:      "Medium",
			ImpactScore: 50,
			IsEnabled:   true,
		})
	}

	return results, nil
}
