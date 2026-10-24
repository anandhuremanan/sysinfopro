//go:build !windows

// Package notify shows Windows notifications under the app's own name.
package notify

import (
	"context"
	"errors"
)

// Send is only implemented on Windows
func Send(title, body string) error {
	return errors.New("notifications are not supported on this platform")
}

// Status is only implemented on Windows
func Status(ctx context.Context) string {
	return "unknown"
}
