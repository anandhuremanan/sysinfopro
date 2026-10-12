// Package winps runs read-only PowerShell queries without flashing a console window.
package winps

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// The script arrives on standard input and runs as one block. This avoids temp files,
// is not subject to script execution policy, and does not use -EncodedCommand, which
// security software often blocks silently. Scripts must be plain ASCII.
const bootstrap = "[Console]::OutputEncoding = [Text.Encoding]::UTF8; & ([scriptblock]::Create([Console]::In.ReadToEnd()))"

// Run executes script and returns its standard output as UTF-8
func Run(ctx context.Context, script string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", bootstrap)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	cmd.Stdin = strings.NewReader(script)

	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("powershell query failed: %w", err)
	}

	result := bytes.TrimSpace(out.Bytes())
	if len(result) == 0 {
		return nil, fmt.Errorf("powershell query returned nothing (it may have been blocked by security software)")
	}
	return result, nil
}
