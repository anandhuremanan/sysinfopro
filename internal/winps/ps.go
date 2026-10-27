// Package winps runs read-only PowerShell queries without flashing a console window.
package winps

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"sysinfopro/internal/winexec"
)

// The script arrives on standard input and runs as one block. This avoids temp files,
// is not subject to script execution policy, and does not use -EncodedCommand, which
// security software often blocks silently. Scripts must be plain ASCII.
const bootstrap = "[Console]::OutputEncoding = [Text.Encoding]::UTF8; & ([scriptblock]::Create([Console]::In.ReadToEnd()))"

// DefaultTimeout applies when the caller's context has no deadline: a hung WMI
// provider must not leave a query (and a powershell.exe) running forever
const DefaultTimeout = 45 * time.Second

// Run executes script and returns its standard output as UTF-8
func Run(ctx context.Context, script string) ([]byte, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}
	cmd, cancel := winexec.Command(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", bootstrap)
	defer cancel()
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
