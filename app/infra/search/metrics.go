package search

import (
	"context"
	stderrors "errors"
	"sort"
	"strings"
	"sync"
	"time"

	"benetnasch/app/domain/port"
)

const (
	defaultSearchMetricCapacity      = 128
	defaultSearchAlertMinimumSamples = uint64(20)
	defaultSearchAlertErrorThreshold = 0.20
	defaultSearchAlertCooldown       = 5 * time.Minute
	maxSearchMetricRoutes            = 64
)

// SearchAlert is a detail-free operational signal. It contains no query,
// filter, user identity, or provider response body.
type SearchAlert struct {
	Index     string
	Mode      port.SearchMode
	Total     uint64
	Failed    uint64
	ErrorRate float64
}

// SearchMetricsObserverConfig controls the bounded search observer. Alerts
// are optional so tests and local deployments can collect metrics without
// requiring a monitoring vendor.
type SearchMetricsObserverConfig struct {
	Capacity           int
	MinimumSamples     uint64
	ErrorRateThreshold float64
	AlertCooldown      time.Duration
	Now                func() time.Time
	OnAlert            func(SearchAlert)
}

type searchMetricObservation struct {
	Index     string
	Mode      port.SearchMode
	Duration  time.Duration
	Succeeded bool
	Canceled  bool
}

type searchRouteMetrics struct {
	Index     string
	Mode      port.SearchMode
	Total     uint64
	Succeeded uint64
	Failed    uint64
	Canceled  uint64
}

// SearchMetricsObserver records bounded recent durations and lifetime
// counters. The snapshot can be exported by the operational diagnostics
// provider and aggregated across application instances externally.
type SearchMetricsObserver struct {
	mu        sync.RWMutex
	capacity  int
	recent    []searchMetricObservation
	total     uint64
	succeeded uint64
	failed    uint64
	canceled  uint64
	routes    map[string]*searchRouteMetrics

	minimumSamples     uint64
	errorRateThreshold float64
	alertCooldown      time.Duration
	lastAlert          map[string]time.Time
	now                func() time.Time
	onAlert            func(SearchAlert)
}

func NewSearchMetricsObserver(capacity int) *SearchMetricsObserver {
	return NewSearchMetricsObserverWithConfig(SearchMetricsObserverConfig{Capacity: capacity})
}

func NewSearchMetricsObserverWithConfig(config SearchMetricsObserverConfig) *SearchMetricsObserver {
	if config.Capacity <= 0 {
		config.Capacity = defaultSearchMetricCapacity
	}
	if config.MinimumSamples == 0 {
		config.MinimumSamples = defaultSearchAlertMinimumSamples
	}
	if config.ErrorRateThreshold <= 0 || config.ErrorRateThreshold > 1 {
		config.ErrorRateThreshold = defaultSearchAlertErrorThreshold
	}
	if config.AlertCooldown <= 0 {
		config.AlertCooldown = defaultSearchAlertCooldown
	}
	return &SearchMetricsObserver{
		capacity:           config.Capacity,
		recent:             make([]searchMetricObservation, 0, config.Capacity),
		routes:             make(map[string]*searchRouteMetrics),
		minimumSamples:     config.MinimumSamples,
		errorRateThreshold: config.ErrorRateThreshold,
		alertCooldown:      config.AlertCooldown,
		lastAlert:          make(map[string]time.Time),
		now:                config.Now,
		onAlert:            config.OnAlert,
	}
}

// Observe records one completed search attempt. It is intentionally called
// at the infrastructure adapter edge, so both legacy and versioned indexes
// are measured without exposing query content to the observer.
func (o *SearchMetricsObserver) Observe(ctx context.Context, index string, mode port.SearchMode, duration time.Duration, err error) {
	if o == nil {
		return
	}
	if duration < 0 {
		duration = 0
	}
	now := o.currentTime()
	index = strings.TrimSpace(index)
	mode = port.SearchMode(strings.TrimSpace(string(mode)))
	canceled := stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded)
	succeeded := err == nil
	observation := searchMetricObservation{
		Index:     index,
		Mode:      mode,
		Duration:  duration,
		Succeeded: succeeded,
		Canceled:  canceled,
	}

	var alert *SearchAlert
	o.mu.Lock()
	o.total++
	if succeeded {
		o.succeeded++
	} else if canceled {
		o.canceled++
	} else {
		o.failed++
	}
	if o.capacity > 0 {
		if len(o.recent) == o.capacity {
			copy(o.recent, o.recent[1:])
			o.recent[len(o.recent)-1] = observation
		} else {
			o.recent = append(o.recent, observation)
		}
	}
	key := searchMetricKey(index, mode)
	route := o.routes[key]
	if route == nil {
		if len(o.routes) < maxSearchMetricRoutes {
			route = &searchRouteMetrics{Index: index, Mode: mode}
			o.routes[key] = route
		}
	}
	if route != nil {
		route.Total++
		if succeeded {
			route.Succeeded++
		} else if canceled {
			route.Canceled++
		} else {
			route.Failed++
		}
		if o.onAlert != nil && route.Total >= o.minimumSamples {
			errorCount := route.Failed + route.Canceled
			errorRate := ratioUint64(errorCount, route.Total)
			last := o.lastAlert[key]
			if errorRate >= o.errorRateThreshold && (last.IsZero() || now.Sub(last) >= o.alertCooldown) {
				o.lastAlert[key] = now
				alert = &SearchAlert{Index: route.Index, Mode: route.Mode, Total: route.Total, Failed: errorCount, ErrorRate: errorRate}
			}
		}
	}
	callback := o.onAlert
	o.mu.Unlock()
	if callback != nil && alert != nil {
		callback(*alert)
	}
}

func (o *SearchMetricsObserver) Snapshot() port.SearchMetricsSnapshot {
	if o == nil {
		return port.SearchMetricsSnapshot{Routes: []port.SearchRouteMetricsSnapshot{}}
	}
	o.mu.RLock()
	defer o.mu.RUnlock()

	keys := make([]string, 0, len(o.routes))
	for key := range o.routes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	routes := make([]port.SearchRouteMetricsSnapshot, 0, len(keys))
	for _, key := range keys {
		route := o.routes[key]
		routes = append(routes, port.SearchRouteMetricsSnapshot{
			Index:         route.Index,
			Mode:          route.Mode,
			Total:         route.Total,
			Succeeded:     route.Succeeded,
			Failed:        route.Failed,
			Canceled:      route.Canceled,
			ErrorRate:     ratioUint64(route.Failed+route.Canceled, route.Total),
			DurationP95MS: durationMilliseconds(searchPercentile95(searchDurations(o.recent, key))),
		})
	}
	return port.SearchMetricsSnapshot{
		Total:         o.total,
		Succeeded:     o.succeeded,
		Failed:        o.failed,
		Canceled:      o.canceled,
		ErrorRate:     ratioUint64(o.failed+o.canceled, o.total),
		DurationP95MS: durationMilliseconds(searchPercentile95(searchDurations(o.recent, ""))),
		Routes:        routes,
	}
}

func (o *SearchMetricsObserver) currentTime() time.Time {
	if o.now != nil {
		if now := o.now(); !now.IsZero() {
			return now.UTC()
		}
	}
	return time.Now().UTC()
}

func searchMetricKey(index string, mode port.SearchMode) string {
	return index + "\x00" + string(mode)
}

func searchDurations(observations []searchMetricObservation, key string) []time.Duration {
	result := make([]time.Duration, 0, len(observations))
	for _, observation := range observations {
		if key != "" && searchMetricKey(observation.Index, observation.Mode) != key {
			continue
		}
		result = append(result, observation.Duration)
	}
	return result
}

func searchPercentile95(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	index := (len(values)*95+99)/100 - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func ratioUint64(numerator, denominator uint64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func durationMilliseconds(value time.Duration) int64 {
	if value <= 0 {
		return 0
	}
	return value.Milliseconds()
}
