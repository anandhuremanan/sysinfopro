// Package winget lists the apps that Windows Package Manager can update and
// updates them, one at a time.
package winget

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"sysinfopro/internal/winexec"
)

type Package struct {
	Name      string `json:"name"`
	ID        string `json:"id"`
	Version   string `json:"version"`
	Available string `json:"available"`
	Source    string `json:"source"`
	// winget cut the ID short to fit its table, so it cannot be updated from here
	Truncated bool `json:"truncated"`
}

type Result struct {
	// ok | missing (winget is not installed) | blocked (turned off by policy) | failed
	Status   string    `json:"status"`
	Message  string    `json:"message"`
	Packages []Package `json:"packages"`
}

// winget's error code for "disabled by Group Policy"
const blockedByPolicy = 0x8A15003A

func command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "winget", args...)
	winexec.Hide(cmd)
	return cmd
}

func isBlocked(out []byte, err error) bool {
	var exit *exec.ExitError
	if errors.As(err, &exit) && uint32(exit.ExitCode()) == blockedByPolicy {
		return true
	}
	return strings.Contains(strings.ToLower(string(out)), "group policy")
}

// ListUpgrades asks winget which installed apps have a newer version
func ListUpgrades(ctx context.Context) Result {
	if _, err := exec.LookPath("winget"); err != nil {
		return Result{Status: "missing", Packages: []Package{}}
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	out, err := command(ctx, "upgrade", "--accept-source-agreements", "--disable-interactivity").CombinedOutput()
	if isBlocked(out, err) {
		return Result{Status: "blocked", Packages: []Package{}}
	}
	packages := parseUpgradeTable(out)
	if err != nil && len(packages) == 0 {
		// winget exits non-zero when there is nothing to upgrade, so only an empty
		// and unexplained result is treated as a failure
		if strings.Contains(strings.ToLower(string(out)), "no installed package") ||
			strings.Contains(strings.ToLower(string(out)), "no applicable upgrade") {
			return Result{Status: "ok", Packages: []Package{}}
		}
		return Result{Status: "failed", Message: lastLine(out), Packages: []Package{}}
	}
	return Result{Status: "ok", Packages: packages}
}

var validID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.+_-]*$`)

// Upgrade updates one app by its exact winget ID. Installers that need
// administrator rights show Windows' usual permission prompt.
func Upgrade(ctx context.Context, id string) error {
	if !validID.MatchString(id) {
		return fmt.Errorf("not a winget package ID: %q", id)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	out, err := command(ctx, "upgrade", "--id", id, "--exact", "--silent",
		"--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity").CombinedOutput()
	if err != nil {
		if isBlocked(out, err) {
			return errors.New("winget is turned off on this PC by your organisation")
		}
		if msg := lastLine(out); msg != "" {
			return errors.New(msg)
		}
		return err
	}
	return nil
}

// cleanLines splits winget's output into lines, dropping the spinner and
// progress bars it redraws with carriage returns
func cleanLines(out []byte) []string {
	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		// keep only what was drawn last on this line
		if i := strings.LastIndex(line, "\r"); i >= 0 {
			line = line[i+1:]
		}
		lines = append(lines, strings.TrimRight(line, " "))
	}
	return lines
}

func lastLine(out []byte) string {
	lines := cleanLines(out)
	for i := len(lines) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(lines[i]); s != "" && strings.Trim(s, `-\|/ `) != "" {
			return s
		}
	}
	return ""
}

func isRule(line string) bool {
	s := strings.TrimSpace(line)
	return len(s) >= 10 && strings.Trim(s, "-") == ""
}

// columnStarts returns where each header word begins, in runes. winget pads its
// columns with spaces, and header names are single words in every language it
// ships, so each word starts a column.
func columnStarts(header []rune) []int {
	var starts []int
	for i, r := range header {
		if r != ' ' && (i == 0 || header[i-1] == ' ') {
			starts = append(starts, i)
		}
	}
	return starts
}

func field(row []rune, from, to int) string {
	if from >= len(row) {
		return ""
	}
	if to < 0 || to > len(row) {
		to = len(row)
	}
	return strings.TrimSpace(string(row[from:to]))
}

// parseUpgradeTable reads the first table of `winget upgrade` output:
//
//	Name            Id               Version  Available Source
//	-------------------------------------------------------------
//	Git             Git.Git          2.44.0   2.45.1    winget
//
// Later tables (packages that "require explicit targeting") are left out.
func parseUpgradeTable(out []byte) []Package {
	lines := cleanLines(out)
	packages := []Package{}
	for i := 1; i < len(lines); i++ {
		if !isRule(lines[i]) {
			continue
		}
		starts := columnStarts([]rune(lines[i-1]))
		if len(starts) < 4 {
			return packages
		}
		end := func(col int) int {
			if col+1 < len(starts) {
				return starts[col+1]
			}
			return -1
		}
		for _, line := range lines[i+1:] {
			row := []rune(line)
			if strings.TrimSpace(line) == "" || len(row) <= starts[3] {
				break // the table ends at a blank line or the summary line
			}
			p := Package{
				Name:      field(row, starts[0], end(0)),
				ID:        field(row, starts[1], end(1)),
				Version:   field(row, starts[2], end(2)),
				Available: field(row, starts[3], end(3)),
			}
			if len(starts) > 4 {
				p.Source = field(row, starts[4], -1)
			}
			p.Truncated = strings.HasSuffix(p.ID, "…") || !validID.MatchString(p.ID)
			if p.ID != "" && p.Available != "" && utf8.ValidString(p.Name) {
				packages = append(packages, p)
			}
		}
		return packages
	}
	return packages
}
