// Package handover lists what to deal with before a PC is sold, given away or
// handed back: accounts still signed in, files that have not been copied off,
// keys and tokens that keep working after the PC is gone, and how to reset it.
// It only reads; nothing is changed.
package handover

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Item struct {
	ID     string `json:"id"`
	Group  string `json:"group"`  // Accounts | Your files | Secrets | Device
	Title  string `json:"title"`  // what was found, in a few words
	Status string `json:"status"` // todo | ok | info
	Detail string `json:"detail"`
	How    string `json:"how"` // what to do about it
}

type FolderSize struct {
	Label   string `json:"label"`
	Path    string `json:"path"`
	Kind    string `json:"kind"` // personal | other (outside the user folder)
	Bytes   int64  `json:"bytes"`
	Files   int    `json:"files"`
	Partial bool   `json:"partial"` // sizing ran out of time: "at least"
}

type Report struct {
	Items      []Item       `json:"items"`
	Folders    []FolderSize `json:"folders"`
	Todo       int          `json:"todo"` // items marked todo
	DurationMs int64        `json:"duration_ms"`
}

// --- what was found, gathered by the platform code ---

type joinState struct {
	known     bool
	azure     bool
	domain    bool
	workplace bool
	org       string // tenant or domain name
}

type browser struct {
	name     string
	accounts []string // signed-in accounts (emails)
	logins   int      // saved passwords, when the browser lets that be counted without reading them; -1 unknown
}

type facts struct {
	msAccounts []string
	join       joinState
	oneDrive   []string // signed-in OneDrive accounts
	redirected []string // personal folders kept in OneDrive
	browsers   []browser
	wifi       int // -1 unknown
	creds      int // -1 unknown
	secrets    []string
	folders    []FolderSize
	bitlocker  *int   // System.Volume.BitLockerProtection; nil when unknown
	licence    string // oem | retail | volume | ""
}

// --- parsing ---

// maskEmail keeps enough to recognise an account without showing it in full:
// anna.k@gmail.com -> a***@gmail.com
func maskEmail(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	user, domain, ok := strings.Cut(s, "@")
	if !ok {
		r := []rune(s)
		return string(r[0]) + "***"
	}
	if user == "" {
		return "***@" + domain
	}
	return string([]rune(user)[0]) + "***@" + domain
}

func maskAll(list []string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, maskEmail(s))
	}
	return out
}

// uniqueFold drops empties and repeats, ignoring case
func uniqueFold(list []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range list {
		s = strings.TrimSpace(s)
		k := strings.ToLower(s)
		if s == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, s)
	}
	return out
}

// parseDsreg reads `dsregcmd /status`: lines like "   AzureAdJoined : YES"
func parseDsreg(out string) joinState {
	j := joinState{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		yes := strings.EqualFold(v, "YES")
		switch k {
		case "AzureAdJoined":
			j.known, j.azure = true, yes
		case "DomainJoined":
			j.known, j.domain = true, yes
		case "WorkplaceJoined":
			j.known, j.workplace = true, yes
		case "TenantName":
			if v != "" && j.org == "" {
				j.org = v
			}
		case "DomainName":
			if v != "" {
				j.org = v
			}
		}
	}
	return j
}

// countWifiProfiles reads `netsh wlan show profiles`; -1 when it is not that output
func countWifiProfiles(out string) (int, []string) {
	if !strings.Contains(out, "Profile") {
		return -1, nil
	}
	n := 0
	var names []string
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok || !strings.Contains(k, "User Profile") {
			continue
		}
		n++
		if v = strings.TrimSpace(v); v != "" {
			names = append(names, v)
		}
	}
	return n, names
}

// countCredentials reads `cmdkey /list`, leaving out the entries Windows keeps
// for itself for the Microsoft account
func countCredentials(out string) int {
	if !strings.Contains(out, "Target") && !strings.Contains(out, "NONE") {
		return -1
	}
	n := 0
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(k) != "Target" {
			continue
		}
		if strings.Contains(strings.ToLower(v), "virtualapp/didlogical") {
			continue
		}
		n++
	}
	return n
}

// chromiumAccounts reads a Chromium browser's "Local State" for its signed-in profiles
func chromiumAccounts(data []byte) []string {
	var state struct {
		Profile struct {
			InfoCache map[string]struct {
				UserName string `json:"user_name"`
			} `json:"info_cache"`
		} `json:"profile"`
	}
	if json.Unmarshal(data, &state) != nil {
		return nil
	}
	var names []string
	for _, p := range state.Profile.InfoCache {
		names = append(names, p.UserName)
	}
	sort.Strings(names)
	return uniqueFold(names)
}

// firefoxProfiles reads profiles.ini: the profile folders, relative or absolute
func firefoxProfiles(ini string) (paths []string, relative []bool) {
	rel := true
	path := ""
	flush := func() {
		if path != "" {
			paths = append(paths, path)
			relative = append(relative, rel)
		}
		path, rel = "", true
	}
	for _, line := range strings.Split(ini, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			flush()
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "Path":
			path = v
		case "IsRelative":
			rel = v != "0"
		}
	}
	flush()
	return
}

// firefoxAccount reads signedInUser.json for the Firefox account email
func firefoxAccount(data []byte) string {
	var s struct {
		AccountData struct {
			Email string `json:"email"`
		} `json:"accountData"`
	}
	if json.Unmarshal(data, &s) != nil {
		return ""
	}
	return s.AccountData.Email
}

// firefoxLogins counts the saved passwords in logins.json; the passwords
// themselves are encrypted and are not looked at
func firefoxLogins(data []byte) int {
	var s struct {
		Logins []json.RawMessage `json:"logins"`
	}
	if json.Unmarshal(data, &s) != nil {
		return 0
	}
	return len(s.Logins)
}

// licenceChannel reads the Description of Windows' SoftwareLicensingProduct
func licenceChannel(description string) string {
	d := strings.ToUpper(description)
	switch {
	case strings.Contains(d, "OEM"):
		return "oem"
	case strings.Contains(d, "RETAIL"):
		return "retail"
	case strings.Contains(d, "VOLUME"):
		return "volume"
	}
	return ""
}

// top-level folders that belong to Windows or programs, not to the person
var systemRoots = map[string]bool{
	"windows": true, "program files": true, "program files (x86)": true, "programdata": true,
	"$recycle.bin": true, "system volume information": true, "recovery": true, "perflogs": true,
	"users": true, "$windows.~bt": true, "$windows.~ws": true, "$winreagent": true, "$sysreset": true,
	"windows.old": true, "esd": true, "intel": true, "amd": true, "nvidia": true, "drivers": true,
	"inetpub": true, "onedrivetemp": true, "xboxgames": true, "msocache": true, "config.msi": true,
	"documents and settings": true, "boot": true, "efi": true, "swsetup": true, "temp": true,
	"tmp": true, "$getcurrent": true, "dumpstack.log.tmp": true, "windowsapps": true,
	"program files (arm)": true,
}

func isSystemRoot(name string) bool {
	return systemRoots[strings.ToLower(name)] || strings.HasPrefix(name, "$")
}

// --- the checklist ---

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func listOf(items []string, max int) string {
	if len(items) <= max {
		return joinAnd(items)
	}
	return strings.Join(items[:max], ", ") + fmt.Sprintf(" and %d more", len(items)-max)
}

func joinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// build turns what was found into the checklist, in the order it is best done
func build(f facts) Report {
	r := Report{Items: []Item{}, Folders: f.folders}
	if r.Folders == nil {
		r.Folders = []FolderSize{}
	}
	add := func(it Item) { r.Items = append(r.Items, it) }
	orgOwned := f.join.azure || f.join.domain

	// Accounts
	if orgOwned {
		where := "an organisation"
		if f.join.org != "" {
			where = f.join.org
		}
		add(Item{ID: "organisation", Group: "Accounts", Status: "info",
			Title:  "This PC belongs to " + where,
			Detail: "It is joined to a work or school network, so the organisation manages it and may still own it.",
			How:    "Hand it back through your IT team rather than resetting it yourself. Copy off your own files first."})
	} else if f.join.workplace {
		add(Item{ID: "work-account", Group: "Accounts", Status: "todo",
			Title:  "A work or school account is added",
			Detail: "Your organisation's account can still reach its email and files from this PC.",
			How:    "Settings > Accounts > Access work or school, choose the account and Disconnect."})
	}

	if accounts := uniqueFold(f.msAccounts); len(accounts) > 0 {
		add(Item{ID: "microsoft-account", Group: "Accounts", Status: "todo",
			Title: "Microsoft account: " + joinAnd(maskAll(accounts)),
			// this list holds accounts saved by Windows and by Microsoft apps (Office, the Store), so it does not claim which
			Detail: "This Microsoft account is saved on the PC, for signing in to Windows or to Microsoft apps such as Office and the Store.",
			How:    "Sign out of Microsoft apps, and Reset this PC removes the account. If you signed in to Windows with it, also remove the PC from account.microsoft.com/devices."})
	} else {
		add(Item{ID: "microsoft-account", Group: "Accounts", Status: "ok",
			Title:  "No Microsoft account saved",
			Detail: "Neither Windows nor Microsoft apps have a Microsoft account remembered for you."})
	}

	if accounts := uniqueFold(f.oneDrive); len(accounts) > 0 {
		detail := "OneDrive is signed in."
		if len(f.redirected) > 0 {
			detail = fmt.Sprintf("OneDrive is signed in, and your %s %s kept in it, so %s already in the cloud once syncing has finished.",
				joinAnd(f.redirected), map[bool]string{true: "is", false: "are"}[len(f.redirected) == 1],
				map[bool]string{true: "it is", false: "they are"}[len(f.redirected) == 1])
		}
		add(Item{ID: "onedrive", Group: "Accounts", Status: "todo",
			Title:  "OneDrive: " + joinAnd(maskAll(accounts)),
			Detail: detail,
			How:    "Check the OneDrive icon shows everything is up to date, then OneDrive settings > Account > Unlink this PC."})
	}

	for _, b := range f.browsers {
		accounts := uniqueFold(b.accounts)
		saved := ""
		if b.logins > 0 {
			saved = fmt.Sprintf(" It has %s saved.", plural(b.logins, "password", "passwords"))
		}
		if len(accounts) > 0 {
			add(Item{ID: "browser-" + strings.ToLower(b.name), Group: "Accounts", Status: "todo",
				Title:  b.name + " is signed in as " + joinAnd(maskAll(accounts)),
				Detail: "Bookmarks, passwords and history sync to that account." + saved,
				How:    "Make sure sync is on so they reach your next PC, then sign out in " + b.name + "'s settings."})
		} else {
			add(Item{ID: "browser-" + strings.ToLower(b.name), Group: "Accounts", Status: "info",
				Title:  b.name + " is not signed in",
				Detail: "Its bookmarks, saved passwords and history are only on this PC." + saved,
				How:    "If you want them, export bookmarks and passwords from " + b.name + "'s settings, or sign in to sync them."})
		}
	}

	// Your files
	var personal, other []FolderSize
	for _, fs := range f.folders {
		if fs.Kind == "other" {
			other = append(other, fs)
		} else {
			personal = append(personal, fs)
		}
	}
	if len(personal) > 0 {
		var total int64
		partial := false
		var parts []string
		for _, p := range personal {
			total += p.Bytes
			partial = partial || p.Partial
			if p.Bytes > 0 {
				parts = append(parts, fmt.Sprintf("%s %s", p.Label, formatBytes(p.Bytes)))
			}
		}
		size := formatBytes(total)
		if partial {
			size = "at least " + size
		}
		it := Item{ID: "personal-files", Group: "Your files", Status: "todo",
			Title:  "Your folders hold " + size,
			Detail: listOf(parts, 6) + ".",
			How:    "Copy what you want to keep to an external drive or cloud storage. Reset this PC erases them."}
		if total == 0 {
			it.Status, it.Title, it.Detail, it.How = "ok", "Your folders are empty", "Documents, Desktop, Pictures and the other personal folders hold nothing.", ""
		}
		if len(f.redirected) > 0 && total > 0 {
			it.How += " Folders kept in OneDrive are already in the cloud."
		}
		add(it)
	}
	if len(other) > 0 {
		var names []string
		var total int64
		partial := false
		for _, o := range other {
			names = append(names, o.Path)
			total += o.Bytes
			partial = partial || o.Partial
		}
		size := formatBytes(total)
		if partial {
			size = "at least " + size
		}
		add(Item{ID: "other-folders", Group: "Your files", Status: "todo",
			Title:  fmt.Sprintf("%s outside your user folder (%s)", plural(len(other), "folder", "folders"), size),
			Detail: "Folders you or your programs made on the drives: " + listOf(names, 4) + ". Some may be installed programs.",
			How:    "Look through them and copy what is yours. When resetting, choose to clean all drives, or these stay on the other drives."})
	}

	// Secrets
	if len(f.secrets) > 0 {
		add(Item{ID: "dev-secrets", Group: "Secrets", Status: "todo",
			Title:  plural(len(f.secrets), "developer key or token", "developer keys and tokens") + " on this PC",
			Detail: "Found: " + listOf(f.secrets, 5) + ". Deleting the files does not stop them working: anyone with a copy can still use them.",
			How:    "Keep the ones you need in a password manager, then revoke the rest at GitHub, AWS, npm and the like, and remove the PC's SSH key from the servers and sites that trust it."})
	}
	if f.creds > 0 {
		add(Item{ID: "credentials", Group: "Secrets", Status: "info",
			Title:  plural(f.creds, "saved sign-in", "saved sign-ins") + " in Credential Manager",
			Detail: "Windows remembers passwords for network shares, Git, remote desktops and apps.",
			How:    "Reset this PC removes them. To remove them one by one: Control Panel > Credential Manager."})
	}
	if f.wifi > 0 {
		add(Item{ID: "wifi", Group: "Secrets", Status: "info",
			Title:  plural(f.wifi, "saved Wi-Fi network", "saved Wi-Fi networks"),
			Detail: "Their passwords are stored on this PC, including your home network's.",
			How:    "Reset this PC removes them. To forget one: Settings > Network & internet > Wi-Fi > Manage known networks."})
	}

	// Device
	if f.bitlocker != nil {
		// System.Volume.BitLockerProtection: 1 on, 2 off, 3 encrypting, 4 decrypting,
		// 5 suspended, 6 on (locked), 8 waiting for activation
		switch *f.bitlocker {
		case 1, 3, 6:
			add(Item{ID: "encryption", Group: "Device", Status: "ok",
				Title:  "The system drive is encrypted",
				Detail: "BitLocker protects the drive, so files cannot be read by taking the drive out.",
				How:    "When resetting, still turn on \"Clean data\" if you are selling the PC."})
		case 5:
			// suspended: the key is stored unprotected on the drive, so it does not protect anything
			add(Item{ID: "encryption", Group: "Device", Status: "todo",
				Title:  "Drive encryption is paused",
				Detail: "BitLocker is suspended, so its key is stored unprotected on the drive and the files can be read by anyone who takes the drive out.",
				How:    "Resume BitLocker before handing the PC over, and turn on \"Clean data\" when resetting."})
		case 2, 4, 8:
			add(Item{ID: "encryption", Group: "Device", Status: "info",
				Title:  "The system drive is not encrypted",
				Detail: "A plain reset leaves deleted files recoverable with the right tools.",
				How:    "When resetting, turn on \"Clean data\": it takes longer, but deleted files cannot be brought back."})
		}
	}
	switch f.licence {
	case "oem":
		add(Item{ID: "licence", Group: "Device", Status: "ok",
			Title:  "Windows licence stays with this PC",
			Detail: "It came with the PC (OEM), so the next owner gets a working Windows after the reset."})
	case "retail":
		add(Item{ID: "licence", Group: "Device", Status: "info",
			Title:  "Windows licence was bought separately",
			Detail: "A retail licence can move to another PC, but only one at a time.",
			How:    "To take it with you, keep the product key or link it to your Microsoft account (Settings > System > Activation). The next owner then needs their own."})
	case "volume":
		add(Item{ID: "licence", Group: "Device", Status: "info",
			Title:  "Windows is licensed by an organisation",
			Detail: "A volume licence belongs to a company or school, not to the PC.",
			How:    "Check with the organisation before passing the PC on: Windows may not stay activated."})
	}

	reset := Item{ID: "reset", Group: "Device", Status: "info",
		Title:  "Last step: reset this PC",
		Detail: "This removes your accounts, apps, files and settings, and leaves a fresh Windows for the next owner.",
		How:    "Settings > System > Recovery > Reset this PC > Remove everything. When selling, open \"Change settings\" and turn on \"Clean data\"."}
	if orgOwned {
		reset.How = "Your IT team usually does this. If they ask you to: " + reset.How
	}
	add(reset)

	for _, it := range r.Items {
		if it.Status == "todo" {
			r.Todo++
		}
	}
	return r
}
