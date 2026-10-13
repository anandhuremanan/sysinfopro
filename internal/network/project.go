package network

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Project is the source project a process was started from
type Project struct {
	Name   string
	Root   string
	Branch string
}

// files that mark the root of a project
var projectMarkers = []string{"package.json", "go.mod", "Cargo.toml", "pyproject.toml", "pom.xml", "build.gradle", "composer.json", ".git"}

const maxProjectDepth = 8

// DetectProject walks up from dir to the nearest folder that looks like a project root.
// It returns nil for folders that are not inside a project (a drive root, the home folder).
func DetectProject(dir string) *Project {
	if dir == "" {
		return nil
	}
	dir = filepath.Clean(dir)
	home, _ := os.UserHomeDir()

	for depth := 0; depth < maxProjectDepth; depth++ {
		parent := filepath.Dir(dir)
		if parent == dir || strings.EqualFold(dir, home) {
			// reached a drive root or the home folder: stray files there are not a project
			return nil
		}

		if marker := firstMarker(dir); marker != "" {
			project := &Project{Name: filepath.Base(dir), Root: dir, Branch: gitBranch(dir)}
			if marker == "package.json" {
				if name := packageName(filepath.Join(dir, marker)); name != "" {
					project.Name = name
				}
			}
			return project
		}
		dir = parent
	}
	return nil
}

func firstMarker(dir string) string {
	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return marker
		}
	}
	return ""
}

func packageName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	manifest := struct {
		Name string `json:"name"`
	}{}
	if json.Unmarshal(data, &manifest) != nil {
		return ""
	}
	return strings.TrimSpace(manifest.Name)
}

// gitBranch reads the checked-out branch of the repository containing dir, if any
func gitBranch(dir string) string {
	for depth := 0; depth < maxProjectDepth; depth++ {
		head, err := os.ReadFile(filepath.Join(dir, ".git", "HEAD"))
		if err == nil {
			line := strings.TrimSpace(string(head))
			if branch, ok := strings.CutPrefix(line, "ref: refs/heads/"); ok {
				return branch
			}
			if len(line) >= 7 {
				return line[:7] // detached HEAD: short commit id
			}
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
