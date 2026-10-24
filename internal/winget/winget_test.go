package winget

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// as winget prints it to a pipe: a spinner redrawn with carriage returns, the
// table, a summary, then a second table that must be ignored
const sample = "\r   - \r   \\ \r   | \r                                                                                                                        \r" +
	"Name                                   Id                                Version        Available      Source\r\n" +
	"------------------------------------------------------------------------------------------------------------------\r\n" +
	"Git                                    Git.Git                           2.44.0         2.45.1         winget\r\n" +
	"Microsoft Visual Studio Code (User)    Microsoft.VisualStudioCode        1.88.1         1.89.0         winget\r\n" +
	"Node.js 20                             OpenJS.NodeJS.20                  20.11.1        20.14.0        winget\r\n" +
	"Some Very Long Application Name Tha…   SomeVendor.SomeVeryLongApplica…   1.0            2.0            winget\r\n" +
	"Zoë's Photo Tool                       Zoe.PhotoTool                     3.1            3.2            winget\r\n" +
	"5 upgrades available.\r\n" +
	"\r\n" +
	"The following packages have an upgrade available, but require explicit targeting for upgrade:\r\n" +
	"Name      Id             Version Available Source\r\n" +
	"------------------------------------------------\r\n" +
	"Pinned    Pinned.App     1.0     2.0       winget\r\n"

func TestParseUpgradeTable(t *testing.T) {
	got := parseUpgradeTable([]byte(sample))
	if len(got) != 5 {
		t.Fatalf("got %d packages, want 5: %+v", len(got), got)
	}
	want := Package{Name: "Git", ID: "Git.Git", Version: "2.44.0", Available: "2.45.1", Source: "winget"}
	if got[0] != want {
		t.Errorf("first = %+v\nwant  %+v", got[0], want)
	}
	if got[1].Name != "Microsoft Visual Studio Code (User)" || got[1].ID != "Microsoft.VisualStudioCode" {
		t.Errorf("names with spaces and brackets: %+v", got[1])
	}
	if !got[3].Truncated || got[2].Truncated || got[0].Truncated {
		t.Errorf("only the cut-off ID should be marked truncated: %+v", got)
	}
	if got[4].Name != "Zoë's Photo Tool" || got[4].ID != "Zoe.PhotoTool" || got[4].Available != "3.2" {
		t.Errorf("non-ASCII name shifted the columns: %+v", got[4])
	}
	for _, p := range got {
		if p.ID == "Pinned.App" {
			t.Error("packages that need explicit targeting must be left out")
		}
	}
}

func TestParseTranslatedHeaders(t *testing.T) {
	out := "Name       ID              Version Verfügbar Quelle\n" +
		"--------------------------------------------------\n" +
		"7-Zip      7zip.7zip       23.01   24.07     winget\n" +
		"1 Aktualisierungen verfügbar.\n"
	got := parseUpgradeTable([]byte(out))
	if len(got) != 1 || got[0].ID != "7zip.7zip" || got[0].Available != "24.07" {
		t.Errorf("got %+v", got)
	}
}

func TestParseNothingToUpgrade(t *testing.T) {
	if got := parseUpgradeTable([]byte("No installed package found matching input criteria.\r\n")); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestBlockedByPolicy(t *testing.T) {
	if !isBlocked([]byte("This operation is disabled by Group Policy: Enable Windows Package Manager\r\n"), errors.New("exit status 1")) {
		t.Error("the policy message should be recognised")
	}
	if isBlocked([]byte("Found Git [Git.Git]"), &exec.ExitError{}) {
		t.Error("an ordinary failure is not a policy block")
	}
}

func TestUpgradeRejectsOddIDs(t *testing.T) {
	for _, id := range []string{"--force", "Some.App …", "a b", "", `x"y`} {
		if err := Upgrade(t.Context(), id); err == nil || !strings.Contains(err.Error(), "not a winget package ID") {
			t.Errorf("Upgrade(%q) = %v, want it refused before running winget", id, err)
		}
	}
}

func TestLastLineSkipsSpinner(t *testing.T) {
	out := []byte("Found Git [Git.Git] Version 2.45.1\r\nInstaller hash does not match.\r\n   - \r   \\ \r\n")
	if got := lastLine(out); got != "Installer hash does not match." {
		t.Errorf("lastLine = %q", got)
	}
}
