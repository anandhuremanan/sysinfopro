//go:build windows

package handover

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"sysinfopro/internal/winexec"
	"sysinfopro/internal/winps"
)

// sizing folders stops after this long and reports "at least"
const sizeBudget = 5 * time.Second

// Check gathers everything at once and returns the checklist
func Check(ctx context.Context) Report {
	began := time.Now()
	f := facts{wifi: -1, creds: -1}
	var wg sync.WaitGroup
	run := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}

	run(func() { f.join = parseDsreg(output(ctx, "dsregcmd", "/status")) })
	run(func() { f.wifi, _ = countWifiProfiles(output(ctx, "netsh", "wlan", "show", "profiles")) })
	run(func() { f.creds = countCredentials(output(ctx, "cmdkey", "/list")) })
	run(func() { f.bitlocker, f.licence = deviceState(ctx) })
	run(func() { f.folders = folderSizes(ctx) })

	f.msAccounts = subkeyNames(registry.CURRENT_USER, `Software\Microsoft\IdentityCRL\StoredIdentities`)
	f.oneDrive, f.redirected = oneDrive()
	f.browsers = browsers()
	f.secrets = devSecrets()
	wg.Wait()

	r := build(f)
	r.DurationMs = time.Since(began).Milliseconds()
	return r
}

// output runs a Windows tool without a console window, for at most 8 seconds
func output(ctx context.Context, name string, args ...string) string {
	c, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, name, args...)
	winexec.Hide(cmd)
	out, _ := cmd.Output()
	return decode(out)
}

// some tools write UTF-16 when their output is not a console
func decode(b []byte) string {
	if len(b) >= 2 && (b[0] == 0xFF && b[1] == 0xFE || len(b) > 3 && b[1] == 0 && b[3] == 0) {
		if b[0] == 0xFF && b[1] == 0xFE {
			b = b[2:]
		}
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
		}
		return string(utf16.Decode(u))
	}
	return string(b)
}

func subkeyNames(root registry.Key, path string) []string {
	k, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer k.Close()
	names, _ := k.ReadSubKeyNames(-1)
	return names
}

func stringValue(root registry.Key, path, name string) string {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, _ := k.GetStringValue(name)
	return v
}

// --- OneDrive ---

var personalFolders = []struct {
	label string
	id    *windows.KNOWNFOLDERID
}{
	{"Documents", windows.FOLDERID_Documents},
	{"Desktop", windows.FOLDERID_Desktop},
	{"Pictures", windows.FOLDERID_Pictures},
	{"Videos", windows.FOLDERID_Videos},
	{"Music", windows.FOLDERID_Music},
	{"Downloads", windows.FOLDERID_Downloads},
}

func oneDrive() (accounts, redirected []string) {
	const base = `Software\Microsoft\OneDrive\Accounts`
	var roots []string
	for _, sub := range subkeyNames(registry.CURRENT_USER, base) {
		email := stringValue(registry.CURRENT_USER, base+`\`+sub, "UserEmail")
		if email == "" {
			continue
		}
		accounts = append(accounts, email)
		if root := stringValue(registry.CURRENT_USER, base+`\`+sub, "UserFolder"); root != "" {
			roots = append(roots, strings.ToLower(strings.TrimRight(root, `\`))+`\`)
		}
	}
	for _, pf := range personalFolders {
		path, err := windows.KnownFolderPath(pf.id, 0)
		if err != nil {
			continue
		}
		p := strings.ToLower(path) + `\`
		for _, root := range roots {
			if strings.HasPrefix(p, root) {
				redirected = append(redirected, pf.label)
				break
			}
		}
	}
	return
}

// --- browsers ---

func browsers() []browser {
	local := os.Getenv("LOCALAPPDATA")
	out := []browser{}
	for _, c := range []struct{ name, dir string }{
		{"Chrome", filepath.Join(local, "Google", "Chrome", "User Data")},
		{"Edge", filepath.Join(local, "Microsoft", "Edge", "User Data")},
		{"Brave", filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data")},
	} {
		data, err := os.ReadFile(filepath.Join(c.dir, "Local State"))
		if err != nil {
			continue
		}
		out = append(out, browser{name: c.name, accounts: chromiumAccounts(data), logins: -1})
	}

	ffDir := filepath.Join(os.Getenv("APPDATA"), "Mozilla", "Firefox")
	if ini, err := os.ReadFile(filepath.Join(ffDir, "profiles.ini")); err == nil {
		b := browser{name: "Firefox"}
		paths, relative := firefoxProfiles(string(ini))
		for i, p := range paths {
			dir := filepath.FromSlash(p)
			if relative[i] {
				dir = filepath.Join(ffDir, dir)
			}
			if f, err := os.Open(filepath.Join(dir, "signedInUser.json")); err == nil {
				b.accounts = append(b.accounts, firefoxAccountFrom(f))
				f.Close()
			}
			if f, err := os.Open(filepath.Join(dir, "logins.json")); err == nil {
				b.logins += firefoxLoginsFrom(f)
				f.Close()
			}
		}
		if len(paths) > 0 {
			out = append(out, b)
		}
	}
	return out
}

// --- developer keys and tokens: only whether they exist ---

func devSecrets() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	in := func(parts ...string) string { return filepath.Join(append([]string{home}, parts...)...) }
	exists := func(p string) bool { info, err := os.Stat(p); return err == nil && info.Size() > 0 }
	// read line by line, so a file of tokens is never held in memory
	contains := func(p, s string) bool {
		f, err := os.Open(p)
		if err != nil {
			return false
		}
		defer f.Close()
		return containsLine(f, s)
	}

	var found []string
	if entries, err := os.ReadDir(in(".ssh")); err == nil {
		keys := 0
		for _, e := range entries {
			n := strings.ToLower(e.Name())
			if e.IsDir() || strings.HasSuffix(n, ".pub") {
				continue
			}
			if strings.HasPrefix(n, "id_") || strings.HasSuffix(n, ".pem") || strings.HasSuffix(n, ".ppk") {
				keys++
			}
		}
		if keys == 1 {
			found = append(found, "an SSH private key")
		} else if keys > 1 {
			found = append(found, plural(keys, "SSH private key", "SSH private keys"))
		}
	}
	checks := []struct {
		label string
		ok    bool
	}{
		{"saved Git passwords (.git-credentials)", exists(in(".git-credentials"))},
		{"GitHub CLI sign-in", exists(filepath.Join(os.Getenv("APPDATA"), "GitHub CLI", "hosts.yml")) || exists(in(".config", "gh", "hosts.yml"))},
		{"AWS keys", exists(in(".aws", "credentials"))},
		{"Azure CLI sign-in", exists(in(".azure", "msal_token_cache.bin")) || exists(in(".azure", "msal_token_cache.json")) || exists(in(".azure", "accessTokens.json"))},
		{"Kubernetes config", exists(in(".kube", "config"))},
		{"Docker registry sign-in", contains(in(".docker", "config.json"), `"auth"`)},
		{"npm token (.npmrc)", contains(in(".npmrc"), "_authToken")},
		{"PyPI token (.pypirc)", contains(in(".pypirc"), "password")},
		{"saved passwords in _netrc", exists(in("_netrc")) || exists(in(".netrc"))},
	}
	for _, c := range checks {
		if c.ok {
			found = append(found, c.label)
		}
	}
	return found
}

// --- encryption and licence, in one PowerShell run ---

const deviceScript = `$ErrorActionPreference='SilentlyContinue'
$bl = $null
try { $bl = (New-Object -ComObject Shell.Application).NameSpace($env:SystemDrive).Self.ExtendedProperty('System.Volume.BitLockerProtection') } catch {}
$lic = Get-CimInstance -ClassName SoftwareLicensingProduct -Filter "ApplicationID='55c92734-d682-4d71-983e-d6ec3f16059f' AND PartialProductKey IS NOT NULL" | Select-Object -First 1
[pscustomobject]@{ bitlocker = $bl; licence = [string]$lic.Description } | ConvertTo-Json -Compress`

func deviceState(ctx context.Context) (*int, string) {
	// through winps, like every other PowerShell query: the script arrives on
	// standard input, so execution policy does not apply and nothing is bypassed
	c, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	b, err := winps.Run(c, deviceScript)
	if err != nil {
		return nil, ""
	}
	out := string(b)
	var v struct {
		BitLocker *int   `json:"bitlocker"`
		Licence   string `json:"licence"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(out)), &v) != nil {
		return nil, ""
	}
	return v.BitLocker, licenceChannel(v.Licence)
}

// --- folder sizes ---

func folderSizes(ctx context.Context) []FolderSize {
	deadline := time.Now().Add(sizeBudget)
	var list []FolderSize
	var skip []string // personal folders, so a moved one is not counted twice
	for _, pf := range personalFolders {
		path, err := windows.KnownFolderPath(pf.id, 0)
		if err != nil {
			continue
		}
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			continue
		}
		list = append(list, FolderSize{Label: pf.label, Path: path, Kind: "personal"})
		skip = append(skip, strings.ToLower(path))
	}
	if home, err := os.UserHomeDir(); err == nil {
		skip = append(skip, strings.ToLower(home))
	}
	for _, d := range fixedDrives() {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || isSystemRoot(e.Name()) {
				continue
			}
			path := filepath.Join(d, e.Name())
			if hiddenOrSystem(path) || coveredBy(path, skip) {
				continue
			}
			list = append(list, FolderSize{Label: e.Name(), Path: path, Kind: "other"})
		}
	}

	// size them all at once, sharing the time budget
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i := range list {
		wg.Add(1)
		go func(fs *FolderSize) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			fs.Bytes, fs.Files, fs.Partial = sizeOf(ctx, fs.Path, deadline)
		}(&list[i])
	}
	wg.Wait()

	// keep the biggest outside folders; empty ones are not worth listing
	var personal, other []FolderSize
	for _, f := range list {
		if f.Kind == "personal" {
			personal = append(personal, f)
		} else if f.Bytes > 0 {
			other = append(other, f)
		}
	}
	sort.Slice(other, func(i, j int) bool { return other[i].Bytes > other[j].Bytes })
	if len(other) > 20 {
		other = other[:20]
	}
	return append(personal, other...)
}

// coveredBy is true when path is one of the folders already listed, or holds one
func coveredBy(path string, folders []string) bool {
	p := strings.ToLower(path)
	for _, f := range folders {
		if p == f || strings.HasPrefix(f, p+`\`) {
			return true
		}
	}
	return false
}

func fixedDrives() []string {
	var drives []string
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	for i := 0; i < 26; i++ {
		if mask&(1<<i) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		p, _ := windows.UTF16PtrFromString(root)
		if windows.GetDriveType(p) == windows.DRIVE_FIXED {
			drives = append(drives, root)
		}
	}
	return drives
}

func hiddenOrSystem(path string) bool {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return true
	}
	a, err := windows.GetFileAttributes(p)
	if err != nil {
		return true
	}
	return a&(windows.FILE_ATTRIBUTE_HIDDEN|windows.FILE_ATTRIBUTE_SYSTEM) != 0
}

func sizeOf(ctx context.Context, root string, deadline time.Time) (bytes int64, files int, partial bool) {
	n := 0
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if n++; n%256 == 0 && (time.Now().After(deadline) || ctx.Err() != nil) {
			partial = true
			return fs.SkipAll
		}
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != root && d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				return fs.SkipDir
			}
			return nil
		}
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() {
			bytes += info.Size()
			files++
		}
		return nil
	})
	return
}
