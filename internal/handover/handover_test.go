package handover

import (
	"strings"
	"testing"
)

func TestMaskEmail(t *testing.T) {
	cases := map[string]string{
		"anna.k@gmail.com": "a***@gmail.com",
		"  b@corp.com ":    "b***@corp.com",
		"Person 1":         "P***",
		"":                 "",
	}
	for in, want := range cases {
		if got := maskEmail(in); got != want {
			t.Errorf("maskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseDsreg(t *testing.T) {
	out := `
+----------------------------------------------------------------------+
| Device State                                                         |
+----------------------------------------------------------------------+

             AzureAdJoined : YES
          EnterpriseJoined : NO
              DomainJoined : NO
               Device Name : PC-01

+----------------------------------------------------------------------+
| Tenant Details                                                       |
+----------------------------------------------------------------------+

                TenantName : Contoso Ltd
                  TenantId : 0000

            WorkplaceJoined : NO
`
	j := parseDsreg(out)
	if !j.known || !j.azure || j.domain || j.workplace || j.org != "Contoso Ltd" {
		t.Errorf("got %+v", j)
	}
	if parseDsreg("garbage").known {
		t.Error("unknown output must stay unknown")
	}
}

func TestCountWifiAndCredentials(t *testing.T) {
	wifi := "Profiles on interface Wi-Fi:\r\n\r\nGroup policy profiles (read only)\r\n---------------------------------\r\n    <None>\r\n\r\nUser profiles\r\n-------------\r\n    All User Profile     : HomeNet\r\n    All User Profile     : Cafe Free\r\n"
	if n, names := countWifiProfiles(wifi); n != 2 || names[1] != "Cafe Free" {
		t.Errorf("wifi = %d %v", n, names)
	}
	if n, _ := countWifiProfiles("The Wireless AutoConfig Service (wlansvc) is not running."); n != -1 {
		t.Errorf("no Wi-Fi service should be unknown, got %d", n)
	}

	creds := `
Currently stored credentials:

    Target: WindowsLive:target=virtualapp/didlogical
    Type: Generic
    User: 02abc
    Local machine persistence

    Target: LegacyGeneric:target=git:https://github.com
    Type: Generic
    User: someone

    Target: Domain:target=fileserver
    Type: Domain Password
`
	if n := countCredentials(creds); n != 2 {
		t.Errorf("creds = %d, want 2 (the Microsoft account's own entry left out)", n)
	}
	if n := countCredentials("\nCurrently stored credentials:\n\n* NONE *\n"); n != 0 {
		t.Errorf("none = %d", n)
	}
}

func TestBrowserParsing(t *testing.T) {
	state := `{"profile":{"info_cache":{"Default":{"user_name":"me@gmail.com","name":"Me"},"Profile 1":{"user_name":""},"Profile 2":{"user_name":"ME@gmail.com"}}}}`
	if got := chromiumAccounts([]byte(state)); len(got) != 1 || got[0] != "ME@gmail.com" && got[0] != "me@gmail.com" {
		t.Errorf("chromium = %v", got)
	}
	if chromiumAccounts([]byte("not json")) != nil {
		t.Error("bad json")
	}

	ini := "[General]\nStartWithLastProfile=1\n\n[Profile0]\nName=default-release\nIsRelative=1\nPath=Profiles/abc.default-release\n\n[Profile1]\nName=other\nIsRelative=0\nPath=D:\\FF\\other\n\n[Install123]\nDefault=Profiles/abc.default-release\n"
	paths, rel := firefoxProfiles(ini)
	if len(paths) != 2 || !rel[0] || rel[1] || paths[1] != `D:\FF\other` {
		t.Errorf("firefox profiles = %v %v", paths, rel)
	}
	if got := firefoxAccount([]byte(`{"version":1,"accountData":{"email":"fox@example.org","uid":"x"}}`)); got != "fox@example.org" {
		t.Errorf("firefox account = %q", got)
	}
	if n := firefoxLogins([]byte(`{"nextId":3,"logins":[{"id":1},{"id":2}]}`)); n != 2 {
		t.Errorf("firefox logins = %d", n)
	}
}

func TestLicenceAndRoots(t *testing.T) {
	cases := map[string]string{
		"Windows(R) Operating System, OEM_DM channel":     "oem",
		"Windows(R) Operating System, RETAIL channel":     "retail",
		"Windows(R) Operating System, VOLUME_MAK channel": "volume",
		"": "",
	}
	for in, want := range cases {
		if got := licenceChannel(in); got != want {
			t.Errorf("licenceChannel(%q) = %q", in, got)
		}
	}
	for _, n := range []string{"Windows", "Program Files (x86)", "$Recycle.Bin", "$WinREAgent", "Users"} {
		if !isSystemRoot(n) {
			t.Errorf("%s should be skipped", n)
		}
	}
	for _, n := range []string{"Projects", "Photos", "work"} {
		if isSystemRoot(n) {
			t.Errorf("%s is the person's", n)
		}
	}
}

func find(r Report, id string) *Item {
	for i := range r.Items {
		if r.Items[i].ID == id {
			return &r.Items[i]
		}
	}
	return nil
}

func TestBuildPersonalPC(t *testing.T) {
	two := 2
	r := build(facts{
		msAccounts: []string{"anna@outlook.com", "ANNA@outlook.com"},
		oneDrive:   []string{"anna@outlook.com"},
		redirected: []string{"Documents", "Desktop"},
		browsers:   []browser{{name: "Chrome", accounts: []string{"anna@gmail.com"}, logins: -1}, {name: "Firefox", logins: 12}},
		wifi:       5,
		creds:      0,
		secrets:    []string{"an SSH private key", "AWS keys"},
		folders: []FolderSize{
			{Label: "Documents", Kind: "personal", Bytes: 3 << 30},
			{Label: "Downloads", Kind: "personal", Bytes: 1 << 30, Partial: true},
			{Label: "Projects", Path: `D:\Projects`, Kind: "other", Bytes: 5 << 30},
		},
		bitlocker: &two,
		licence:   "retail",
	})

	ms := find(r, "microsoft-account")
	if ms == nil || ms.Status != "todo" || ms.Title != "Microsoft account: a***@outlook.com" {
		t.Errorf("microsoft account = %+v", ms)
	}
	if od := find(r, "onedrive"); od == nil || !strings.Contains(od.Detail, "Documents and Desktop are kept in it") {
		t.Errorf("onedrive = %+v", od)
	}
	if c := find(r, "browser-chrome"); c == nil || c.Status != "todo" || strings.Contains(c.Title, "anna@") {
		t.Errorf("chrome = %+v", c)
	}
	if ff := find(r, "browser-firefox"); ff == nil || ff.Status != "info" || !strings.Contains(ff.Detail, "12 passwords") {
		t.Errorf("firefox = %+v", ff)
	}
	if pf := find(r, "personal-files"); pf == nil || !strings.Contains(pf.Title, "at least 4.0 GB") {
		t.Errorf("personal files = %+v", pf)
	}
	if of := find(r, "other-folders"); of == nil || !strings.Contains(of.Detail, `D:\Projects`) {
		t.Errorf("other folders = %+v", of)
	}
	if s := find(r, "dev-secrets"); s == nil || !strings.Contains(s.Title, "2 developer keys") {
		t.Errorf("secrets = %+v", s)
	}
	if find(r, "credentials") != nil {
		t.Error("no saved sign-ins: no item")
	}
	if e := find(r, "encryption"); e == nil || !strings.Contains(e.How, "Clean data") {
		t.Errorf("encryption = %+v", e)
	}
	if r.Items[len(r.Items)-1].ID != "reset" {
		t.Error("reset comes last")
	}
	// microsoft account, onedrive, chrome, personal files, other folders, secrets
	if r.Todo != 6 {
		t.Errorf("todo = %d", r.Todo)
	}
}

func TestBuildCompanyPC(t *testing.T) {
	r := build(facts{join: joinState{known: true, azure: true, org: "Contoso"}, wifi: -1, creds: -1, licence: "volume"})
	org := find(r, "organisation")
	if org == nil || org.Title != "This PC belongs to Contoso" {
		t.Errorf("organisation = %+v", org)
	}
	if !strings.HasPrefix(find(r, "reset").How, "Your IT team") {
		t.Error("company PCs are reset by IT")
	}
	if find(r, "microsoft-account").Status != "ok" || r.Folders == nil {
		t.Errorf("got %+v", r)
	}
}

func TestSuspendedBitLockerIsNotProtection(t *testing.T) {
	five := 5
	r := build(facts{bitlocker: &five})
	for _, it := range r.Items {
		if it.ID == "encryption" {
			if it.Status != "todo" || strings.Contains(it.Detail, "cannot be read") {
				t.Errorf("suspended BitLocker = %+v", it)
			}
			return
		}
	}
	t.Error("no encryption item")
}
