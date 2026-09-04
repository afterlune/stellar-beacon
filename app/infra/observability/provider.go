// Package observability composes sanitized infrastructure metrics for the
// application-facing operational diagnostics contract.
package observability

import (
	"context"
	"time"

	"benetnasch/app/domain/port"
)

type aiSnapshotter interface {
	OperationalSnapshot(context.Context) port.AIOperationalSnapshot
}

type searchSnapshotter interface {
	Snapshot() port.SearchMetricsSnapshot
}

// Provider is a read-only composition of the AI and search observers. It has
// no database or network access; external monitoring can scrape the exposed
// application endpoint on each instance and aggregate these snapshots.
type Provider struct {
	ai     aiSnapshotter
	search searchSnapshotter
	now    func() time.Time
}

func NewProvider(ai aiSnapshotter, search searchSnapshotter) *Provider {
	return &Provider{ai: ai, search: search, now: time.Now}
}

func (p *Provider) Snapshot(ctx context.Context) port.OperationalMetricsSnapshot {
	if ctx == nil {
		ctx = context.Background()
	}
	result := port.OperationalMetricsSnapshot{GeneratedAt: p.currentTime()}
	if p == nil {
		return result
	}
	if p.ai != nil {
		result.AI = p.ai.OperationalSnapshot(ctx)
	}
	if p.search != nil {
		result.Search = p.search.Snapshot()
	}
	return result
}

func (p *Provider) currentTime() time.Time {
	if p != nil && p.now != nil {
		if now := p.now(); !now.IsZero() {
			return now.UTC()
		}
	}
	return time.Now().UTC()
}

var _ port.OperationalMetricsProvider = (*Provider)(nil)
