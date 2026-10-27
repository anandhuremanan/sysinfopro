package gitsweep

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProgramSetting(t *testing.T) {
	cases := []struct {
		name, config, want string
	}{
		{"plain", "[core]\n\trepositoryformatversion = 0\n\tbare = false\n[remote \"origin\"]\n\turl = https://example.com/a.git\n", ""},
		{"fsmonitor off is fine", "[core]\n\tfsmonitor = false\n", ""},
		{"fsmonitor on", "[core]\n\tfsmonitor = some-program\n", "core.fsmonitor"},
		{"filter section", "[filter \"x\"]\n\tclean = something\n", "[filter]"},
		{"include", "[include]\n\tpath = other.cfg\n", "[include]"},
		{"includeIf", "[includeIf \"gitdir:/x/\"]\n\tpath = other.cfg\n", "[includeif]"},
		{"diff textconv", "[diff \"pdf\"]\n\ttextconv = something\n", "diff.pdf.textconv"},
		{"external diff", "[diff]\n\texternal = something\n", "diff.external"},
		{"signature display", "[log]\n\tshowSignature = true\n", "log.showsignature"},
		{"gpg program", "[gpg]\n\tprogram = something\n", "gpg.program"},
		{"gpg format program", "[gpg \"ssh\"]\n\tprogram = something\n", "gpg.ssh.program"},
		{"ssh command", "[core]\n\tsshCommand = something\n", "core.sshcommand"},
		{"case-insensitive section", "[Core]\n\tWorktree = ../elsewhere\n", "core.worktree"},
		{"comments ignored", "# [filter \"x\"]\n; external = y\n[core]\n\tbare = false\n", ""},
	}
	for _, c := range cases {
		if got := programSetting(strings.NewReader(c.config)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSkipReason(t *testing.T) {
	root := t.TempDir()

	ok := filepath.Join(root, "ok")
	write(t, filepath.Join(ok, ".git", "config"), "[core]\n\tbare = false\n")
	if r := skipReason(ok); r != "" {
		t.Errorf("plain repo skipped: %q", r)
	}

	risky := filepath.Join(root, "risky")
	write(t, filepath.Join(risky, ".git", "config"), "[filter \"x\"]\n\tclean = something\n")
	if r := skipReason(risky); !strings.Contains(r, "not opened") {
		t.Errorf("risky repo not skipped: %q", r)
	}

	network := filepath.Join(root, "network")
	write(t, filepath.Join(network, ".git"), "gitdir: //server/share/repo.git\n")
	if r := skipReason(network); !strings.Contains(r, "network") {
		t.Errorf("network gitdir not skipped: %q", r)
	}

	// a .git file pointing at a local folder is followed and that folder's settings checked
	worktree := filepath.Join(root, "worktree")
	write(t, filepath.Join(root, "real.git", "config"), "[diff]\n\texternal = something\n")
	write(t, filepath.Join(worktree, ".git"), "gitdir: ../real.git\n")
	if r := skipReason(worktree); !strings.Contains(r, "diff.external") {
		t.Errorf("worktree target settings not checked: %q", r)
	}
}

// a skipped project never reaches git
func TestCheckSkipsBeforeRunningGit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "risky")
	write(t, filepath.Join(dir, ".git", "config"), "[core]\n\tfsmonitor = something\n")
	ran := false
	run := func(ctx context.Context, d string, args ...string) (string, error) {
		ran = true
		return "", nil
	}
	r := check(context.Background(), dir, run)
	if ran {
		t.Fatal("git was started in a project whose settings should have stopped it")
	}
	if r.Status != "skipped" || r.Error == "" {
		t.Errorf("got status %q, error %q", r.Status, r.Error)
	}
}
