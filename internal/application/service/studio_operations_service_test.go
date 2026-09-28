package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
)

type fakeStudioOperationsRepository struct {
	userID   int
	summary  port.StudioOperationsSummary
	trend    []port.StudioAnalyticsTrendPoint
	top      []port.StudioTopArticle
	calendar []port.StudioCalendarEvent
}

func (f *fakeStudioOperationsRepository) StudioOperationsSummary(_ context.Context, userID int, _, _ time.Time) (port.StudioOperationsSummary, error) {
	f.userID = userID
	return f.summary, nil
}
func (f *fakeStudioOperationsRepository) StudioAnalyticsTrend(_ context.Context, userID int, _, _ time.Time) ([]port.StudioAnalyticsTrendPoint, error) {
	f.userID = userID
	return f.trend, nil
}
func (f *fakeStudioOperationsRepository) StudioTopArticles(_ context.Context, userID int, _, _ time.Time, _ int) ([]port.StudioTopArticle, error) {
	f.userID = userID
	return f.top, nil
}
func (f *fakeStudioOperationsRepository) StudioCalendar(_ context.Context, userID int, _, _ time.Time) ([]port.StudioCalendarEvent, error) {
	f.userID = userID
	return f.calendar, nil
}

func TestStudioAnalyticsScopesRepositoryAndAggregatesPerformance(t *testing.T) {
	repo := &fakeStudioOperationsRepository{
		summary: port.StudioOperationsSummary{PublishedArticles: 2, ScheduledArticles: 1},
		trend: []port.StudioAnalyticsTrendPoint{{
			Date: "2026-09-20", PublishedArticles: 2, Views: 40, UniqueReaders: 30,
			EffectiveSessions: 10, TotalActiveMs: 100000, CompletedSessions: 6,
			SeriesClicks: 2, SeriesImpressions: 10, RelatedClicks: 1, RelatedImpressions: 10,
		}},
		top: []port.StudioTopArticle{{ArticleID: 4, Title: "测试文章", Views: 40}},
	}
	svc := &MyContentAnalyticsService{studio: repo}
	result := svc.GetStudioAnalytics(platformTestContext(http.MethodGet, "/v1/studio/analytics?range=7d", ""))
	if !result.Flag {
		t.Fatalf("analytics failed: %+v", result)
	}
	data, ok := result.Data.(model.StudioAnalyticsDTO)
	if !ok {
		t.Fatalf("unexpected analytics type: %T", result.Data)
	}
	if repo.userID != 7 || data.Operations.PublishedArticles != 2 || data.Performance.Views != 40 || data.Performance.CompletionRate != 60 || data.Performance.Continuation.ContinuationRate != 15 {
		t.Fatalf("unexpected analytics: user=%d data=%+v", repo.userID, data)
	}
}

func TestStudioCalendarValidatesRangeAndScopesOwner(t *testing.T) {
	repo := &fakeStudioOperationsRepository{calendar: []port.StudioCalendarEvent{{ArticleID: 8, Title: "日历文章", State: "scheduled"}}}
	svc := &MyContentAnalyticsService{studio: repo}
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	result := svc.GetStudioCalendar(platformTestContext(http.MethodGet, "/v1/studio/calendar?start="+start+"&end="+end, ""))
	if !result.Flag || repo.userID != 7 {
		t.Fatalf("calendar failed: result=%+v user=%d", result, repo.userID)
	}
}
