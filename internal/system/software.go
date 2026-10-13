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
	parseFunc        func(data []byte) (string, string)
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

	for _, item := range cliApps {
		found := false
		for k, regItem := range mergedMap {
			if strings.Contains(k, strings.ToLower(item.Name)) || strings.Contains(strings.ToLower(item.Name), k) {
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

func scanCLITools() []SoftwareItem {
	targets := []apiTarget{
		{
			name:             "Node.js",
			cmd:              "node",
			cmdArgs:          []string{"-v"},
			versionRegex:     `v?(\d+\.\d+\.\d+)`,
			apiURL:           "https://nodejs.org/dist/index.json",
			officialDownload: "https://nodejs.org",
			parseFunc: func(data []byte) (string, string) {
				var list []struct {
					Version string `json:"version"`
					Date    string `json:"date"`
				}
				if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 {
					return strings.TrimPrefix(list[0].Version, "v"), list[0].Date
				}
				return "20.11.0", "2024-01-18"
			},
		},
		{
			name:             "Git",
			cmd:              "git",
			cmdArgs:          []string{"--version"},
			versionRegex:     `(\d+\.\d+\.\d+)`,
			apiURL:           "https://api.github.com/repos/git-for-windows/git/releases/latest",
			officialDownload: "https://git-scm.com",
			parseFunc:        parseGitHubRelease,
		},
		{
			name:             "Python",
			cmd:              "python",
			cmdArgs:          []string{"--version"},
			versionRegex:     `(\d+\.\d+\.\d+)`,
			apiURL:           "https://endoflife.date/api/python.json",
			officialDownload: "https://www.python.org",
			parseFunc: func(data []byte) (string, string) {
				var list []struct {
					Latest            string `json:"latest"`
					LatestReleaseDate string `json:"latestReleaseDate"`
				}
				if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 {
					return list[0].Latest, list[0].LatestReleaseDate
				}
				return "3.12.1", "2023-12-08"
			},
		},
		{
			name:             "Go",
			cmd:              "go",
			cmdArgs:          []string{"version"},
			versionRegex:     `go(\d+\.\d+\.\d+)`,
			apiURL:           "https://go.dev/dl/?mode=json",
			officialDownload: "https://go.dev",
			parseFunc: func(data []byte) (string, string) {
				var list []struct {
					Version string `json:"version"`
				}
				if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 {
					ver := strings.TrimPrefix(list[0].Version, "go")
					return ver, time.Now().Format("2006-01-02")
				}
				return "1.22.0", "2024-02-06"
			},
		},
		{
			name:             "Visual Studio Code",
			cmd:              "code",
			cmdArgs:          []string{"--version"},
			versionRegex:     `(\d+\.\d+\.\d+)`,
			apiURL:           "https://api.github.com/repos/microsoft/vscode/releases/latest",
			officialDownload: "https://code.visualstudio.com",
			parseFunc:        parseGitHubRelease,
		},
		{
			name:             "Rust",
			cmd:              "rustc",
			cmdArgs:          []string{"--version"},
			versionRegex:     `(\d+\.\d+\.\d+)`,
			apiURL:           "https://api.github.com/repos/rust-lang/rust/releases/latest",
			officialDownload: "https://www.rust-lang.org",
			parseFunc:        parseGitHubRelease,
		},
		{
			name:             "PowerShell",
			cmd:              "powershell",
			cmdArgs:          []string{"$PSVersionTable.PSVersion.ToString()"},
			versionRegex:     `(\d+\.\d+\.\d+)`,
			apiURL:           "https://api.github.com/repos/PowerShell/PowerShell/releases/latest",
			officialDownload: "https://github.com/PowerShell/PowerShell",
			parseFunc:        parseGitHubRelease,
		},
	}

	httpClient := &http.Client{Timeout: 2 * time.Second}
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

			latestVer, releaseDate := fetchLatestVersion(httpClient, t.apiURL, t.parseFunc)
			isOutdated := isVersionOlder(currentVer, latestVer)

			mu.Lock()
			results = append(results, SoftwareItem{
				Name:             t.name,
				CurrentVersion:   currentVer,
				LatestVersion:    latestVer,
				IsOutdated:       isOutdated,
				ReleaseDate:      releaseDate,
				OfficialDownload: t.officialDownload,
			})
			mu.Unlock()
		}(target)
	}

	wg.Wait()
	return results
}

func fetchLatestVersion(client *http.Client, url string, parseFn func([]byte) (string, string)) (string, string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "Vendor Managed", "Recent"
	}
	req.Header.Set("User-Agent", "SysInfoPro-Diagnostics")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "Vendor Managed", "Recent"
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "Vendor Managed", "Recent"
	}

	return parseFn(buf.Bytes())
}

func parseGitHubRelease(data []byte) (string, string) {
	var release struct {
		TagName     string `json:"tag_name"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.Unmarshal(data, &release); err == nil && release.TagName != "" {
		re := regexp.MustCompile(`(\d+\.\d+\.\d+)`)
		match := re.FindString(release.TagName)
		if match == "" {
			match = release.TagName
		}

		dateStr := "Recent"
		if len(release.PublishedAt) >= 10 {
			dateStr = release.PublishedAt[:10]
		}
		return match, dateStr
	}
	return "Vendor Managed", "Recent"
}

func checkCommandVersion(cmdName string, args []string, regexPattern string) string {
	cmd := exec.Command(cmdName, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW (0x08000000)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return ""
	}

	output := strings.TrimSpace(out.String())
	re := regexp.MustCompile(regexPattern)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return output
}

func isVersionOlder(current string, latest string) bool {
	cleanCur := strings.TrimPrefix(strings.TrimSpace(current), "v")
	cleanLat := strings.TrimPrefix(strings.TrimSpace(latest), "v")

	if cleanCur == cleanLat || latest == "Vendor Managed" || latest == "Latest" {
		return false
	}

	curParts := strings.Split(cleanCur, ".")
	latParts := strings.Split(cleanLat, ".")

	for i := 0; i < len(curParts) && i < len(latParts); i++ {
		if curParts[i] != latParts[i] {
			return curParts[i] < latParts[i]
		}
	}
	return len(curParts) < len(latParts)
}
