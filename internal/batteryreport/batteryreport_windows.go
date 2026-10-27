//go:build windows

package batteryreport

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetPowerStatus = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetSystemPowerStatus")

// hasBattery asks Windows directly; true when it cannot tell, so a failed
// report is never mistaken for "no battery"
func hasBattery() bool {
	var s struct {
		ACLineStatus, BatteryFlag, BatteryLifePercent, SystemStatusFlag byte
		BatteryLifeTime, BatteryFullLifeTime                            uint32
	}
	if r, _, _ := procGetPowerStatus.Call(uintptr(unsafe.Pointer(&s))); r == 0 {
		return true
	}
	const noSystemBattery = 128
	return s.BatteryFlag&noSystemBattery == 0
}

func noBattery() *Report {
	return &Report{Batteries: []Battery{}, History: []Point{}, Sessions: []Session{}, Summary: "This PC has no battery."}
}

// Get asks Windows for a fresh battery report and reads it. It needs no
// administrator rights and takes a second or two.
func Get(ctx context.Context) (*Report, error) {
	if !hasBattery() {
		return noBattery(), nil
	}
	dir, err := os.MkdirTemp("", "sysinfopro-battery-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "battery.xml")

	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, "powercfg", "/batteryreport", "/xml", "/output", out)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	msg, _ := cmd.CombinedOutput()

	f, err := os.Open(out)
	if err != nil {
		// the PC has a battery, so this is a failure to read it, not "no battery"
		return nil, fmt.Errorf("Windows could not produce its battery report: %s", strings.TrimSpace(string(msg)))
	}
	defer f.Close()
	return Parse(f)
}
