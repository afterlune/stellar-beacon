package observability

import (
	"context"
	"reflect"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

type aiSnapshotFake struct {
	snapshot port.AIOperationalSnapshot
}

func (f aiSnapshotFake) OperationalSnapshot(context.Context) port.AIOperationalSnapshot {
	return f.snapshot
}

type searchSnapshotFake struct {
	snapshot port.SearchMetricsSnapshot
}

func (f searchSnapshotFake) Snapshot() port.SearchMetricsSnapshot {
	return f.snapshot
}

func TestProviderCombinesSanitizedAIAndSearchSnapshots(t *testing.T) {
	wantTime := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	wantAI := port.AIOperationalSnapshot{Enabled: true}
	wantSearch := port.SearchMetricsSnapshot{Total: 3, DurationP95MS: 42}
	provider := NewProvider(aiSnapshotFake{snapshot: wantAI}, searchSnapshotFake{snapshot: wantSearch})
	provider.now = func() time.Time { return wantTime }

	got := provider.Snapshot(context.Background())
	if !got.GeneratedAt.Equal(wantTime) || !reflect.DeepEqual(got.AI, wantAI) || !reflect.DeepEqual(got.Search, wantSearch) {
		t.Fatalf("operational snapshot = %#v", got)
	}
}

func TestProviderIsSafeWhenDependenciesAreNil(t *testing.T) {
	got := (*Provider)(nil).Snapshot(nil)
	if got.AI.Enabled || got.Search.Total != 0 || got.GeneratedAt.IsZero() {
		t.Fatalf("nil provider snapshot = %#v", got)
	}
}
