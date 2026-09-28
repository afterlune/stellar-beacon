package monitoring

import (
	"strings"
	"sync"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

var latencyBucketUpperMs = [...]float64{50, 100, 250, 500, 1000, 3000}

type RequestMetrics struct {
	mu         sync.Mutex
	startedAt  time.Time
	requests   uint64
	status2xx  uint64
	status3xx  uint64
	status4xx  uint64
	status5xx  uint64
	latencySum float64
	buckets    [len(latencyBucketUpperMs) + 1]uint64
}

var DefaultRequestMetrics = &RequestMetrics{startedAt: time.Now()}

// RecordHTTP stores only aggregate counters and a bounded latency histogram.
// It never retains the request path, query string, headers, or payload.
func RecordHTTP(_ string, rawPath string, status int, elapsed time.Duration) {
	if isMonitorRequest(rawPath) {
		return
	}
	if DefaultRequestMetrics == nil {
		return
	}
	m := DefaultRequestMetrics
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests++
	switch {
	case status >= 200 && status < 300:
		m.status2xx++
	case status >= 300 && status < 400:
		m.status3xx++
	case status >= 400 && status < 500:
		m.status4xx++
	case status >= 500:
		m.status5xx++
	}
	latencyMs := float64(elapsed) / float64(time.Millisecond)
	m.latencySum += latencyMs
	for index, upper := range latencyBucketUpperMs {
		if latencyMs <= upper {
			m.buckets[index]++
			return
		}
	}
	m.buckets[len(m.buckets)-1]++
}

func (m *RequestMetrics) Take(now time.Time) port.MonitorHTTPMetrics {
	if m == nil {
		return port.MonitorHTTPMetrics{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	seconds := now.Sub(m.startedAt).Seconds()
	if seconds <= 0 {
		seconds = 1
	}
	result := port.MonitorHTTPMetrics{
		Requests: m.requests, Status2xx: m.status2xx, Status3xx: m.status3xx,
		Status4xx: m.status4xx, Status5xx: m.status5xx, SampleSeconds: seconds,
		LatencyBuckets: append([]uint64(nil), m.buckets[:]...),
	}
	if result.Requests > 0 {
		result.AvgLatencyMs = m.latencySum / float64(result.Requests)
		result.P95LatencyMs = histogramPercentile(result.LatencyBuckets, 0.95)
	}
	m.requests, m.status2xx, m.status3xx, m.status4xx, m.status5xx = 0, 0, 0, 0, 0
	m.latencySum = 0
	clear(m.buckets[:])
	m.startedAt = now
	return result
}

func histogramPercentile(counts []uint64, percentile float64) float64 {
	var total uint64
	for _, count := range counts {
		total += count
	}
	if total == 0 {
		return 0
	}
	target := uint64(float64(total)*percentile + 0.999999)
	var accumulated uint64
	for index, count := range counts {
		accumulated += count
		if accumulated >= target {
			if index < len(latencyBucketUpperMs) {
				return latencyBucketUpperMs[index]
			}
			return latencyBucketUpperMs[len(latencyBucketUpperMs)-1]
		}
	}
	return 0
}

func isMonitorRequest(path string) bool {
	return path == "/healthz" || path == "/readyz" || strings.HasPrefix(path, "/v1/admin/monitor/health")
}
