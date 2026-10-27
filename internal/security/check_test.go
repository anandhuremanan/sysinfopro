package security

import (
	"encoding/json"
	"testing"
)

func statusOf(t *testing.T, report *Report, id string) Check {
	t.Helper()
	for _, c := range report.Checks {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("check %q missing", id)
	return Check{}
}

func parse(t *testing.T, data string) rawState {
	t.Helper()
	raw := rawState{}
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

// Shape captured from a real Windows 11 machine with a third-party antivirus
const healthy = `{"antivirus":[{"name":"Windows Defender","state":393472},{"name":"eScan Total Security","state":266240}],
"defender":{"available":true,"realtime":false,"signature_days":-1},
"firewall":[{"name":"Domain","enabled":true},{"name":"Private","enabled":true},{"name":"Public","enabled":true}],
"uac":1,"secure_boot":1,"bitlocker":1,"reboot_pending":false,"last_update_days":10,"is_admin":false,"rdp_deny":1}`

func TestEvaluateHealthy(t *testing.T) {
	report := evaluate(parse(t, healthy))
	if report.Passed != len(report.Checks) || report.Failed != 0 || report.Warned != 0 {
		t.Fatalf("expected all %d checks to pass, got %+v", len(report.Checks), report)
	}
	av := statusOf(t, report, "antivirus")
	if av.Detail != "eScan Total Security is protecting this PC" {
		t.Errorf("antivirus detail = %q; the passive Defender entry must not be listed", av.Detail)
	}
}

func TestEvaluateProblems(t *testing.T) {
	report := evaluate(parse(t, `{"antivirus":[{"name":"Windows Defender","state":393472}],
"defender":{"available":true,"realtime":false,"signature_days":3},
"firewall":[{"name":"Domain","enabled":true},{"name":"Public","enabled":false}],
"uac":0,"secure_boot":0,"bitlocker":2,"reboot_pending":true,"last_update_days":91,"rdp_deny":0}`))

	want := map[string]string{
		"antivirus":   StatusFail,
		"firewall":    StatusFail,
		"uac":         StatusFail,
		"secure_boot": StatusWarn,
		"encryption":  StatusWarn,
		"reboot":      StatusWarn,
		"updates":     StatusWarn,
		"rdp":         StatusWarn,
	}
	for id, status := range want {
		if got := statusOf(t, report, id); got.Status != status {
			t.Errorf("%s = %s (%s), want %s", id, got.Status, got.Detail, status)
		} else if got.Advice == "" {
			t.Errorf("%s has no advice", id)
		}
	}
	if report.Failed != 3 || report.Warned != 5 {
		t.Errorf("counts = %d failed, %d warned; want 3 and 5", report.Failed, report.Warned)
	}
}

func TestEvaluateUnreadable(t *testing.T) {
	report := evaluate(parse(t, `{"antivirus":[],"defender":{"available":false},"firewall":[],
"uac":null,"secure_boot":null,"bitlocker":null,"reboot_pending":false,"last_update_days":-1,"rdp_deny":null}`))

	for _, id := range []string{"antivirus", "firewall", "uac", "secure_boot", "encryption", "updates", "rdp"} {
		if got := statusOf(t, report, id); got.Status != StatusUnknown {
			t.Errorf("%s = %s, want unknown when the value cannot be read", id, got.Status)
		}
	}
	if report.Failed != 0 {
		t.Errorf("an unreadable setting must never count as a failure, got %d", report.Failed)
	}
}

func TestStaleDefinitionsWarn(t *testing.T) {
	// 0x061010: Defender on (0x10) with out-of-date definitions (0x10)
	report := evaluate(parse(t, `{"antivirus":[{"name":"Windows Defender","state":397328}],"defender":{"available":true,"realtime":true}}`))
	if got := statusOf(t, report, "antivirus"); got.Status != StatusWarn {
		t.Errorf("antivirus = %s (%s), want warn for stale definitions", got.Status, got.Detail)
	}
}

func TestSnoozedAntivirusIsNotProtection(t *testing.T) {
	// 0x041100: third-party antivirus snoozed (0x11); Defender passive
	report := evaluate(parse(t, `{"antivirus":[{"name":"Windows Defender","state":393472},{"name":"Acme AV","state":266496}],"defender":{"available":true,"realtime":false}}`))
	if got := statusOf(t, report, "antivirus"); got.Status != StatusWarn {
		t.Errorf("antivirus = %s (%s), want warn: a snoozed antivirus is not scanning", got.Status, got.Detail)
	}
}

func TestBitLockerStates(t *testing.T) {
	want := map[int]string{1: StatusPass, 6: StatusPass, 3: StatusPass, 2: StatusWarn, 4: StatusWarn, 5: StatusWarn, 8: StatusWarn, 0: StatusUnknown}
	for code, status := range want {
		v := code
		got := encryptionCheck(rawState{BitLocker: &v})
		if got.Status != status {
			t.Errorf("BitLocker %d = %s (%s), want %s", code, got.Status, got.Detail, status)
		}
	}
}
