package service

import (
	"testing"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

func TestDashboardGrowthDTOFillsSelectedPeriodsAndRates(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	result := dashboardGrowthDTO(
		port.NewsletterStats{Total: 8, Active: 4, Pending: 2, Unsubscribed: 2, Sent: 8, Failed: 2},
		[]port.GrowthTrend{{Period: "2026-09-02", SubscribeConfirms: 2, ShareClicks: 5}},
		[]port.NewsletterDeliveryTrend{{Period: "2026-09-03", Sent: 3, Failed: 1}},
		start, now, "day",
	)

	if result.Subscribers.ConfirmationRate != 66.66666666666666 {
		t.Fatalf("confirmation rate = %v", result.Subscribers.ConfirmationRate)
	}
	if result.Deliveries.SuccessRate != 80 {
		t.Fatalf("delivery success rate = %v, want 80", result.Deliveries.SuccessRate)
	}
	if len(result.Trend) != 3 {
		t.Fatalf("trend length = %d, want 3", len(result.Trend))
	}
	if result.Trend[0].Period != "2026-09-01" || result.Trend[0].SubscribeConfirms != 0 {
		t.Fatalf("first period was not zero-filled: %+v", result.Trend[0])
	}
	if result.Trend[1].ShareClicks != 5 || result.Trend[2].DeliverySent != 3 || result.Trend[2].DeliveryFailed != 1 {
		t.Fatalf("aggregated trend lost data: %+v", result.Trend)
	}
}

func TestDashboardGrowthDTOMonthPeriods(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC)
	result := dashboardGrowthDTO(port.NewsletterStats{}, nil, nil, start, now, "month")
	if len(result.Trend) != 3 {
		t.Fatalf("monthly trend length = %d, want 3", len(result.Trend))
	}
	if result.Trend[0].Period != "2026-01" || result.Trend[2].Period != "2026-03" {
		t.Fatalf("monthly periods = %+v", result.Trend)
	}
}
