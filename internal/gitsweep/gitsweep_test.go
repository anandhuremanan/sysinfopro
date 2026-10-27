package gitsweep

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStatus(t *testing.T) {
	out := "# branch.oid abc\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +2 -3\n" +
		"1 .M N... 100644 100644 100644 a b src/app.go\n2 R. N... 100644 100644 100644 a b R100 new.go\told.go\n" +
		"u UU N... 1 2 3 4 a b c conflict.go\n? notes.txt\n? tmp/\n"
	var r Repo
	parseStatus(out, &r)
	if r.Branch != "main" || r.Behind != 3 || r.Changed != 2 || r.Conflicts != 1 || r.Untracked != 2 {
		t.Errorf("got %+v", r)
	}
	var d Repo
	parseStatus("# branch.head (detached)\n", &d)
	if d.Branch != "" {
		t.Errorf("detached = %q", d.Branch)
	}
}

func TestCleanRemote(t *testing.T) {
	cases := []struct {
		in, want string
		secret   bool
	}{
		{"https://github.com/me/app.git", "https://github.com/me/app.git", false},
		{"https://me:hunter2@github.com/me/app.git", "https://github.com/me/app.git", true},
		{"https://ghp_abcdefghijklmnopqrstuvwxyz0123@github.com/me/app.git", "https://github.com/me/app.git", true},
		{"https://me@dev.azure.com/org/p/_git/app", "https://dev.azure.com/org/p/_git/app", false},
		{"git@github.com:me/app.git", "git@github.com:me/app.git", false},
		{"ssh://git@github.com/me/app.git", "ssh://git@github.com/me/app.git", false},
		{"ssh://me:hunter2@host.example/repo.git", "ssh://me@host.example/repo.git", true},
		{"ftp://me:pw@files.example/repo", "ftp://me@files.example/repo", true},
	}
	for _, c := range cases {
		got, secret := cleanRemote(c.in + "\n")
		if got != c.want || secret != c.secret {
			t.Errorf("cleanRemote(%q) = %q, %v", c.in, got, secret)
		}
	}
}

// real git, in temporary folders: a pushed project, one with unpushed work, one with no remote
func TestFindWithGit(t *testing.T) {
	git := gitPath()
	if git == "" {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(git, append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "init.defaultBranch=main", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit := func(dir, file string) {
		os.WriteFile(filepath.Join(dir, file), []byte(file), 0o644)
		run(dir, "add", ".")
		run(dir, "commit", "-q", "-m", file)
	}

	server := filepath.Join(root, "server.git")
	os.MkdirAll(server, 0o755)
	run(server, "init", "-q", "--bare")

	pushed := filepath.Join(root, "work", "pushed")
	os.MkdirAll(pushed, 0o755)
	run(pushed, "init", "-q")
	commit(pushed, "a.txt")
	os.WriteFile(filepath.Join(pushed, ".gitignore"), []byte("node_modules/"), 0o644)
	run(pushed, "add", ".gitignore")
	run(pushed, "commit", "-q", "-m", "ignore")
	run(pushed, "remote", "add", "origin", server)
	run(pushed, "push", "-q", "-u", "origin", "main")

	ahead := filepath.Join(root, "work", "ahead")
	run(root, "clone", "-q", server, ahead)
	commit(ahead, "b.txt")
	run(ahead, "checkout", "-q", "-b", "experiment")
	commit(ahead, "c.txt")
	os.WriteFile(filepath.Join(ahead, "draft.txt"), []byte("x"), 0o644)

	local := filepath.Join(root, "work", "local-only")
	os.MkdirAll(local, 0o755)
	run(local, "init", "-q")
	commit(local, "x.txt")

	// commits made on a detached HEAD belong to no branch
	detached := filepath.Join(root, "work", "detached")
	run(root, "clone", "-q", server, detached)
	run(detached, "checkout", "-q", "--detach")
	commit(detached, "lost.txt")

	edited := filepath.Join(root, "work", "edited")
	run(root, "clone", "-q", server, edited)
	os.WriteFile(filepath.Join(edited, "a.txt"), []byte("changed"), 0o644)

	// projects inside node_modules are not the user's
	hidden := filepath.Join(root, "work", "pushed", "node_modules", "dep")
	os.MkdirAll(hidden, 0o755)
	run(hidden, "init", "-q")

	res, err := find(context.Background(), filepath.Join(root, "work"), gitRunner(git), nil)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]Repo{}
	for _, r := range res.Repos {
		by[r.Name] = r
	}
	if r := by["detached"]; r.Status != "at-risk" || r.LocalOnly != 1 || len(r.Branches) != 1 || r.Branches[0].Name != "(detached HEAD)" {
		t.Errorf("detached = %+v", r)
	}
	delete(by, "detached")
	if len(res.Repos) != 5 {
		t.Fatalf("found %d projects: %+v", len(res.Repos), res.Repos)
	}
	if r := by["pushed"]; r.Status != "clean" || r.LocalOnly != 0 || r.NoRemote {
		t.Errorf("pushed = %+v", r)
	}
	if r := by["ahead"]; r.Status != "at-risk" || r.LocalOnly != 2 || r.Untracked != 1 || len(r.Branches) != 2 {
		t.Errorf("ahead = %+v", r)
	}
	if r := by["local-only"]; r.Status != "at-risk" || !r.NoRemote || r.LocalOnly != 1 {
		t.Errorf("local-only = %+v", r)
	}
	if r := by["edited"]; r.Status != "changes" || r.Changed != 1 {
		t.Errorf("edited = %+v", r)
	}
	if res.AtRisk != 3 || res.WithChanges != 1 || res.Repos[0].Status != "at-risk" {
		t.Errorf("totals: %d at risk, %d with changes, first %s", res.AtRisk, res.WithChanges, res.Repos[0].Name)
	}
	if strings.Contains(by["ahead"].Remote, "@") {
		t.Errorf("remote = %q", by["ahead"].Remote)
	}
}
