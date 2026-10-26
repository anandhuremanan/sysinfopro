//go:build !windows

package netdoctor

import (
	"context"
	"errors"
	"time"
)

func primaryAdapter() *Adapter { return nil }

func ping(address string, timeout time.Duration) (int64, error) {
	return 0, errors.New("not supported on this platform")
}

func readProxy(ctx context.Context) Proxy { return Proxy{} }
