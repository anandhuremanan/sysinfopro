//go:build windows

package batteryreport

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// Get asks Windows for a fresh battery report and reads it. It needs no
// administrator rights and takes a second or two.
func Get(ctx context.Context) (*Report, error) {
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
	msg, err := cmd.CombinedOutput()

	f, openErr := os.Open(out)
	if openErr != nil {
		if err != nil {
			// powercfg says why, for example on a PC without a battery
			return &Report{Batteries: []Battery{}, History: []Point{}, Sessions: []Session{}, Summary: "This PC has no battery."}, nil
		}
		return nil, fmt.Errorf("Windows did not produce a battery report: %s", msg)
	}
	defer f.Close()
	return Parse(f)
}
