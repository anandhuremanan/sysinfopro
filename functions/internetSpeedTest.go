package functions

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Fast CDN endpoints for multi-stream speed testing
var downloadEndpoints = []string{
	"https://speed.cloudflare.com/__down?bytes=50000000",
	"https://speed.cloudflare.com/__down?bytes=25000000",
}

const uploadEndpoint = "https://speed.cloudflare.com/__up"

func GetInterNetSpeed() (*InternetSpeedStat, error) {
	speedStat := &InternetSpeedStat{}

	// 1. Measure Latency, Jitter, Packet Loss
	ping, jitter, loss := measureLatencyMetrics()
	speedStat.PingLatencyMs = ping
	speedStat.JitterMs = jitter
	speedStat.PacketLossPct = loss

	// 2. Measure Download Speed (Multi-stream parallel HTTP GET)
	downloadSpeed := measureParallelDownloadSpeed()
	speedStat.DownloadSpeed = downloadSpeed

	// 3. Measure Upload Speed (Multi-stream parallel HTTP POST)
	uploadSpeed := measureParallelUploadSpeed()
	speedStat.UploadSpeed = uploadSpeed

	return speedStat, nil
}

func measureLatencyMetrics() (float64, float64, float64) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	samples := 6
	success := 0
	latencies := []float64{}

	for i := 0; i < samples; i++ {
		start := time.Now()
		resp, err := client.Get("https://1.1.1.1/cdn-cgi/trace")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lat := float64(time.Since(start).Microseconds()) / 1000.0
			latencies = append(latencies, lat)
			success++
		}
		time.Sleep(50 * time.Millisecond)
	}

	if success == 0 {
		return 0, 0, 100.0
	}

	loss := ((float64(samples) - float64(success)) / float64(samples)) * 100.0
	sum := 0.0
	for _, l := range latencies {
		sum += l
	}
	avgPing := sum / float64(len(latencies))

	// Calculate jitter (mean absolute deviation)
	jitterSum := 0.0
	for _, l := range latencies {
		jitterSum += math.Abs(l - avgPing)
	}
	jitter := jitterSum / float64(len(latencies))

	return math.Round(avgPing*10) / 10, math.Round(jitter*10) / 10, math.Round(loss*10) / 10
}

// measureParallelDownloadSpeed uses 4 parallel HTTP streams to measure peak Mbps (matching Google Speed Test)
func measureParallelDownloadSpeed() float64 {
	numStreams := 4
	testDuration := 6 * time.Second

	client := &http.Client{
		Timeout: testDuration + 2*time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 10,
			DisableCompression:  true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	var totalBytes int64
	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		go func(streamIdx int) {
			defer wg.Done()

			url := downloadEndpoints[streamIdx%len(downloadEndpoints)]
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			buf := make([]byte, 64*1024) // 64KB buffer
			for {
				select {
				case <-ctx.Done():
					return
				default:
					n, readErr := resp.Body.Read(buf)
					if n > 0 {
						atomic.AddInt64(&totalBytes, int64(n))
					}
					if readErr != nil {
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime).Seconds()

	if elapsed < 0.5 || totalBytes == 0 {
		return 0
	}

	// Mbps = (bytes * 8) / (1,000,000 * seconds)
	mbps := (float64(totalBytes) * 8.0) / (1000000.0 * elapsed)
	return math.Round(mbps*10) / 10
}

// measureParallelUploadSpeed uses 4 parallel HTTP POST streams to measure peak upload Mbps
func measureParallelUploadSpeed() float64 {
	numStreams := 4
	testDuration := 5 * time.Second
	payloadSize := 2 * 1024 * 1024 // 2MB payload chunk

	// Generate random payload buffer once
	payload := make([]byte, payloadSize)
	rand.Read(payload)

	client := &http.Client{
		Timeout: testDuration + 2*time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 10,
			DisableCompression:  true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	var totalBytes int64
	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for ctx.Err() == nil {
				req, err := http.NewRequestWithContext(ctx, "POST", uploadEndpoint, bytes.NewReader(payload))
				if err != nil {
					return
				}
				req.Header.Set("Content-Type", "application/octet-stream")

				resp, err := client.Do(req)
				if err != nil {
					return
				}

				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				atomic.AddInt64(&totalBytes, int64(payloadSize))
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(startTime).Seconds()

	if elapsed < 0.5 || totalBytes == 0 {
		return 0
	}

	// Mbps = (bytes * 8) / (1,000,000 * seconds)
	mbps := (float64(totalBytes) * 8.0) / (1000000.0 * elapsed)
	return math.Round(mbps*10) / 10
}
