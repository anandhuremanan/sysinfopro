package storage

import (
	"os"
	"path/filepath"
)

// Folder is a suggested place to watch for growth
type Folder struct {
	Label string
	Path  string
}

// WatchFolders suggests the folders where disk space usually disappears, leaving
// out any that do not exist on this PC
func WatchFolders() []Folder {
	home, _ := os.UserHomeDir()
	candidates := []Folder{
		{"Local app data", os.Getenv("LOCALAPPDATA")},
		{"Roaming app data", os.Getenv("APPDATA")},
		{"Temp", os.TempDir()},
		{"Downloads", filepath.Join(home, "Downloads")},
		{"User folder", home},
	}
	folders := []Folder{}
	for _, c := range candidates {
		if info, err := os.Stat(c.Path); c.Path != "" && err == nil && info.IsDir() {
			folders = append(folders, c)
		}
	}
	return folders
}
