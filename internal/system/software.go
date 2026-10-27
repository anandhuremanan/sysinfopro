package system

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

type SoftwareItem struct {
	Name             string `json:"name"`
	CurrentVersion   string `json:"current_version"`
	LatestVersion    string `json:"latest_version"`
	IsOutdated       bool   `json:"is_outdated"`
	ReleaseDate      string `json:"release_date"`
	OfficialDownload string `json:"official_download"`
}

type apiTarget struct {
	name             string
	cmd              string
	cmdArgs          []string
	versionRegex     string
	apiURL           string
	officialDownload string
	// pick reads the release list and returns the version to compare with and
	// its release date; "" for either when it cannot tell
	pick func(data []byte, current string) (latest, date string)
}

func GetSoftwareHealthDetails() ([]SoftwareItem, error) {
	// 1. Programmatically scan ALL installed applications from Windows Registry
	registryApps := scanRegistryInstalledApps()

	// 2. Fetch real-time latest release versions for CLI & core developer tools in parallel
	cliApps := scanCLITools()

	// 3. Merge registry apps and CLI apps, prioritizing live API checks
	mergedMap := make(map[string]SoftwareItem)

	for _, item := range registryApps {
		mergedMap[strings.ToLower(item.Name)] = item
	}

	keys := make([]string, 0, len(mergedMap))
	for k := range mergedMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, item := range cliApps {
		found := false
		for _, k := range keys {
			regItem := mergedMap[k]
			if sameProduct(k, item.Name) {
				regItem.CurrentVersion = item.CurrentVersion
				regItem.LatestVersion = item.LatestVersion
				regItem.IsOutdated = item.IsOutdated
				regItem.ReleaseDate = item.ReleaseDate
				regItem.OfficialDownload = item.OfficialDownload
				mergedMap[k] = regItem
				found = true
				break
			}
		}
		if !found {
			mergedMap[strings.ToLower(item.Name)] = item
		}
	}

	results := []SoftwareItem{}
	for _, item := range mergedMap {
		results = append(results, item)
	}

	// Sort alphabetically by app name
	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	return results, nil
}

// InstalledApps returns installed application names with their versions, read from
// the registry only (no network), so it is quick enough for before/after comparisons.
func InstalledApps() map[string]string {
	apps := map[string]string{}
	for _, item := range scanRegistryInstalledApps() {
		apps[item.Name] = item.CurrentVersion
	}
	return apps
}

func scanRegistryInstalledApps() []SoftwareItem {
	appMap := make(map[string]SoftwareItem)

	keys := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	for _, k := range keys {
		key, err := registry.OpenKey(k.root, k.path, registry.READ)
		if err != nil {
			continue
		}

		subkeys, err := key.ReadSubKeyNames(-1)
		key.Close()
		if err != nil {
			continue
		}

		for _, sub := range subkeys {
			subKey, err := registry.OpenKey(k.root, k.path+`\`+sub, registry.READ)
			if err != nil {
				continue
			}

			name, _, _ := subKey.GetStringValue("DisplayName")
			version, _, _ := subKey.GetStringValue("DisplayVersion")
			url, _, _ := subKey.GetStringValue("URLInfoAbout")
			systemComponent, _, _ := subKey.GetIntegerValue("SystemComponent")
			parentKeyName, _, _ := subKey.GetStringValue("ParentKeyName")

			subKey.Close()

			nameClean := strings.TrimSpace(name)
			if nameClean == "" || systemComponent == 1 || parentKeyName != "" {
				continue
			}

			// Filter out Windows updates, KB patches, and Redistributables noise
			if strings.HasPrefix(nameClean, "KB") ||
				strings.Contains(nameClean, "Security Update") ||
				strings.Contains(nameClean, "Update for Microsoft") ||
				strings.Contains(nameClean, "Language Pack") {
				continue
			}

			keyLower := strings.ToLower(nameClean)
			if _, exists := appMap[keyLower]; !exists {
				if version == "" {
					version = "Detected"
				}

				if url == "" {
					url = "https://www.google.com/search?q=" + strings.ReplaceAll(nameClean, " ", "+") + "+official+download"
				}

				appMap[keyLower] = SoftwareItem{
					Name:             nameClean,
					CurrentVersion:   version,
					LatestVersion:    "Vendor Managed",
					IsOutdated:       false,
					ReleaseDate:      "Installed",
					OfficialDownload: url,
				}
			}
		}
	}

	results := []SoftwareItem{}
	for _, item := range appMap {
		results = append(results, item)
	}
	return results
}

// sameProduct reports whether an installed program's name is the tool's: the
// tool's name as a whole word ("Go Programming Language" is Go, "Google Chrome"
// is not)
func sameProduct(installed, tool string) bool {
	re := regexp.MustCompile(`(^|[^a-z0-9])` + regexp.QuoteMeta(strings.ToLower(tool)) + `($|[^a-z0-9])`)
	return re.MatchString(strings.ToLower(installed))
}

// majorMinor turns "3.13.7" into "3.13"; majorOf turns it into "3"
func majorMinor(v string) string {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return v
	}
	return parts[0] + "." + parts[1]
}

func majorOf(v string) string { return strings.SplitN(v, ".", 2)[0] }

// pickNode compares with the newest release of the same major version, so
// someone on a supported LTS line is not told to jump to the newest major
func pickNode(data []byte, current string) (string, string) {
	var list []struct {
		Version string `json:"version"`
		Date    string `json:"date"`
	}
	if json.Unmarshal(data, &list) != nil || len(list) == 0 {
		return "", ""
	}
	for _, r := range list { // newest first
		v := strings.TrimPrefix(r.Version, "v")
		if majorOf(v) == majorOf(current) {
			return v, r.Date
		}
	}
	return strings.TrimPrefix(list[0].Version, "v"), list[0].Date
}

// pickPython compares with the newest release of the same 3.x line, or the
// newest line when the user's line is no longer listed
func pickPython(data []byte, current string) (string, string) {
	var list []struct {
		Cycle             string `json:"cycle"`
		Latest            string `json:"latest"`
		LatestReleaseDate string `json:"latestReleaseDate"`
	}
	if json.Unmarshal(data, &list) != nil || len(list) == 0 {
		return "", ""
	}
	for _, r := range list {
		if r.Cycle == majorMinor(current) {
			return r.Latest, r.LatestReleaseDate
		}
	}
	return list[0].Latest, list[0].LatestReleaseDate
}

// pickGo compares with the newest release of the same 1.x line; go.dev lists the
// two supported lines, and an older one is compared with the newest
func pickGo(data []byte, current string) (string, string) {
	var list []struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &list) != nil || len(list) == 0 {
		return "", ""
	}
	for _, r := range list {
		v := strings.TrimPrefix(r.Version, "go")
		if majorMinor(v) == majorMinor(current) {
			return v, "" // go.dev does not give release dates
		}
	}
	return strings.TrimPrefix(list[0].Version, "go"), ""
}

func pickGitHub(data []byte, _ string) (string, string) { return parseGitHubRelease(data) }

func scanCLITools() []SoftwareItem {
	targets := []apiTarget{
		{name: "Node.js", cmd: "node", cmdArgs: []string{"-v"}, versionRegex: `v?(\d+\.\d+\.\d+)`,
			apiURL: "https://nodejs.org/dist/index.json", officialDownload: "https://nodejs.org", pick: pickNode},
		{name: "Git", cmd: "git", cmdArgs: []string{"--version"}, versionRegex: `(\d+\.\d+\.\d+)`,
			apiURL: "https://api.github.com/repos/git-for-windows/git/releases/latest", officialDownload: "https://git-scm.com", pick: pickGitHub},
		{name: "Python", cmd: "python", cmdArgs: []string{"--version"}, versionRegex: `(\d+\.\d+\.\d+)`,
			apiURL: "https://endoflife.date/api/python.json", officialDownload: "https://www.python.org", pick: pickPython},
		{name: "Go", cmd: "go", cmdArgs: []string{"version"}, versionRegex: `go(\d+\.\d+\.\d+)`,
			apiURL: "https://go.dev/dl/?mode=json", officialDownload: "https://go.dev", pick: pickGo},
		{name: "Visual Studio Code", cmd: "code", cmdArgs: []string{"--version"}, versionRegex: `(\d+\.\d+\.\d+)`,
			apiURL: "https://api.github.com/repos/microsoft/vscode/releases/latest", officialDownload: "https://code.visualstudio.com", pick: pickGitHub},
		{name: "Rust", cmd: "rustc", cmdArgs: []string{"--version"}, versionRegex: `(\d+\.\d+\.\d+)`,
			apiURL: "https://api.github.com/repos/rust-lang/rust/releases/latest", officialDownload: "https://www.rust-lang.org", pick: pickGitHub},
		// PowerShell 7 (pwsh). Windows PowerShell 5.1, built into Windows, is a
		// different product with no newer version, so it is not compared with 7.x
		{name: "PowerShell", cmd: "pwsh", cmdArgs: []string{"--version"}, versionRegex: `(\d+\.\d+\.\d+)`,
			apiURL: "https://api.github.com/repos/PowerShell/PowerShell/releases/latest", officialDownload: "https://github.com/PowerShell/PowerShell", pick: pickGitHub},
	}

	httpClient := &http.Client{Timeout: 4 * time.Second}
	results := []SoftwareItem{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(t apiTarget) {
			defer wg.Done()
			currentVer := checkCommandVersion(t.cmd, t.cmdArgs, t.versionRegex)
			if currentVer == "" {
				return
			}
			latestVer, releaseDate := fetchLatestVersion(httpClient, t.apiURL, func(data []byte) (string, string) {
				return t.pick(data, currentVer)
			})

			mu.Lock()
			results = append(results, SoftwareItem{
				Name:             t.name,
				CurrentVersion:   currentVer,
				LatestVersion:    latestVer, // "" when it could not be looked up
				IsOutdated:       isVersionOlder(currentVer, latestVer),
				ReleaseDate:      releaseDate,
				OfficialDownload: t.officialDownload,
			})
			mu.Unlock()
		}(target)
	}

	wg.Wait()
	return results
}

// fetchLatestVersion asks the publisher; ("", "") when that fails, never a guess
func fetchLatestVersion(client *http.Client, url string, parseFn func([]byte) (string, string)) (string, string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", "SysInfoPro-Diagnostics")

	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", ""
	}
	return parseFn(buf.Bytes())
}

func parseGitHubRelease(data []byte) (string, string) {
	var release struct {
		TagName     string `json:"tag_name"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.Unmarshal(data, &release); err != nil || release.TagName == "" {
		return "", ""
	}
	match := regexp.MustCompile(`(\d+\.\d+\.\d+)`).FindString(release.TagName)
	if match == "" {
		return "", "" // a tag that is not a version number cannot be compared
	}
	date := ""
	if len(release.PublishedAt) >= 10 {
		date = release.PublishedAt[:10]
	}
	return match, date
}

func checkCommandVersion(cmdName string, args []string, regexPattern string) string {
	cmd := exec.Command(cmdName, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}

	matches := regexp.MustCompile(regexPattern).FindStringSubmatch(strings.TrimSpace(out.String()))
	if len(matches) > 1 {
		return matches[1]
	}
	return "" // no version number in the output: unknown, not the raw text
}

// compareVersions compares dotted versions number by number (so 1.22.10 is newer
// than 1.22.9, and 10.0 newer than 9.4); missing parts count as 0
func compareVersions(a, b string) int {
	num := func(s string) int {
		n := 0
		for _, r := range s {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		return n
	}
	ap := strings.Split(strings.TrimPrefix(strings.TrimSpace(a), "v"), ".")
	bp := strings.Split(strings.TrimPrefix(strings.TrimSpace(b), "v"), ".")
	for i := 0; i < len(ap) || i < len(bp); i++ {
		var x, y int
		if i < len(ap) {
			x = num(ap[i])
		}
		if i < len(bp) {
			y = num(bp[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// isVersionOlder is true only when the latest version is known and newer
func isVersionOlder(current, latest string) bool {
	if current == "" || latest == "" || latest == "Vendor Managed" || latest == "Latest" {
		return false
	}
	return compareVersions(current, latest) < 0
}
