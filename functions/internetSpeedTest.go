package functions

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const downloadEndpoint = "https://speed.cloudflare.com/__down?bytes="

// Cloudflare rate-limits large files per network first, so a refused size steps down to the next
var downloadSizes = []int{25000000, 10000000, 1000000}

const uploadEndpoint = "https://speed.cloudflare.com/__up"

// Tiny plain-text response from the nearest Cloudflare edge
const latencyEndpoint = "https://1.1.1.1/cdn-cgi/trace"

const (
	speedTestStreams  = 4
	speedTestDuration = 8 * time.Second
	// Connection setup and TCP slow start are excluded from the result
	speedTestWarmup = 1500 * time.Millisecond
	// The first moments mostly fill socket buffers, so the live reading skips them
	speedLiveSettle   = 600 * time.Millisecond
	speedSampleEvery  = 200 * time.Millisecond
	speedLiveWindow   = 5 // samples (1s) behind the live reading
	latencySamples    = 10
	uploadPayloadSize = 1024 * 1024
)

// SpeedProgress is a live reading emitted while a test phase runs
type SpeedProgress struct {
	Phase    string  `json:"phase"` // "latency" | "download" | "upload"
	Mbps     float64 `json:"mbps"`
	Progress float64 `json:"progress"` // 0-100 within the phase
}

type SpeedProgressFunc func(SpeedProgress)

// speedRun is the state shared by the parallel streams of one test phase
type speedRun struct {
	totalBytes  int64
	sizeTier    int32
	rateLimited atomic.Bool
}

const (
	speedErrRateLimited = "rate_limited"
	speedErrFailed      = "failed"
)

// errorCode explains a phase that measured nothing
func (r *speedRun) errorCode(mbps float64) string {
	if mbps > 0 {
		return ""
	}
	if r.rateLimited.Load() {
		return speedErrRateLimited
	}
	return speedErrFailed
}

func GetInterNetSpeed() (*InternetSpeedStat, error) {
	return GetInterNetSpeedWithProgress(nil)
}

func GetInterNetSpeedWithProgress(progress SpeedProgressFunc) (*InternetSpeedStat, error) {
	if progress == nil {
		progress = func(SpeedProgress) {}
	}
	speedStat := &InternetSpeedStat{}

	// 1. Measure Latency, Jitter, Packet Loss
	ping, jitter, loss := measureLatencyMetrics(progress)
	speedStat.PingLatencyMs = ping
	speedStat.JitterMs = jitter
	speedStat.PacketLossPct = loss

	// 2. Measure Download Speed (Multi-stream parallel HTTP GET)
	download := &speedRun{}
	downloadProvider := speedProviderCloudflare
	speedStat.DownloadSpeed = measureThroughput("download", progress, download, downloadWorker)
	speedStat.DownloadError = download.errorCode(speedStat.DownloadSpeed)
	if speedStat.DownloadError != "" {
		// Cloudflare refused or was unreachable: try Measurement Lab before giving up
		downloadProvider = ""
		if mbps, err := measureWithMLab("download", progress); err == nil {
			speedStat.DownloadSpeed = mbps
			speedStat.DownloadError = ""
			downloadProvider = speedProviderMLab
		}
	}

	// 3. Measure Upload Speed (Multi-stream parallel HTTP POST)
	payload := make([]byte, uploadPayloadSize)
	rand.Read(payload)
	upload := &speedRun{}
	speedStat.UploadSpeed = measureThroughput("upload", progress, upload, func(ctx context.Context, client *http.Client, run *speedRun) {
		uploadWorker(ctx, client, payload, run)
	})
	speedStat.UploadError = upload.errorCode(speedStat.UploadSpeed)
	uploadProvider := speedProviderCloudflare
	if speedStat.UploadError != "" {
		uploadProvider = ""
		if mbps, err := measureWithMLab("upload", progress); err == nil {
			speedStat.UploadSpeed = mbps
			speedStat.UploadError = ""
			uploadProvider = speedProviderMLab
		}
	}
	speedStat.Provider = speedProviderLabel(downloadProvider, uploadProvider)

	return speedStat, nil
}

func measureLatencyMetrics(progress SpeedProgressFunc) (float64, float64, float64) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	probe := func() (float64, bool) {
		start := time.Now()
		resp, err := client.Get(latencyEndpoint)
		if err != nil {
			return 0, false
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return float64(time.Since(start).Microseconds()) / 1000.0, true
	}

	// First request pays for DNS, TCP and TLS setup; later ones reuse the connection
	probe()

	latencies := []float64{}
	for i := 0; i < latencySamples; i++ {
		if lat, ok := probe(); ok {
			latencies = append(latencies, lat)
		}
		progress(SpeedProgress{Phase: "latency", Progress: float64(i+1) / latencySamples * 100})
		time.Sleep(50 * time.Millisecond)
	}

	if len(latencies) == 0 {
		return 0, 0, 100.0
	}

	loss := (float64(latencySamples-len(latencies)) / latencySamples) * 100.0

	// Jitter (typical difference between consecutive samples)
	diffs := []float64{}
	for i := 1; i < len(latencies); i++ {
		diffs = append(diffs, math.Abs(latencies[i]-latencies[i-1]))
	}

	// Medians ignore the occasional slow outlier (e.g. a reconnect)
	ping := median(latencies)
	jitter := median(diffs)

	return math.Round(ping*10) / 10, math.Round(jitter*10) / 10, math.Round(loss*10) / 10
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64{}, values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

type speedWorker func(ctx context.Context, client *http.Client, run *speedRun)

// measureThroughput runs parallel streams and returns the steady-state Mbps after warm-up
func measureThroughput(phase string, progress SpeedProgressFunc, run *speedRun, worker speedWorker) float64 {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			MaxIdleConnsPerHost: 10,
			DisableCompression:  true,
		},
	}
	defer client.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(context.Background(), speedTestDuration)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < speedTestStreams; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, client, run)
		}()
	}
	workersDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(workersDone)
	}()

	type sample struct {
		at    time.Duration
		bytes int64
	}
	start := time.Now()
	samples := []sample{{0, 0}}
	baseIdx := -1
	settleIdx := -1

	ticker := time.NewTicker(speedSampleEvery)
	defer ticker.Stop()

sampling:
	for {
		select {
		case <-ticker.C:
			now := sample{time.Since(start), atomic.LoadInt64(&run.totalBytes)}
			samples = append(samples, now)
			if baseIdx < 0 && now.at >= speedTestWarmup {
				baseIdx = len(samples) - 1
			}

			if settleIdx < 0 && now.at >= speedLiveSettle {
				settleIdx = len(samples) - 1
			}

			live := 0.0
			if settleIdx >= 0 {
				from := samples[max(settleIdx, len(samples)-1-speedLiveWindow)]
				live = toMbps(now.bytes-from.bytes, now.at-from.at)
			}
			progress(SpeedProgress{
				Phase:    phase,
				Mbps:     live,
				Progress: math.Min(100, now.at.Seconds()/speedTestDuration.Seconds()*100),
			})
		case <-ctx.Done():
			break sampling
		case <-workersDone:
			// every stream failed early (e.g. connection dropped)
			break sampling
		}
	}
	cancel()
	<-workersDone

	last := samples[len(samples)-1]
	if baseIdx < 0 || last.at-samples[baseIdx].at < time.Second {
		// too short for a steady-state window, fall back to the overall average
		baseIdx = 0
	}
	base := samples[baseIdx]
	return math.Round(toMbps(last.bytes-base.bytes, last.at-base.at)*10) / 10
}

// Mbps = (bytes * 8) / (1,000,000 * seconds)
func toMbps(bytes int64, elapsed time.Duration) float64 {
	if elapsed <= 0 || bytes <= 0 {
		return 0
	}
	return (float64(bytes) * 8.0) / (1000000.0 * elapsed.Seconds())
}

func downloadWorker(ctx context.Context, client *http.Client, run *speedRun) {
	buf := make([]byte, 64*1024) // 64KB buffer

	// Keep requesting so fast connections don't run out of file before the test ends
	for ctx.Err() == nil {
		tier := atomic.LoadInt32(&run.sizeTier)
		url := downloadEndpoint + strconv.Itoa(downloadSizes[tier])
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			return
		}
		if resp.StatusCode != http.StatusOK {
			// an error page is not throughput
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusTooManyRequests {
				return
			}
			if int(tier) == len(downloadSizes)-1 {
				run.rateLimited.Store(true)
				return
			}
			// this size is refused: every stream moves to the next smaller one
			atomic.CompareAndSwapInt32(&run.sizeTier, tier, tier+1)
			continue
		}

		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				atomic.AddInt64(&run.totalBytes, int64(n))
			}
			if readErr != nil {
				resp.Body.Close()
				if readErr != io.EOF {
					return
				}
				break
			}
		}
	}
}

// countingReader counts bytes as the transport sends them, so a partly sent chunk still counts
type countingReader struct {
	r     io.Reader
	total *int64
	sent  int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		atomic.AddInt64(c.total, int64(n))
		atomic.AddInt64(&c.sent, int64(n))
	}
	return n, err
}

func uploadWorker(ctx context.Context, client *http.Client, payload []byte, run *speedRun) {
	for ctx.Err() == nil {
		body := &countingReader{r: bytes.NewReader(payload), total: &run.totalBytes}
		req, err := http.NewRequestWithContext(ctx, "POST", uploadEndpoint, body)
		if err != nil {
			return
		}
		req.ContentLength = int64(len(payload))
		req.Header.Set("Content-Type", "application/octet-stream")

		resp, err := client.Do(req)
		if err != nil {
			return
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			// rejected upload: take its bytes back out and stop this stream
			atomic.AddInt64(&run.totalBytes, -atomic.LoadInt64(&body.sent))
			if resp.StatusCode == http.StatusTooManyRequests {
				run.rateLimited.Store(true)
			}
			return
		}
	}
}
