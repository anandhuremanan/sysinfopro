package system

import (
	"strings"
	"testing"
)

// a pretend PC: files that exist, and what each program prints for its version
func fakeDev(dirs []pathDir, files map[string]string, env map[string]string) devSource {
	lower := map[string]string{}
	for k, v := range files {
		lower[strings.ToLower(k)] = v
	}
	return devSource{
		dirs:        dirs,
		pathext:     []string{".COM", ".EXE", ".BAT", ".CMD"},
		exists:      func(p string) bool { _, ok := lower[strings.ToLower(p)]; return ok },
		run:         func(exe string, _ []string) (string, bool) { out := lower[strings.ToLower(exe)]; return out, out != "" },
		env:         func(name string) string { return env[name] },
		windowsApps: `C:\Users\dev\AppData\Local\Microsoft\WindowsApps`,
	}
}

func TestParseVersion(t *testing.T) {
	cases := []struct{ id, out, want string }{
		{"python", "Python 3.12.1\r\n", "3.12.1"},
		{"node", "v20.11.0\n", "20.11.0"},
		{"java", "openjdk version \"21\" 2023-09-19\nOpenJDK Runtime", "21"},
		{"java", "java version \"1.8.0_381\"", "1.8.0"},
		{"git", "git version 2.43.0.windows.1", "2.43.0"},
		{"go", "go version go1.22.0 windows/amd64", "1.22.0"},
	}
	for _, c := range cases {
		if got := parseVersion(c.id, c.out); got != c.want {
			t.Errorf("parseVersion(%s, %q) = %q, want %q", c.id, c.out, got, c.want)
		}
	}
	if major("python", "3.12.1") != "3.12" || major("java", "1.8.0") != "1.8" || major("node", "20.11.0") != "20" {
		t.Error("major versions")
	}
}

func spec(id string) toolSpec {
	for _, s := range devToolSpecs {
		if s.id == id {
			return s
		}
	}
	panic(id)
}

func TestStorePlaceholderOnly(t *testing.T) {
	src := fakeDev([]pathDir{{path: `C:\Users\dev\AppData\Local\Microsoft\WindowsApps`, scope: "user"}},
		map[string]string{`C:\Users\dev\AppData\Local\Microsoft\WindowsApps\python.exe`: ""}, nil)
	tool := checkTool(spec("python"), src)
	if tool.Status != "fail" || tool.Copies[0].Note != "Store placeholder" || !strings.Contains(tool.Issues[0].Text, "not installed") {
		t.Errorf("got %+v", tool)
	}
}

func TestPlaceholderBeforeRealPython(t *testing.T) {
	src := fakeDev([]pathDir{
		{path: `C:\Users\dev\AppData\Local\Microsoft\WindowsApps`, scope: "user"},
		{path: `C:\Python312`, scope: "user"},
	}, map[string]string{
		`C:\Users\dev\AppData\Local\Microsoft\WindowsApps\python.exe`: "",
		`C:\Python312\python.exe`:                                     "Python 3.12.1",
	}, nil)
	tool := checkTool(spec("python"), src)
	if tool.Status != "fail" || !strings.Contains(tool.Issues[0].Text, "comes before") || !strings.Contains(tool.Issues[0].Fix, "App execution aliases") {
		t.Errorf("got %+v", tool.Issues)
	}
}

func TestShadowedVersionManager(t *testing.T) {
	src := fakeDev([]pathDir{
		{path: `C:\Program Files\nodejs`, scope: "system"},
		{path: `C:\nvm4w\nodejs`, raw: `%NVM_SYMLINK%`, scope: "user"},
	}, map[string]string{
		`C:\Program Files\nodejs\node.exe`: "v18.19.0",
		`C:\nvm4w\nodejs\node.exe`:         "v22.1.0",
	}, map[string]string{"NVM_SYMLINK": `C:\nvm4w\nodejs`})
	tool := checkTool(spec("node"), src)
	if tool.Status != "warn" || tool.Version != "18.19.0" || tool.Copies[1].Note != "nvm" {
		t.Fatalf("got %+v", tool)
	}
	if !strings.Contains(tool.Issues[0].Text, "switching versions with nvm has no effect") {
		t.Errorf("issues = %+v", tool.Issues)
	}
	if len(tool.Vars) != 1 || tool.Vars[0].Key != "NVM_SYMLINK" {
		t.Errorf("vars = %+v", tool.Vars)
	}
}

func TestJavaHomeMismatch(t *testing.T) {
	src := fakeDev([]pathDir{{path: `C:\Program Files\Common Files\Oracle\Java\javapath`, scope: "system"}},
		map[string]string{
			`C:\Program Files\Common Files\Oracle\Java\javapath\java.exe`: `java version "21.0.2"`,
			`C:\jdk-17\bin\java.exe`: `openjdk version "17.0.9"`,
		}, map[string]string{"JAVA_HOME": `C:\jdk-17`})
	tool := checkTool(spec("java"), src)
	if tool.Status != "warn" || len(tool.Issues) != 1 || !strings.Contains(tool.Issues[0].Text, "JAVA_HOME is Java 17") {
		t.Errorf("got %+v", tool.Issues)
	}
}

func TestStaleGoroot(t *testing.T) {
	src := fakeDev([]pathDir{{path: `C:\Program Files\Go\bin`, scope: "system"}},
		map[string]string{`C:\Program Files\Go\bin\go.exe`: "go version go1.22.0 windows/amd64"},
		map[string]string{"GOROOT": `C:\Go`})
	if tool := checkTool(spec("go"), src); tool.Status != "warn" || !strings.Contains(tool.Issues[0].Text, "GOROOT") {
		t.Errorf("got %+v", tool.Issues)
	}
	ok := fakeDev([]pathDir{{path: `C:\Program Files\Go\bin`, scope: "system"}},
		map[string]string{`C:\Program Files\Go\bin\go.exe`: "go version go1.22.0 windows/amd64"},
		map[string]string{"GOROOT": `C:\Program Files\Go\`})
	if tool := checkTool(spec("go"), ok); tool.Status != "ok" {
		t.Errorf("a matching GOROOT is fine: %+v", tool.Issues)
	}
}

func TestHiddenSecondVersion(t *testing.T) {
	src := fakeDev([]pathDir{{path: `C:\Python39`, scope: "system"}, {path: `C:\Python39`, scope: "user"}, {path: `C:\Python312`, scope: "user"}},
		map[string]string{`C:\Python39\python.exe`: "Python 3.9.13", `C:\Python312\python.exe`: "Python 3.12.1"}, nil)
	tool := checkTool(spec("python"), src)
	if len(tool.Copies) != 2 {
		t.Fatalf("a folder listed twice is one copy: %+v", tool.Copies)
	}
	if tool.Status != "warn" || !strings.Contains(tool.Issues[0].Text, "3.12.1 is also installed") {
		t.Errorf("got %+v", tool.Issues)
	}
}

func TestMissingAndHealthy(t *testing.T) {
	if tool := checkTool(spec("go"), fakeDev(nil, nil, nil)); tool.Status != "missing" {
		t.Errorf("missing = %+v", tool)
	}
	src := fakeDev([]pathDir{{path: `C:\Program Files\Git\cmd`, scope: "system"}},
		map[string]string{`C:\Program Files\Git\cmd\git.exe`: "git version 2.43.0.windows.1"}, nil)
	if tool := checkTool(spec("git"), src); tool.Status != "ok" || tool.Version != "2.43.0" || !tool.Copies[0].Active {
		t.Errorf("healthy = %+v", tool)
	}
}
