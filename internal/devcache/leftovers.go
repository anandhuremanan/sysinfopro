package devcache

import (
	"os"
	"path/filepath"
	"strings"
)

// A leftover is a folder inside a project that its tools recreate on the next
// install or build: node_modules, Rust's target, a Python virtual environment and
// so on. Each kind only counts when the project files that own it sit next to it,
// so a folder that merely shares the name (a "target" folder of photos, say) is
// never offered for deletion.
//
// dist and build are deliberately absent: those names too often hold real source,
// such as an app's icons and installer files.
type leftoverRule struct {
	folders []string
	label   string
	note    string
	// owned reports whether this folder really belongs to the tool; project is
	// the folder that contains it
	owned func(project, folder string) bool
}

func hasFile(dir string, names ...string) bool {
	for _, name := range names {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func hasMatch(dir string, patterns ...string) bool {
	for _, pattern := range patterns {
		if matches, _ := filepath.Glob(filepath.Join(dir, pattern)); len(matches) > 0 {
			return true
		}
	}
	return false
}

func always(string, string) bool { return true }

func nodeProject(project, _ string) bool { return hasFile(project, "package.json") }

func dotnetProject(project, _ string) bool {
	return hasMatch(project, "*.csproj", "*.fsproj", "*.vbproj")
}

const rebuildNote = "Recreated the next time the project's dev server or build runs"

var leftoverRules = []leftoverRule{
	{[]string{nodeModules}, "node_modules", "Restored by running your package manager's install in the project", always},
	{[]string{"target"}, "Rust build", "Rebuilt by the next cargo build",
		func(project, _ string) bool { return hasFile(project, "Cargo.toml") }},
	{[]string{"target"}, "Maven build", "Rebuilt by the next mvn package",
		func(project, _ string) bool { return hasFile(project, "pom.xml") }},
	{[]string{".venv", "venv"}, "Python environment", "Recreate it with python -m venv, then reinstall the project's packages",
		func(_, folder string) bool { return hasFile(folder, "pyvenv.cfg") }},
	{[]string{".next"}, "Next.js build", rebuildNote, nodeProject},
	{[]string{".nuxt"}, "Nuxt build", rebuildNote, nodeProject},
	{[]string{".svelte-kit"}, "SvelteKit build", rebuildNote, nodeProject},
	{[]string{".turbo"}, "Turborepo cache", rebuildNote, nodeProject},
	{[]string{".parcel-cache"}, "Parcel cache", rebuildNote, nodeProject},
	{[]string{".angular"}, "Angular cache", rebuildNote, nodeProject},
	{[]string{".gradle"}, "Gradle project cache", "Rebuilt by the next Gradle build",
		func(project, _ string) bool {
			return hasFile(project, "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts")
		}},
	{[]string{"bin"}, ".NET build output", "Rebuilt by the next dotnet build", dotnetProject},
	{[]string{"obj"}, ".NET intermediate files", "Rebuilt by the next dotnet build", dotnetProject},
	{[]string{".dart_tool"}, "Dart tool cache", "Recreated by the next pub get or flutter run",
		func(project, _ string) bool { return hasFile(project, "pubspec.yaml") }},
	{[]string{".terraform"}, "Terraform providers", "Downloaded again by terraform init",
		func(project, _ string) bool { return hasMatch(project, "*.tf") }},
}

// matchLeftover reports which kind of leftover the folder at path is, if any
func matchLeftover(path string) (leftoverRule, bool) {
	name := strings.ToLower(filepath.Base(path))
	project := filepath.Dir(path)
	for _, rule := range leftoverRules {
		for _, folder := range rule.folders {
			if name == folder && rule.owned(project, path) {
				return rule, true
			}
		}
	}
	return leftoverRule{}, false
}

// Files whose last change shows when a project was last worked on: manifests and
// lock files, the markers package managers write on every install, the Rust
// compiler's build stamp, and git's index, which changes on commits, checkouts
// and status. The leftover folder's own timestamp is checked as well.
var lastUsedMarkers = []string{
	"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb", "bun.lock",
	filepath.Join(nodeModules, ".package-lock.json"),
	filepath.Join(nodeModules, ".modules.yaml"),
	filepath.Join(nodeModules, ".yarn-state.yml"),
	filepath.Join(nodeModules, ".yarn-integrity"),
	"Cargo.toml", "Cargo.lock", filepath.Join("target", ".rustc_info.json"),
	"pom.xml", "build.gradle", "build.gradle.kts",
	"pyproject.toml", "requirements.txt", "poetry.lock", "uv.lock",
	"pubspec.yaml", "pubspec.lock", ".terraform.lock.hcl",
	filepath.Join(".git", "index"),
}
