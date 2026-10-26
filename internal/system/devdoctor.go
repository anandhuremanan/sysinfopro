package system

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Dev doctor: for each common developer tool, which copy actually runs when its
// command is typed in a new terminal, which other copies are installed, and the
// settings that quietly point builds at a different one.

type ToolCopy struct {
	Path    string `json:"path"`
	Scope   string `json:"scope"`   // system | user: which PATH list it was found through
	Version string `json:"version"` // "" when it did not say
	Active  bool   `json:"active"`  // this is the one that runs
	// who put it there, when it is a version manager or a stand-in
	Note string `json:"note"`
}

type ToolIssue struct {
	Level string `json:"level"` // warn | fail
	Text  string `json:"text"`
	Fix   string `json:"fix"`
}

type DevTool struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Command string      `json:"command"`
	Status  string      `json:"status"` // ok | warn | fail | missing
	Version string      `json:"version"`
	Copies  []ToolCopy  `json:"copies"`
	Issues  []ToolIssue `json:"issues"`
	// settings that matter for this tool, as stored (JAVA_HOME and the like)
	Vars []EnvVariable `json:"vars"`
}

type toolSpec struct {
	id, name, command string
	args              []string
	vars              []string
}

var devToolSpecs = []toolSpec{
	{id: "python", name: "Python", command: "python", args: []string{"--version"}, vars: []string{"PYTHONHOME", "PYTHONPATH"}},
	{id: "node", name: "Node.js", command: "node", args: []string{"--version"}, vars: []string{"NVM_HOME", "NVM_SYMLINK"}},
	{id: "java", name: "Java", command: "java", args: []string{"-version"}, vars: []string{"JAVA_HOME"}},
	{id: "git", name: "Git", command: "git", args: []string{"--version"}},
	{id: "go", name: "Go", command: "go", args: []string{"version"}, vars: []string{"GOROOT", "GOPATH"}},
	{id: "dotnet", name: ".NET SDK", command: "dotnet", args: []string{"--version"}, vars: []string{"DOTNET_ROOT"}},
	{id: "rust", name: "Rust (cargo)", command: "cargo", args: []string{"--version"}},
}

// one folder on PATH, in the order Windows searches them
type pathDir struct {
	path, raw, scope string
}

type devSource struct {
	dirs    []pathDir
	pathext []string
	exists  func(path string) bool
	// runs the program and returns what it printed and whether it exited cleanly
	run func(exe string, args []string) (string, bool)
	// a stored environment variable, expanded; "" when not set
	env         func(name string) string
	windowsApps string
}

var (
	javaVersion = regexp.MustCompile(`version "?(\d+(?:\.\d+)*)`)
	anyVersion  = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?`)
)

func parseVersion(id, out string) string {
	if id == "java" {
		if m := javaVersion.FindStringSubmatch(out); m != nil {
			return m[1]
		}
	}
	return anyVersion.FindString(out)
}

// major is the part of a version that matters for "same version?": 3.12 for
// Python, 20 for Node, 17 for Java (1.8 stays 1.8)
func major(id, v string) string {
	parts := strings.Split(v, ".")
	switch {
	case len(parts) >= 2 && (id == "python" || id == "go" || parts[0] == "1"):
		return parts[0] + "." + parts[1]
	default:
		return parts[0]
	}
}

// manager names the version manager or stand-in that owns a folder, if any
func manager(dir pathDir, src devSource) string {
	p := strings.ToLower(dir.path) + `\`
	raw := strings.ToLower(dir.raw)
	switch {
	case strings.Contains(raw, "%nvm_symlink%") || strings.Contains(raw, "%nvm_home%"),
		strings.Contains(p, `\nvm\`), strings.Contains(p, `\nvm4w\`):
		return "nvm"
	case strings.Contains(p, `\volta\`):
		return "Volta"
	case strings.Contains(p, `\fnm`):
		return "fnm"
	case strings.Contains(p, `pyenv`):
		return "pyenv"
	case strings.Contains(p, `\.cargo\bin\`):
		return "rustup"
	case strings.Contains(p, `\scoop\shims\`):
		return "Scoop"
	case strings.Contains(p, `\chocolatey\bin\`):
		return "Chocolatey"
	case strings.Contains(p, `\common files\oracle\java\javapath`):
		return "Oracle installer"
	}
	if nvmHome := src.env("NVM_SYMLINK"); nvmHome != "" && strings.EqualFold(strings.TrimRight(dir.path, `\`), strings.TrimRight(nvmHome, `\`)) {
		return "nvm"
	}
	return ""
}

// managers that switch between versions; when one of them is not first, switching does nothing
var switchers = map[string]bool{"nvm": true, "Volta": true, "fnm": true, "pyenv": true, "rustup": true}

func isStoreAlias(dir pathDir, src devSource) bool {
	return src.windowsApps != "" && strings.EqualFold(strings.TrimRight(dir.path, `\`), strings.TrimRight(src.windowsApps, `\`))
}

// findCopies lists every program the command could start, in PATH order
func findCopies(command string, src devSource) ([]ToolCopy, []pathDir) {
	copies := []ToolCopy{}
	dirs := []pathDir{}
	seen := map[string]bool{}
	for _, d := range src.dirs {
		if strings.TrimSpace(d.path) == "" {
			continue
		}
		for _, ext := range src.pathext {
			full := filepath.Join(d.path, command+strings.ToLower(ext))
			key := strings.ToLower(full)
			if seen[key] || !src.exists(full) {
				continue
			}
			seen[key] = true
			copies = append(copies, ToolCopy{Path: full, Scope: d.scope})
			dirs = append(dirs, d)
			break // within one folder, the first matching extension wins
		}
	}
	return copies, dirs
}

func checkTool(spec toolSpec, src devSource) DevTool {
	t := DevTool{ID: spec.id, Name: spec.name, Command: spec.command, Issues: []ToolIssue{}, Vars: []EnvVariable{}}
	for _, v := range spec.vars {
		if val := src.env(v); val != "" {
			t.Vars = append(t.Vars, EnvVariable{Key: v, Value: val})
		}
	}

	copies, dirs := findCopies(spec.command, src)
	// ask every copy its version at once; java in particular is slow to start
	var wg sync.WaitGroup
	for i := range copies {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if out, ok := src.run(copies[i].Path, spec.args); ok {
				copies[i].Version = parseVersion(spec.id, out)
			}
		}(i)
	}
	wg.Wait()

	for i := range copies {
		if m := manager(dirs[i], src); m != "" {
			copies[i].Note = m
		}
		if isStoreAlias(dirs[i], src) {
			if copies[i].Version == "" {
				copies[i].Note = "Store placeholder"
			} else {
				copies[i].Note = "Microsoft Store"
			}
		}
	}
	t.Copies = copies
	add := func(level, text, fix string) { t.Issues = append(t.Issues, ToolIssue{level, text, fix}) }

	if len(copies) == 0 {
		t.Status = "missing"
		return t
	}
	copies[0].Active = true
	active := copies[0]
	t.Version = active.Version

	switch {
	case active.Note == "Store placeholder":
		real := ""
		for _, c := range copies[1:] {
			if c.Version != "" {
				real = c.Path
				break
			}
		}
		if real != "" {
			add("fail", fmt.Sprintf("Typing %s opens the Microsoft Store, because Windows' placeholder comes before %s in PATH.", spec.command, real),
				fmt.Sprintf("Turn off the %s.exe alias in Settings > Apps > Advanced app settings > App execution aliases.", spec.command))
		} else {
			add("fail", fmt.Sprintf("%s is not installed: typing %s only opens the Microsoft Store.", spec.name, spec.command),
				"Install it from python.org with \"Add python.exe to PATH\" ticked, or from the Microsoft Store.")
		}
	case active.Version == "":
		add("warn", fmt.Sprintf("%s did not answer when asked its version, so it may be broken.", active.Path),
			"Reinstall it, or remove its folder from PATH if it is a leftover.")
	}

	// a version manager that is not first cannot switch anything
	if !switchers[active.Note] {
		for _, c := range copies[1:] {
			if switchers[c.Note] {
				add("warn", fmt.Sprintf("You have %s, but %s comes first in PATH, so switching versions with %s has no effect.", c.Note, active.Path, c.Note),
					fmt.Sprintf("Uninstall the other copy, or move %s above %s in PATH.", filepath.Dir(c.Path), filepath.Dir(active.Path)))
				break
			}
		}
	}

	// other installed versions that never run
	if active.Version != "" {
		var others []string
		for _, c := range copies[1:] {
			if c.Version != "" && major(spec.id, c.Version) != major(spec.id, active.Version) && c.Note != "Store placeholder" {
				others = append(others, c.Version)
			}
		}
		if len(others) > 0 && !switchers[active.Note] {
			add("warn", fmt.Sprintf("%s %s runs, but %s is also installed further down PATH and never runs.", spec.name, active.Version, strings.Join(unique(others), " and ")),
				"If you meant to use the other one, move its folder above the first in PATH.")
		}
	}

	checkVars(&t, spec, src, active)

	t.Status = "ok"
	for _, i := range t.Issues {
		if i.Level == "fail" {
			t.Status = "fail"
			break
		}
		t.Status = "warn"
	}
	return t
}

// checkVars looks at the settings build tools read instead of PATH
func checkVars(t *DevTool, spec toolSpec, src devSource, active ToolCopy) {
	add := func(level, text, fix string) { t.Issues = append(t.Issues, ToolIssue{level, text, fix}) }
	switch spec.id {
	case "java":
		home := src.env("JAVA_HOME")
		if home == "" {
			return
		}
		exe := filepath.Join(home, "bin", "java.exe")
		if !src.exists(exe) {
			add("warn", fmt.Sprintf("JAVA_HOME points to %s, which has no Java in it. Maven, Gradle and IDEs will fail.", home),
				`Set JAVA_HOME to the JDK folder (the one that contains bin\java.exe).`)
			return
		}
		out, ok := src.run(exe, spec.args)
		homeVersion := ""
		if ok {
			homeVersion = parseVersion("java", out)
		}
		if homeVersion != "" && active.Version != "" && major("java", homeVersion) != major("java", active.Version) {
			add("warn", fmt.Sprintf("The java command runs Java %s, but JAVA_HOME is Java %s, so builds use a different Java than the terminal.", active.Version, homeVersion),
				"Point JAVA_HOME and PATH at the same JDK.")
		}
	case "go":
		root := src.env("GOROOT")
		if root == "" {
			return
		}
		goDir := filepath.Dir(filepath.Dir(active.Path)) // ...\Go\bin\go.exe -> ...\Go
		if !strings.EqualFold(strings.TrimRight(root, `\`), strings.TrimRight(goDir, `\`)) {
			add("warn", fmt.Sprintf("GOROOT is set to %s, but the go command is in %s. A leftover GOROOT breaks builds after an update.", root, goDir),
				"Remove the GOROOT variable: Go finds its own folder.")
		}
	case "python":
		if home := src.env("PYTHONHOME"); home != "" {
			add("warn", fmt.Sprintf("PYTHONHOME is set (%s). It makes every Python load its libraries from there, which breaks other versions and virtual environments.", home),
				"Remove PYTHONHOME unless a program you use needs it.")
		}
	}
}

func unique(list []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range list {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// CheckDevTools resolves each tool against the PATH new terminals get
func CheckDevTools() []DevTool {
	src := liveDevSource()
	tools := make([]DevTool, len(devToolSpecs))
	var wg sync.WaitGroup
	for i, spec := range devToolSpecs {
		wg.Add(1)
		go func(i int, spec toolSpec) {
			defer wg.Done()
			tools[i] = checkTool(spec, src)
		}(i, spec)
	}
	wg.Wait()
	return tools
}

func liveDevSource() devSource {
	src := devSource{
		exists:      func(p string) bool { info, err := os.Lstat(p); return err == nil && !info.IsDir() },
		run:         runQuiet,
		env:         storedVar,
		windowsApps: filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WindowsApps"),
	}
	exts := storedVar("PATHEXT")
	if exts == "" {
		exts = ".COM;.EXE;.BAT;.CMD"
	}
	for _, e := range strings.Split(exts, ";") {
		if e = strings.TrimSpace(e); e != "" {
			src.pathext = append(src.pathext, e)
		}
	}
	system, user, err := readStoredPath()
	if err != nil {
		for _, p := range splitPath(os.Getenv("PATH")) {
			src.dirs = append(src.dirs, pathDir{path: p, raw: p, scope: "process"})
		}
		return src
	}
	for _, p := range splitPath(system) {
		src.dirs = append(src.dirs, pathDir{path: expandEnv(p), raw: p, scope: "system"})
	}
	for _, p := range splitPath(user) {
		src.dirs = append(src.dirs, pathDir{path: expandEnv(p), raw: p, scope: "user"})
	}
	return src
}
