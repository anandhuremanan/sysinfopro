//go:build !windows

package handover

import "context"

// Check is only meaningful on Windows; elsewhere it returns the reset step alone
func Check(ctx context.Context) Report {
	return build(facts{wifi: -1, creds: -1})
}
