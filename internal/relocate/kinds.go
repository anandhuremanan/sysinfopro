package relocate

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"sysinfopro/internal/system"
	"sysinfopro/internal/winexec"
)

// kindsOverride lets tests replace the list
var kindsOverride []kind

func kinds() []kind {
	if kindsOverride != nil {
		return kindsOverride
	}
	home, _ := os.UserHomeDir()
	local := os.Getenv("LOCALAPPDATA")
	roaming := os.Getenv("APPDATA")

	node := [][2]string{{"node.exe", "Node.js programs"}}
	return []kind{
		{
			id: "npm", name: "npm cache", group: "dev", method: "setting", folder: "npm-cache",
			locate: toolSetting("npm", "config", "get", "cache"),
			point:  runTool("npm", "config", "set", "cache"),
			procs:  node,
		},
		{
			id: "pnpm", name: "pnpm store", group: "dev", method: "setting", folder: "pnpm-store",
			// pnpm reports the store with its version folder (…\store\v10); the setting is the folder above
			locate: parentOfVersion(toolSetting("pnpm", "store", "path")),
			point:  runTool("pnpm", "config", "set", "--global", "store-dir"),
			procs:  node,
			note:   "pnpm links packages from its store into each project only when both are on the same drive. Move it only if your projects are on the new drive too, or each project gets its own full copy.",
		},
		{
			id: "yarn", name: "Yarn cache", group: "dev", method: "setting", folder: "yarn-cache",
			locate: yarnClassic(parentOfVersion(toolSetting("yarn", "cache", "dir"))),
			point:  runTool("yarn", "config", "set", "cache-folder"),
			procs:  node,
		},
		{
			id: "pip", name: "pip cache", group: "dev", method: "setting", folder: "pip-cache",
			locate: envOrDefault("PIP_CACHE_DIR", filepath.Join(local, "pip", "cache")),
			point:  setEnv("PIP_CACHE_DIR"),
			procs:  [][2]string{{"python.exe", "Python"}, {"pip.exe", "pip"}},
		},
		{
			id: "go-mod", name: "Go module cache", group: "dev", method: "setting", folder: "go-mod",
			locate: toolSetting("go", "env", "GOMODCACHE"),
			point:  goEnv("GOMODCACHE"),
			procs:  [][2]string{{"go.exe", "Go"}, {"gopls.exe", "your editor's Go support (gopls)"}},
		},
		{
			id: "go-build", name: "Go build cache", group: "dev", method: "setting", folder: "go-build",
			locate: toolSetting("go", "env", "GOCACHE"),
			point:  goEnv("GOCACHE"),
			procs:  [][2]string{{"go.exe", "Go"}, {"gopls.exe", "your editor's Go support (gopls)"}},
		},
		{
			id: "nuget", name: "NuGet packages", group: "dev", method: "setting", folder: "nuget-packages",
			locate: envOrDefault("NUGET_PACKAGES", filepath.Join(home, ".nuget", "packages")),
			point:  setEnv("NUGET_PACKAGES"),
			procs:  [][2]string{{"dotnet.exe", ".NET"}, {"devenv.exe", "Visual Studio"}, {"MSBuild.exe", "MSBuild"}},
		},
		{
			id: "gradle", name: "Gradle home", group: "dev", method: "setting", folder: "gradle",
			locate: envOrDefault("GRADLE_USER_HOME", filepath.Join(home, ".gradle")),
			point:  setEnv("GRADLE_USER_HOME"),
			procs:  [][2]string{{"java.exe", "Java programs (run gradle --stop for Gradle's background process)"}, {"studio64.exe", "Android Studio"}, {"idea64.exe", "IntelliJ IDEA"}},
			note:   "Android Studio and IntelliJ pick up the new place after they are restarted.",
		},
		{
			id: "maven", name: "Maven repository", group: "dev", method: "junction", folder: "maven-repository",
			locate: folder(filepath.Join(home, ".m2", "repository")),
			procs:  [][2]string{{"java.exe", "Java programs"}, {"idea64.exe", "IntelliJ IDEA"}},
		},
		{
			id: "cargo", name: "Cargo registry", group: "dev", method: "junction", folder: "cargo-registry",
			locate: folder(filepath.Join(envOr("CARGO_HOME", filepath.Join(home, ".cargo")), "registry")),
			procs:  [][2]string{{"cargo.exe", "Cargo"}, {"rust-analyzer.exe", "your editor's Rust support"}},
		},
		{
			id: "chrome", name: "Chrome", group: "browser", method: "junction", folder: "Chrome User Data",
			locate: folder(filepath.Join(local, "Google", "Chrome", "User Data")),
			procs:  [][2]string{{"chrome.exe", "Chrome"}},
			note:   "Profiles, history, extensions and cache. Chrome finds them through a link in the old place.",
		},
		{
			id: "edge", name: "Microsoft Edge", group: "browser", method: "junction", folder: "Edge User Data",
			locate: folder(filepath.Join(local, "Microsoft", "Edge", "User Data")),
			procs:  [][2]string{{"msedge.exe", "Edge (it can keep running in the background: turn off Startup boost in Edge's settings)"}},
			note:   "Profiles, history, extensions and cache. Edge finds them through a link in the old place.",
		},
		{
			id: "brave", name: "Brave", group: "browser", method: "junction", folder: "Brave User Data",
			locate: folder(filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data")),
			procs:  [][2]string{{"brave.exe", "Brave"}},
		},
		{
			id: "firefox", name: "Firefox profiles", group: "browser", method: "junction", folder: "Firefox Profiles",
			locate: folder(filepath.Join(roaming, "Mozilla", "Firefox", "Profiles")),
			procs:  [][2]string{{"firefox.exe", "Firefox"}},
		},
		{
			id: "firefox-cache", name: "Firefox cache", group: "browser", method: "junction", folder: "Firefox Cache",
			locate: folder(filepath.Join(local, "Mozilla", "Firefox", "Profiles")),
			procs:  [][2]string{{"firefox.exe", "Firefox"}},
		},
	}
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

// folder finds an app's fixed folder
func folder(path string) func(context.Context) (string, bool, error) {
	return func(context.Context) (string, bool, error) {
		return path, isDir(path), nil
	}
}

// envOrDefault reads a user environment variable the tool honours, falling back
// to the tool's default folder
func envOrDefault(name, fallback string) func(context.Context) (string, bool, error) {
	return func(context.Context) (string, bool, error) {
		path := system.UserVariable(name)
		if path == "" {
			path = os.Getenv(name)
		}
		if path == "" {
			path = fallback
		}
		return path, isDir(path), nil
	}
}

func setEnv(name string) func(context.Context, string) error {
	return func(_ context.Context, dir string) error {
		if err := system.SetUserVariable(name, dir); err != nil {
			return err
		}
		os.Setenv(name, dir) // so this app's next check sees it too
		return nil
	}
}

// toolSetting asks a tool where it keeps something. A tool that is not
// installed means the item is not on this PC.
func toolSetting(tool string, args ...string) func(context.Context) (string, bool, error) {
	return func(ctx context.Context) (string, bool, error) {
		if _, err := exec.LookPath(tool); err != nil {
			return "", false, nil
		}
		out, err := winexec.Output(ctx, tool, args...)
		if err != nil {
			return "", false, nil
		}
		path := strings.TrimSpace(lastLine(string(out)))
		if path == "" || !filepath.IsAbs(path) {
			return "", false, nil
		}
		return path, isDir(path), nil
	}
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

var versionFolder = regexp.MustCompile(`^v\d+$`)

// parentOfVersion turns a store path ending in a version folder (v10) into the folder above it
func parentOfVersion(locate func(context.Context) (string, bool, error)) func(context.Context) (string, bool, error) {
	return func(ctx context.Context) (string, bool, error) {
		path, _, err := locate(ctx)
		if path != "" && versionFolder.MatchString(filepath.Base(path)) {
			path = filepath.Dir(path)
		}
		return path, path != "" && isDir(path), err
	}
}

// yarnClassic only offers Yarn 1: later versions keep their cache in each project
func yarnClassic(locate func(context.Context) (string, bool, error)) func(context.Context) (string, bool, error) {
	return func(ctx context.Context) (string, bool, error) {
		if _, err := exec.LookPath("yarn"); err != nil {
			return "", false, nil
		}
		out, err := winexec.Output(ctx, "yarn", "--version")
		if err != nil || !strings.HasPrefix(strings.TrimSpace(string(out)), "1.") {
			return "", false, nil
		}
		return locate(ctx)
	}
}

func runTool(tool string, args ...string) func(context.Context, string) error {
	return func(ctx context.Context, dir string) error {
		out, err := winexec.CombinedOutput(ctx, tool, append(args, dir)...)
		if err != nil {
			return errors.New(strings.TrimSpace(lastLine(string(out) + " " + err.Error())))
		}
		return nil
	}
}

func goEnv(name string) func(context.Context, string) error {
	return func(ctx context.Context, dir string) error {
		// set as an environment variable, go env -w refuses to override it
		if system.UserVariable(name) != "" || os.Getenv(name) != "" {
			return setEnv(name)(ctx, dir)
		}
		out, err := winexec.CombinedOutput(ctx, "go", "env", "-w", name+"="+dir)
		if err != nil {
			return errors.New(strings.TrimSpace(string(out)))
		}
		return nil
	}
}
