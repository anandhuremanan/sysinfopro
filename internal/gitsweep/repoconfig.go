package gitsweep

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Git projects only reads status and history, but a repository's own settings
// file can tell git to start other programs while doing so. A repository copied
// from somewhere else (a zip, a USB stick, a shared folder) is not something the
// user vetted, so before git is run in a project its settings are read here as
// plain text, without running git, and projects that ask for outside programs are
// skipped and listed as such.

// skipReason says why a project is not opened, or "" when it is fine to open
func skipReason(dir string) string {
	gitDir, reason := resolveGitDir(dir)
	if reason != "" {
		return reason
	}
	if b, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil && isNetworkPath(strings.TrimSpace(string(b))) {
		return "Its git folder points to a network location, so it was not opened"
	}
	f, err := os.Open(filepath.Join(gitDir, "config"))
	if err != nil {
		return "" // no settings file: nothing to worry about
	}
	defer f.Close()
	if key := programSetting(io.LimitReader(f, 1<<20)); key != "" {
		return "Its git settings (" + key + ") would start other programs, so it was not opened"
	}
	return ""
}

// resolveGitDir finds the folder that holds the project's git data: .git itself,
// or the folder a .git file points to (worktrees and submodules)
func resolveGitDir(dir string) (string, string) {
	p := filepath.Join(dir, ".git")
	info, err := os.Lstat(p)
	if err != nil {
		return p, ""
	}
	if info.IsDir() {
		return p, ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return p, ""
	}
	line := strings.TrimSpace(string(b))
	target, ok := strings.CutPrefix(line, "gitdir:")
	if !ok {
		return p, ""
	}
	target = strings.TrimSpace(target)
	if isNetworkPath(target) {
		return "", "Its git folder points to a network location, so it was not opened"
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(dir, target)
	}
	return target, ""
}

func isNetworkPath(p string) bool {
	return strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//")
}

// programSetting returns the first setting in a git settings file that would make
// git start another program, or "" when there is none
func programSetting(r io.Reader) string {
	section := "" // "core", "diff.foo", ...
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line[0] == '[' {
			section = sectionName(line)
			base, _, _ := strings.Cut(section, ".")
			// whole sections that run programs or pull in other settings files
			switch base {
			case "filter", "include", "includeif":
				return "[" + base + "]"
			}
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.ToLower(strings.TrimSpace(value))
		full := section + "." + key
		base, _, _ := strings.Cut(section, ".")
		switch {
		case full == "core.fsmonitor" && value != "false" && value != "":
			return full
		case full == "core.sshcommand", full == "core.worktree", full == "core.pager",
			full == "diff.external", full == "gpg.program":
			return full
		case full == "log.showsignature" && value != "false":
			return full
		case base == "diff" && (key == "textconv" || key == "command"):
			return section + "." + key
		case base == "gpg" && key == "program":
			return section + "." + key
		}
	}
	return ""
}

// sectionName turns `[diff "pdf"]` into "diff.pdf" and `[Core]` into "core"
func sectionName(line string) string {
	inner := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
	name, sub, hasSub := strings.Cut(inner, " ")
	name = strings.ToLower(strings.TrimSpace(name))
	if hasSub {
		sub = strings.Trim(strings.TrimSpace(sub), `"`)
		return name + "." + sub
	}
	return name
}
