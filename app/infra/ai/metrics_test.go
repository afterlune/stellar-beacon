package ai

import (
	"context"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

func TestRunMetricsObserverKeepsBoundedRecentWindow(t *testing.T) {
	observer := NewRunMetricsObserver(2)
	observer.ObserveAIRun(context.Background(), port.AgentRun{ID: "run-1", Status: port.AIRunSucceeded})
	observer.ObserveAIRun(context.Background(), port.AgentRun{ID: "run-2", Status: port.AIRunFailed})
	observer.ObserveAIRun(context.Background(), port.AgentRun{ID: "run-3", Status: port.AIRunCanceled})

	snapshot := observer.Snapshot()
	if snapshot.Total != 3 || snapshot.Succeeded != 1 || snapshot.Failed != 1 || snapshot.Canceled != 1 {
		t.Fatalf("snapshot counters = %#v", snapshot)
	}
	if len(snapshot.Recent) != 2 || snapshot.Recent[0].ID != "run-2" || snapshot.Recent[1].ID != "run-3" {
		t.Fatalf("recent runs = %#v, want run-2 and run-3", snapshot.Recent)
	}
}

func TestRunMetricsObserverTracksProviderRatesLatencyTokensAndBudgetAlerts(t *testing.T) {
	now := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	var alerts []BudgetAlert
	observer := NewRunMetricsObserverWithConfig(RunMetricsObserverConfig{
		Capacity:         4,
		DailyTokenBudget: 100,
		Now:              func() time.Time { return now },
		OnBudgetAlert: func(alert BudgetAlert) {
			alerts = append(alerts, alert)
		},
	})
	observer.ObserveAIRun(context.Background(), port.AgentRun{
		ID: "run-success", UseCase: port.AIUseCaseChat, Provider: "openai", Model: "chat",
		Status: port.AIRunSucceeded, Usage: port.TokenUsage{InputTokens: 30, OutputTokens: 55, TotalTokens: 85},
		Duration: 100 * time.Millisecond, FirstTokenAt: now.Add(20 * time.Millisecond), FirstTokenLatency: 20 * time.Millisecond,
	})
	observer.ObserveAIRun(context.Background(), port.AgentRun{
		ID: "run-failed", UseCase: port.AIUseCaseWriting, Provider: "anthropic", Model: "writer",
		Status: port.AIRunFailed, Usage: port.TokenUsage{InputTokens: 10, OutputTokens: 10, TotalTokens: 20},
		Duration: 200 * time.Millisecond, FirstTokenAt: now.Add(50 * time.Millisecond), FirstTokenLatency: 50 * time.Millisecond,
	})

	snapshot := observer.Snapshot()
	if snapshot.Total != 2 || snapshot.Succeeded != 1 || snapshot.Failed != 1 || snapshot.ErrorRate != 0.5 || snapshot.SuccessRate != 0.5 {
		t.Fatalf("run counters = %#v", snapshot)
	}
	if snapshot.InputTokens != 40 || snapshot.OutputTokens != 65 || snapshot.TotalTokens != 105 || snapshot.DailyTokensUsed != 105 || snapshot.DailyTokensRemaining != 0 || !snapshot.DailyBudgetExceeded {
		t.Fatalf("token and budget metrics = %#v", snapshot)
	}
	if snapshot.DurationP95 != 200*time.Millisecond || snapshot.FirstTokenP95 != 50*time.Millisecond {
		t.Fatalf("latency metrics = %#v", snapshot)
	}
	if len(snapshot.Routes) != 2 || len(alerts) != 2 || alerts[0].Exceeded || !alerts[1].Exceeded {
		t.Fatalf("route or budget alerts = routes=%#v alerts=%#v", snapshot.Routes, alerts)
	}
	foundAnthropic := false
	for _, route := range snapshot.Routes {
		if route.Provider == "anthropic" {
			foundAnthropic = route.ErrorRate == 1
		}
	}
	if !foundAnthropic {
		t.Fatalf("route metrics are incorrect: %#v", snapshot.Routes)
	}

	now = now.Add(24 * time.Hour)
	newDay := observer.Snapshot()
	if newDay.DailyTokensUsed != 0 || newDay.DailyBudgetAlert || newDay.DailyBudgetExceeded {
		t.Fatalf("daily budget did not reset: %#v", newDay)
	}
}

func TestRunMetricsObserverTracksRateLimitedCallsSeparately(t *testing.T) {
	observer := NewRunMetricsObserver(4)
	observer.ObserveAIRun(context.Background(), port.AgentRun{
		UseCase: port.AIUseCaseChat, Provider: "openai", Model: "chat",
		Status: port.AIRunFailed, ErrorCode: string(apperrors.AICodeRateLimited),
	})

	snapshot := observer.Snapshot()
	if snapshot.Failed != 1 || snapshot.RateLimited != 1 || len(snapshot.Routes) != 1 || snapshot.Routes[0].RateLimited != 1 {
		t.Fatalf("rate limited metrics = %#v", snapshot)
	}
}
