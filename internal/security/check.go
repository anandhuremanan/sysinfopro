package security

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"myproject/internal/winps"
)

//go:embed security.ps1
var securityScript string

const (
	StatusPass    = "pass"
	StatusWarn    = "warn"
	StatusFail    = "fail"
	StatusUnknown = "unknown"
)

type Check struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"` // pass | warn | fail | unknown
	Detail string `json:"detail"`
	Advice string `json:"advice"`
}

type Report struct {
	Checks  []Check `json:"checks"`
	Passed  int     `json:"passed"`
	Warned  int     `json:"warned"`
	Failed  int     `json:"failed"`
	Unknown int     `json:"unknown"`
}

// rawState is what security.ps1 emits
type rawState struct {
	Antivirus []struct {
		Name  string `json:"name"`
		State int64  `json:"state"`
	} `json:"antivirus"`
	Defender struct {
		Available     bool `json:"available"`
		Realtime      bool `json:"realtime"`
		SignatureDays int  `json:"signature_days"`
	} `json:"defender"`
	Firewall []struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	} `json:"firewall"`
	UAC            *int `json:"uac"`
	SecureBoot     *int `json:"secure_boot"`
	BitLocker      *int `json:"bitlocker"`
	RebootPending  bool `json:"reboot_pending"`
	LastUpdateDays int  `json:"last_update_days"`
	// fDenyTSConnections: 0 = Remote Desktop on, 1 = off, nil = could not read
	RDPDeny *int `json:"rdp_deny"`
}

// GetSecurityReport runs read-only checks of the machine's protection settings
func GetSecurityReport(ctx context.Context) (*Report, error) {
	out, err := winps.Run(ctx, securityScript)
	if err != nil {
		return nil, err
	}
	raw := rawState{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("could not read security state: %w", err)
	}
	return evaluate(raw), nil
}

func evaluate(raw rawState) *Report {
	checks := []Check{
		antivirusCheck(raw),
		firewallCheck(raw),
		updatesCheck(raw),
		encryptionCheck(raw),
		secureBootCheck(raw),
		uacCheck(raw),
		rebootCheck(raw),
		remoteDesktopCheck(raw),
	}

	report := &Report{Checks: checks}
	for _, c := range checks {
		switch c.Status {
		case StatusPass:
			report.Passed++
		case StatusWarn:
			report.Warned++
		case StatusFail:
			report.Failed++
		default:
			report.Unknown++
		}
	}
	return report
}

func antivirusCheck(raw rawState) Check {
	c := Check{ID: "antivirus", Title: "Antivirus"}

	// productState is 0xAABBCC: BB = 0x10/0x11 when protection is on, CC = 0x10 when definitions are stale
	// BB = 0x10 is on; 0x11 is "snoozed", turned off for a while
	active, stale, snoozed := []string{}, []string{}, []string{}
	for _, av := range raw.Antivirus {
		status := (av.State >> 8) & 0xFF
		switch status {
		case 0x10:
			active = append(active, av.Name)
			if av.State&0xFF == 0x10 {
				stale = append(stale, av.Name)
			}
		case 0x11:
			snoozed = append(snoozed, av.Name)
		}
	}

	switch {
	case len(active) > 0 && len(stale) == len(active):
		c.Status = StatusWarn
		c.Detail = strings.Join(stale, ", ") + " is on, but its definitions are out of date"
		c.Advice = "Update your antivirus definitions."
	case len(active) > 0:
		c.Status = StatusPass
		c.Detail = strings.Join(active, ", ") + " is protecting this PC"
	case raw.Defender.Available && raw.Defender.Realtime:
		c.Status = StatusPass
		c.Detail = "Microsoft Defender real-time protection is on"
	case len(snoozed) > 0:
		c.Status = StatusWarn
		c.Detail = strings.Join(snoozed, ", ") + " is paused for now, so nothing is scanning"
		c.Advice = "Turn real-time protection back on."
	case len(raw.Antivirus) == 0 && !raw.Defender.Available:
		c.Status = StatusUnknown
		c.Detail = "Could not read antivirus status"
	default:
		c.Status = StatusFail
		c.Detail = "No antivirus is actively protecting this PC"
		c.Advice = "Turn on Microsoft Defender in Windows Security."
	}
	return c
}

func firewallCheck(raw rawState) Check {
	c := Check{ID: "firewall", Title: "Firewall"}
	if len(raw.Firewall) == 0 {
		c.Status = StatusUnknown
		c.Detail = "Could not read firewall profiles"
		return c
	}

	off := []string{}
	for _, profile := range raw.Firewall {
		if !profile.Enabled {
			off = append(off, profile.Name)
		}
	}
	if len(off) == 0 {
		c.Status = StatusPass
		c.Detail = "On for all network profiles"
		return c
	}
	c.Status = StatusFail
	c.Detail = "Off for: " + strings.Join(off, ", ")
	c.Advice = "Turn the firewall on in Windows Security, unless a third-party firewall replaces it."
	return c
}

func updatesCheck(raw rawState) Check {
	c := Check{ID: "updates", Title: "Windows updates"}
	switch days := raw.LastUpdateDays; {
	case days < 0:
		c.Status = StatusUnknown
		c.Detail = "Could not read update history"
	case days <= 45:
		c.Status = StatusPass
		c.Detail = fmt.Sprintf("Last update installed %d days ago", days)
	default:
		c.Status = StatusWarn
		c.Detail = fmt.Sprintf("Last update installed %d days ago", days)
		c.Advice = "Check for updates in Windows Update."
	}
	return c
}

func encryptionCheck(raw rawState) Check {
	c := Check{ID: "encryption", Title: "Drive encryption"}
	if raw.BitLocker == nil {
		c.Status = StatusUnknown
		c.Detail = "Could not read BitLocker status"
		return c
	}
	// System.Volume.BitLockerProtection: 1 on, 2 off, 3 encrypting, 4 decrypting,
	// 5 suspended, 6 on (locked), 8 waiting for activation
	switch *raw.BitLocker {
	case 1, 6:
		c.Status = StatusPass
		c.Detail = "System drive is encrypted with BitLocker"
	case 3:
		c.Status = StatusPass
		c.Detail = "System drive is being encrypted"
	case 5:
		c.Status = StatusWarn
		c.Detail = "BitLocker protection is suspended"
		c.Advice = "Resume BitLocker protection."
	case 8:
		c.Status = StatusWarn
		c.Detail = "Encryption is waiting to be turned on, so files are not protected yet"
		c.Advice = "Finish turning on device encryption in Settings > Privacy & security > Device encryption."
	case 4:
		c.Status = StatusWarn
		c.Detail = "BitLocker is being turned off"
		c.Advice = "If you did not mean to turn it off, turn BitLocker back on."
	case 2:
		c.Status = StatusWarn
		c.Detail = "System drive is not encrypted"
		c.Advice = "Turn on BitLocker or device encryption so a lost laptop does not expose your files."
	default:
		c.Status = StatusUnknown
		c.Detail = "This drive does not report an encryption status"
	}
	return c
}

func secureBootCheck(raw rawState) Check {
	c := Check{ID: "secure_boot", Title: "Secure Boot"}
	switch {
	case raw.SecureBoot == nil:
		c.Status = StatusUnknown
		c.Detail = "Not reported (legacy BIOS or unsupported firmware)"
	case *raw.SecureBoot == 1:
		c.Status = StatusPass
		c.Detail = "On"
	default:
		c.Status = StatusWarn
		c.Detail = "Off"
		c.Advice = "Enable Secure Boot in your firmware (UEFI) settings."
	}
	return c
}

func uacCheck(raw rawState) Check {
	c := Check{ID: "uac", Title: "User Account Control"}
	switch {
	case raw.UAC == nil:
		c.Status = StatusUnknown
		c.Detail = "Could not read the setting"
	case *raw.UAC == 1:
		c.Status = StatusPass
		c.Detail = "On"
	default:
		c.Status = StatusFail
		c.Detail = "Off: programs can make system changes without asking"
		c.Advice = "Turn User Account Control back on."
	}
	return c
}

func rebootCheck(raw rawState) Check {
	c := Check{ID: "reboot", Title: "Pending restart"}
	if raw.RebootPending {
		c.Status = StatusWarn
		c.Detail = "Windows is waiting for a restart to finish installing updates"
		c.Advice = "Restart this PC."
		return c
	}
	c.Status = StatusPass
	c.Detail = "No restart is pending"
	return c
}

func remoteDesktopCheck(raw rawState) Check {
	c := Check{ID: "rdp", Title: "Remote Desktop"}
	if raw.RDPDeny == nil {
		c.Status = StatusUnknown
		c.Detail = "Could not read the setting"
		return c
	}
	if *raw.RDPDeny == 0 {
		c.Status = StatusWarn
		c.Detail = "On: this PC accepts remote connections"
		c.Advice = "Turn Remote Desktop off if you do not use it."
		return c
	}
	c.Status = StatusPass
	c.Detail = "Off"
	return c
}
