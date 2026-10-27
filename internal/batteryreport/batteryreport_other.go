//go:build !windows

package batteryreport

import "context"

func Get(ctx context.Context) (*Report, error) {
	return &Report{Batteries: []Battery{}, History: []Point{}, Sessions: []Session{}, Summary: "This PC has no battery."}, nil
}
