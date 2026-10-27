//go:build !windows

package slownow

import "context"

func collect(ctx context.Context) Sample {
	return Sample{PerformancePct: -1, DiskBusyPct: -1}
}
