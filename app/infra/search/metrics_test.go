package search

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestSearchMetricsObserverAggregatesModesLatencyAndCancellation(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	alerts := make([]SearchAlert, 0, 1)
	observer := NewSearchMetricsObserverWithConfig(SearchMetricsObserverConfig{
		Capacity:           3,
		MinimumSamples:     2,
		ErrorRateThreshold: 0.5,
		AlertCooldown:      time.Hour,
		Now:                func() time.Time { return now },
		OnAlert:            func(alert SearchAlert) { alerts = append(alerts, alert) },
	})

	observer.Observe(context.Background(), "articles", port.SearchModeKeyword, 100*time.Millisecond, nil)
	observer.Observe(context.Background(), "articles", port.SearchModeKeyword, 200*time.Millisecond, errors.New("search unavailable"))
	observer.Observe(context.Background(), "article_chunks_v2", port.SearchModeSemantic, 300*time.Millisecond, context.Canceled)

	snapshot := observer.Snapshot()
	if snapshot.Total != 3 || snapshot.Succeeded != 1 || snapshot.Failed != 1 || snapshot.Canceled != 1 || snapshot.ErrorRate != 2.0/3.0 {
		t.Fatalf("search metrics = %#v", snapshot)
	}
	if snapshot.DurationP95MS != 300 || len(snapshot.Routes) != 2 {
		t.Fatalf("search latency/routes = %#v", snapshot)
	}
	if len(alerts) != 1 || alerts[0].Index != "articles" || alerts[0].Failed != 1 {
		t.Fatalf("search alerts = %#v", alerts)
	}
}

func TestSearchMetricsObserverKeepsOnlyBoundedDurations(t *testing.T) {
	observer := NewSearchMetricsObserver(2)
	observer.Observe(context.Background(), "articles", port.SearchModeKeyword, time.Second, nil)
	observer.Observe(context.Background(), "articles", port.SearchModeKeyword, 2*time.Second, nil)
	observer.Observe(context.Background(), "articles", port.SearchModeKeyword, 3*time.Second, nil)

	snapshot := observer.Snapshot()
	if snapshot.Total != 3 || snapshot.DurationP95MS != 3000 {
		t.Fatalf("bounded search metrics = %#v", snapshot)
	}
	if len(observer.recent) != 2 || observer.recent[0].Duration != 2*time.Second || observer.recent[1].Duration != 3*time.Second {
		t.Fatalf("recent search window = %#v", observer.recent)
	}
}

func TestSearchMetricsObserverCapsRouteCardinality(t *testing.T) {
	observer := NewSearchMetricsObserver(1)
	for index := 0; index < maxSearchMetricRoutes+10; index++ {
		observer.Observe(context.Background(), "index-"+strconv.Itoa(index), port.SearchModeKeyword, time.Millisecond, nil)
	}

	snapshot := observer.Snapshot()
	if snapshot.Total != maxSearchMetricRoutes+10 || len(snapshot.Routes) != maxSearchMetricRoutes {
		t.Fatalf("search route cardinality = total=%d routes=%d", snapshot.Total, len(snapshot.Routes))
	}
}
