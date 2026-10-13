package functions

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/m-lab/ndt7-client-go"
	"github.com/m-lab/ndt7-client-go/spec"
)

const (
	speedProviderCloudflare = "Cloudflare"
	speedProviderMLab       = "Measurement Lab"

	// an ndt7 test runs for about 10 seconds; the rest is for finding a server
	mlabTestDuration = 10 * time.Second
	mlabTimeout      = 25 * time.Second
)

// measureWithMLab runs one phase ("download" or "upload") against the nearest
// Measurement Lab server. It is the fallback when Cloudflare refuses the test.
func measureWithMLab(phase string, progress SpeedProgressFunc) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mlabTimeout)
	defer cancel()

	client := ndt7.NewClient("sysinfopro", "3.0")

	var measurements <-chan spec.Measurement
	var err error
	if phase == "download" {
		measurements, err = client.StartDownload(ctx)
	} else {
		measurements, err = client.StartUpload(ctx)
	}
	if err != nil {
		return 0, fmt.Errorf("measurement lab %s test failed: %w", phase, err)
	}

	mbps := 0.0
	for m := range measurements {
		// the client-side counter exists in both directions and updates several times a second
		if m.Origin != spec.OriginClient || m.AppInfo == nil || m.AppInfo.ElapsedTime <= 0 {
			continue
		}
		// bytes * 8 / microseconds = megabits per second
		mbps = float64(m.AppInfo.NumBytes) * 8 / float64(m.AppInfo.ElapsedTime)
		elapsed := time.Duration(m.AppInfo.ElapsedTime) * time.Microsecond
		progress(SpeedProgress{
			Phase:    phase,
			Mbps:     mbps,
			Progress: math.Min(100, elapsed.Seconds()/mlabTestDuration.Seconds()*100),
		})
	}

	if mbps <= 0 {
		return 0, fmt.Errorf("measurement lab %s test returned no data", phase)
	}
	return math.Round(mbps*10) / 10, nil
}

// speedProviderLabel names the service(s) the two phases were measured against
func speedProviderLabel(download, upload string) string {
	switch {
	case download == "" && upload == "":
		return ""
	case download == "":
		return upload
	case upload == "" || download == upload:
		return download
	default:
		return download + " and " + upload
	}
}
