// Package gitsweep finds every git project under a folder and reports the work
// that exists only on this PC: changes not committed, commits never pushed,
// stashes, and projects with no remote at all. Nothing is changed; git is only
// asked read-only questions, and nothing is fetched.
package gitsweep

import (
	"context"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Branch struct {
	Name string `json:"name"`
	// commits on it that no remote has
	LocalOnly int  `json:"local_only"`
	Upstream  bool `json:"upstream"`
}

type Repo struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Branch    string `json:"branch"` // "" when detached
	Changed   int    `json:"changed"`
	Untracked int    `json:"untracked"`
	Conflicts int    `json:"conflicts"`
	// commits on local branches that are on no remote
	LocalOnly int      `json:"local_only"`
	Branches  []Branch `json:"branches"` // only branches with local-only commits
	Behind    int      `json:"behind"`   // as of the last fetch
	Stashes   int      `json:"stashes"`
	Remote    string   `json:"remote"` // first remote's address, without any password
	NoRemote  bool     `json:"no_remote"`
	// a password or token is written into the remote's address
	SecretInURL bool  `json:"secret_in_url"`
	Empty       bool  `json:"empty"`       // no commits yet
	LastCommit  int64 `json:"last_commit"` // Unix ms, 0 when none
	// at-risk: work that exists only here; changes: uncommitted edits; clean
	Status string `json:"status"`
	Error  string `json:"error"`
}

type Result struct {
	Root        string `json:"root"`
	Repos       []Repo `json:"repos"`
	GitMissing  bool   `json:"git_missing"`
	AtRisk      int    `json:"at_risk"`
	WithChanges int    `json:"with_changes"`
	// projects git could not read; never counted as pushed
	Unreadable int   `json:"unreadable"`
	DurationMs int64 `json:"duration_ms"`
}

type Progress struct {
	Phase   string `json:"phase"` // finding | checking
	Found   int    `json:"found"`
	Done    int    `json:"done"`
	Current string `json:"current"`
}

// runner runs git in a folder and returns its output; tests replace it
type runner func(ctx context.Context, dir string, args ...string) (string, error)

// folders never searched for projects: too big, or full of other people's clones
var skipDirs = map[string]bool{
	"windows": true, "program files": true, "program files (x86)": true, "programdata": true,
	"$recycle.bin": true, "system volume information": true, "recovery": true, "appdata": true,
	"node_modules": true, ".venv": true, "venv": true, "__pycache__": true, "site-packages": true,
	".gradle": true, ".m2": true, ".nuget": true, ".cargo": true, ".rustup": true, "target": true,
	"vendor": true, ".pnpm-store": true, ".npm": true, ".cache": true, "bin": true, "obj": true,
	"dist": true, "build": true, ".next": true, ".nuxt": true, "scoop": true,
}

// Find walks root for git projects and checks each one
func Find(ctx context.Context, root string, onProgress func(Progress)) (*Result, error) {
	git := gitPath()
	if git == "" {
		return &Result{Root: root, Repos: []Repo{}, GitMissing: true}, nil
	}
	return find(ctx, root, gitRunner(git), onProgress)
}

func find(ctx context.Context, root string, run runner, onProgress func(Progress)) (*Result, error) {
	began := time.Now()
	res := &Result{Root: root, Repos: []Repo{}}
	var dirs []string
	last := time.Now()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || !d.IsDir() {
			if err != nil && d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if path != root && (skipDirs[strings.ToLower(name)] || d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0) {
			return fs.SkipDir
		}
		if name == ".git" {
			return fs.SkipDir
		}
		// a .git folder, or a .git file (worktrees and submodules)
		if _, err := os.Lstat(filepath.Join(path, ".git")); err == nil {
			dirs = append(dirs, path)
		}
		if onProgress != nil && time.Since(last) > 200*time.Millisecond {
			last = time.Now()
			onProgress(Progress{Phase: "finding", Found: len(dirs), Current: path})
		}
		return nil
	})
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	_ = err

	repos := make([]Repo, len(dirs))
	var done atomic.Int64
	var next atomic.Int64
	var wg sync.WaitGroup
	var mu sync.Mutex
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1) - 1)
				if i >= len(dirs) || ctx.Err() != nil {
					return
				}
				repos[i] = check(ctx, dirs[i], run)
				n := done.Add(1)
				mu.Lock()
				if onProgress != nil && time.Since(last) > 150*time.Millisecond {
					last = time.Now()
					onProgress(Progress{Phase: "checking", Found: len(dirs), Done: int(n), Current: dirs[i]})
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	rank := map[string]int{"at-risk": 0, "changes": 1, "unreadable": 2, "clean": 3}
	sort.SliceStable(repos, func(i, j int) bool {
		if rank[repos[i].Status] != rank[repos[j].Status] {
			return rank[repos[i].Status] < rank[repos[j].Status]
		}
		return repos[i].LastCommit > repos[j].LastCommit
	})
	for _, r := range repos {
		switch r.Status {
		case "at-risk":
			res.AtRisk++
		case "changes":
			res.WithChanges++
		case "unreadable":
			res.Unreadable++
		}
	}
	res.Repos = repos
	res.DurationMs = time.Since(began).Milliseconds()
	return res, nil
}

var abLine = regexp.MustCompile(`^# branch\.ab \+(\d+) -(\d+)`)

// parseStatus reads `git status --porcelain=v2 --branch`
func parseStatus(out string, r *Repo) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			head := strings.TrimPrefix(line, "# branch.head ")
			if head != "(detached)" {
				r.Branch = head
			}
		case strings.HasPrefix(line, "# branch.ab "):
			if m := abLine.FindStringSubmatch(line); m != nil {
				r.Behind, _ = strconv.Atoi(m[2])
			}
		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "):
			r.Changed++
		case strings.HasPrefix(line, "u "):
			r.Conflicts++
		case strings.HasPrefix(line, "? "):
			r.Untracked++
		}
	}
}

// cleanRemote removes any password or token from a remote's address and says whether there was one
func cleanRemote(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.User == nil || (u.Scheme != "http" && u.Scheme != "https") {
		return raw, false
	}
	_, hasPassword := u.User.Password()
	// https://TOKEN@github.com: a bare user name over https is almost always a token
	secret := hasPassword || len(u.User.Username()) >= 20
	u.User = nil
	return u.String(), secret
}

func check(ctx context.Context, dir string, run runner) Repo {
	r := Repo{Path: dir, Name: filepath.Base(dir), Branches: []Branch{}}

	out, err := run(ctx, dir, "status", "--porcelain=v2", "--branch")
	if err != nil {
		r.Error = firstLine(out)
		if r.Error == "" {
			r.Error = err.Error()
		}
		r.Status = "unreadable" // not known to be clean
		return r
	}
	parseStatus(out, &r)

	if out, err := run(ctx, dir, "log", "-1", "--format=%ct"); err == nil {
		if s, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64); err == nil {
			r.LastCommit = s * 1000
		}
	} else {
		r.Empty = true
	}

	remotes, _ := run(ctx, dir, "remote")
	names := strings.Fields(remotes)
	r.NoRemote = len(names) == 0
	if !r.NoRemote {
		if u, err := run(ctx, dir, "remote", "get-url", names[0]); err == nil {
			r.Remote, r.SecretInURL = cleanRemote(u)
		}
	}

	if !r.Empty {
		// every commit on a local branch that no remote has
		args := []string{"rev-list", "--count", "--branches"}
		if !r.NoRemote {
			args = append(args, "--not", "--remotes")
		}
		if out, err := run(ctx, dir, args...); err == nil {
			r.LocalOnly, _ = strconv.Atoi(strings.TrimSpace(out))
		}
		if r.LocalOnly > 0 {
			refs, _ := run(ctx, dir, "for-each-ref", "--format=%(refname:short)%09%(upstream)", "refs/heads")
			for i, line := range strings.Split(strings.TrimSpace(refs), "\n") {
				if i >= 15 {
					break
				}
				name, upstream, _ := strings.Cut(strings.TrimSpace(line), "\t")
				if name == "" {
					continue
				}
				a := []string{"rev-list", "--count", "refs/heads/" + name}
				if !r.NoRemote {
					a = append(a, "--not", "--remotes")
				}
				if out, err := run(ctx, dir, a...); err == nil {
					if n, _ := strconv.Atoi(strings.TrimSpace(out)); n > 0 {
						r.Branches = append(r.Branches, Branch{Name: name, LocalOnly: n, Upstream: upstream != ""})
					}
				}
			}
		}
		// on a detached HEAD, commits made since are on no branch at all, so the
		// count above misses them; they are the easiest work to lose
		if r.Branch == "" {
			a := []string{"rev-list", "--count", "HEAD", "--not", "--branches"}
			if !r.NoRemote {
				a = append(a, "--remotes")
			}
			if out, err := run(ctx, dir, a...); err == nil {
				if n, _ := strconv.Atoi(strings.TrimSpace(out)); n > 0 {
					r.LocalOnly += n
					r.Branches = append(r.Branches, Branch{Name: "(detached HEAD)", LocalOnly: n})
				}
			}
		}
	}

	if out, err := run(ctx, dir, "stash", "list"); err == nil && strings.TrimSpace(out) != "" {
		r.Stashes = len(strings.Split(strings.TrimSpace(out), "\n"))
	}

	switch {
	case r.LocalOnly > 0 || r.Stashes > 0 || (r.NoRemote && !r.Empty):
		r.Status = "at-risk"
	case r.Changed+r.Untracked+r.Conflicts > 0:
		r.Status = "changes"
	default:
		r.Status = "clean"
	}
	return r
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(strings.TrimPrefix(s, "fatal: "))
}
