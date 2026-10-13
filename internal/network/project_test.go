package network

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectProjectFromSubfolder(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "work", "shop")
	write(t, filepath.Join(app, "package.json"), `{"name": "@acme/shop-web", "version": "1.0.0"}`)
	write(t, filepath.Join(app, ".git", "HEAD"), "ref: refs/heads/feature/cart\n")
	write(t, filepath.Join(app, "src", "server", "index.js"), "")

	project := DetectProject(filepath.Join(app, "src", "server"))
	if project == nil {
		t.Fatal("expected a project")
	}
	if project.Name != "@acme/shop-web" || project.Root != app || project.Branch != "feature/cart" {
		t.Errorf("got %+v, want name from package.json, root %s and branch feature/cart", project, app)
	}
}

func TestDetectProjectFallsBackToFolderName(t *testing.T) {
	root := t.TempDir()
	api := filepath.Join(root, "billing-api")
	write(t, filepath.Join(api, "go.mod"), "module example.com/billing\n")
	// the repository root is a level above the module
	write(t, filepath.Join(root, ".git", "HEAD"), "0123456789abcdef0123456789abcdef01234567\n")

	project := DetectProject(api)
	if project == nil || project.Name != "billing-api" {
		t.Fatalf("got %+v, want the folder name billing-api", project)
	}
	if project.Branch != "0123456" {
		t.Errorf("branch = %q, want the short commit id for a detached HEAD", project.Branch)
	}
}

func TestDetectProjectBrokenManifest(t *testing.T) {
	app := filepath.Join(t.TempDir(), "broken")
	write(t, filepath.Join(app, "package.json"), `{ not json`)

	project := DetectProject(app)
	if project == nil || project.Name != "broken" || project.Branch != "" {
		t.Errorf("got %+v, want folder name and no branch", project)
	}
}

func TestDetectProjectNone(t *testing.T) {
	plain := filepath.Join(t.TempDir(), "a", "b")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	// t.TempDir lives under the user profile, which may itself hold stray manifests;
	// the walk must stop before the home folder instead of reporting it as a project
	if project := DetectProject(plain); project != nil {
		home, _ := os.UserHomeDir()
		if project.Root == home || filepath.Dir(project.Root) == project.Root {
			t.Errorf("home or drive root reported as a project: %+v", project)
		}
	}
	if DetectProject("") != nil {
		t.Error("an empty folder path is not a project")
	}
}
