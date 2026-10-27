// Package winexec runs command-line tools without flashing a console window,
// with a default time limit so a hung tool can never block the app forever.
package winexec

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// DefaultTimeout applies when the caller's context has no deadline of its own
const DefaultTimeout = 2 * time.Minute

// Command prepares name with args to run without a console window. When ctx has
// no deadline, DefaultTimeout applies. Call cancel once the command has finished.
func Command(ctx context.Context, name string, args ...string) (cmd *exec.Cmd, cancel context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		ctx, cancel = context.WithCancel(ctx)
	} else {
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
	}
	cmd = exec.CommandContext(ctx, name, args...)
	Hide(cmd)
	return cmd, cancel
}

// Output runs the command and returns its standard output
func Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd, cancel := Command(ctx, name, args...)
	defer cancel()
	return cmd.Output()
}

// CombinedOutput runs the command and returns its standard output and error together
func CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd, cancel := Command(ctx, name, args...)
	defer cancel()
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	err := cmd.Run()
	return b.Bytes(), err
}
