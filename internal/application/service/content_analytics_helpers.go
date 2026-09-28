package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
)

func (s *MyContentAnalyticsService) contentTrend(ctx context.Context, window contentAnalyticsRange, byDate map[string]port.ContentDailyMetric, prefix string) []model.ContentAnalyticsTrendDTO {
	trend := make([]model.ContentAnalyticsTrendDTO, 0, len(window.Buckets))
	for _, bucket := range window.Buckets {
		totals := contentTotalsForDays(byDate, bucket.Days)
		overview := overviewFromContentMetrics(totals)
		overview.UniqueReaders = s.uniqueReadersForDays(ctx, prefix, bucket.Days, byDate)
		trend = append(trend, model.ContentAnalyticsTrendDTO{
			Period:            bucket.Label,
			Views:             overview.Views,
			UniqueReaders:     overview.UniqueReaders,
			EffectiveSessions: overview.EffectiveSessions,
			AvgActiveMs:       overview.AvgActiveMs,
			CompletionRate:    overview.CompletionRate,
			Continuation:      overview.Continuation,
		})
	}
	return trend
}

func (s *MyContentAnalyticsService) uniqueReadersForDays(ctx context.Context, prefix string, days []time.Time, byDate map[string]port.ContentDailyMetric) int64 {
	keys := make([]string, 0, len(days))
	var fallback int64
	for _, day := range days {
		keys = append(keys, prefix+day.Format("2006-01-02"))
		fallback += byDate[day.Format("2006-01-02")].UniqueReaders
	}
	if len(keys) == 0 {
		return 0
	}
	value, err := s.cache.PFCount(ctx, keys...)
	if err != nil {
		slog.WarnContext(ctx, "count unique content readers failed", "error", err)
		return fallback
	}
	return value
}

func (s *MyContentAnalyticsService) uniqueReadersForArticleDays(ctx context.Context, articleID int, days []time.Time, fallback int64) int64 {
	keys := make([]string, 0, len(days))
	for _, day := range days {
		keys = append(keys, contentUniqueReadersArticlePrefix+strconv.Itoa(articleID)+":"+day.Format("2006-01-02"))
	}
	if len(keys) == 0 {
		return fallback
	}
	value, err := s.cache.PFCount(ctx, keys...)
	if err != nil {
		slog.WarnContext(ctx, "count article unique readers failed", "articleId", articleID, "error", err)
		return fallback
	}
	return value
}

func (s *MyContentAnalyticsService) uniqueReadersForAuthorDays(ctx context.Context, userID int, days []time.Time, fallback int64) int64 {
	if userID <= 0 || len(days) == 0 || s.cache == nil {
		return fallback
	}
	keys := make([]string, 0, len(days))
	for _, day := range days {
		keys = append(keys, contentUniqueReadersAuthorPrefix+strconv.Itoa(userID)+":"+day.Format("2006-01-02"))
	}
	value, err := s.cache.PFCount(ctx, keys...)
	if err != nil {
		slog.WarnContext(ctx, "count author unique readers failed", "authorId", userID, "error", err)
		return fallback
	}
	if value == 0 && fallback > 0 {
		return fallback
	}
	return value
}

func (s *MyContentAnalyticsService) removeSessionKey(ctx context.Context, sessionKey string) {
	if err := s.cache.Delete(ctx, contentReadSessionIdempotencyPrefix+sessionKey); err != nil {
		slog.WarnContext(ctx, "release read session idempotency key failed", "error", err)
	}
}

func validReadSessionID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 8 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':' {
			continue
		}
		return false
	}
	return true
}

func hashContentValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type contentAnalyticsRange struct {
	Value   string
	Unit    string
	Start   time.Time
	End     time.Time
	Days    []time.Time
	Buckets []contentAnalyticsBucket
}

type contentAnalyticsBucket struct {
	Label string
	Days  []time.Time
}

func resolveContentAnalyticsRange(rangeValue string, now time.Time) contentAnalyticsRange {
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	result := contentAnalyticsRange{Value: "7d", Unit: "day", End: now}
	switch rangeValue {
	case "30d":
		result.Value = "30d"
		result.Start = now.AddDate(0, 0, -29)
	case "90d":
		result.Value = "90d"
		result.Start = now.AddDate(0, 0, -89)
	case "12m":
		result.Value = "12m"
		result.Unit = "month"
		result.Start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -11, 0)
	default:
		result.Start = now.AddDate(0, 0, -6)
	}
	for cursor := result.Start; !cursor.After(result.End); cursor = cursor.AddDate(0, 0, 1) {
		result.Days = append(result.Days, cursor)
	}
	if result.Unit == "day" {
		for _, day := range result.Days {
			result.Buckets = append(result.Buckets, contentAnalyticsBucket{Label: day.Format("2006-01-02"), Days: []time.Time{day}})
		}
		return result
	}
	for cursor := result.Start; !cursor.After(result.End); cursor = cursor.AddDate(0, 1, 0) {
		monthEnd := cursor.AddDate(0, 1, -1)
		if monthEnd.After(result.End) {
			monthEnd = result.End
		}
		days := make([]time.Time, 0, 31)
		for day := cursor; !day.After(monthEnd); day = day.AddDate(0, 0, 1) {
			days = append(days, day)
		}
		result.Buckets = append(result.Buckets, contentAnalyticsBucket{Label: cursor.Format("2006-01"), Days: days})
	}
	return result
}

func contentMetricsByDate(rows []port.ContentDailyMetric) map[string]port.ContentDailyMetric {
	result := make(map[string]port.ContentDailyMetric, len(rows))
	for _, row := range rows {
		result[row.Date] = row
	}
	return result
}

func contentTotals(rows []port.ContentDailyMetric) port.ContentDailyMetric {
	var totals port.ContentDailyMetric
	for _, row := range rows {
		totals.Views += row.Views
		totals.UniqueReaders += row.UniqueReaders
		totals.EffectiveSessions += row.EffectiveSessions
		totals.TotalActiveMs += row.TotalActiveMs
		totals.CompletedSessions += row.CompletedSessions
		totals.SeriesImpressions += row.SeriesImpressions
		totals.SeriesClicks += row.SeriesClicks
		totals.RelatedImpressions += row.RelatedImpressions
		totals.RelatedClicks += row.RelatedClicks
	}
	return totals
}

func contentTotalsForDays(byDate map[string]port.ContentDailyMetric, days []time.Time) port.ContentDailyMetric {
	var totals port.ContentDailyMetric
	for _, day := range days {
		row := byDate[day.Format("2006-01-02")]
		totals.Views += row.Views
		totals.UniqueReaders += row.UniqueReaders
		totals.EffectiveSessions += row.EffectiveSessions
		totals.TotalActiveMs += row.TotalActiveMs
		totals.CompletedSessions += row.CompletedSessions
		totals.SeriesImpressions += row.SeriesImpressions
		totals.SeriesClicks += row.SeriesClicks
		totals.RelatedImpressions += row.RelatedImpressions
		totals.RelatedClicks += row.RelatedClicks
	}
	return totals
}

func overviewFromContentMetrics(metric port.ContentDailyMetric) model.ContentAnalyticsOverviewDTO {
	avgActiveMs := int64(0)
	if metric.EffectiveSessions > 0 {
		avgActiveMs = metric.TotalActiveMs / metric.EffectiveSessions
	}
	completionRate := float64(0)
	if metric.EffectiveSessions > 0 {
		completionRate = math.Round(float64(metric.CompletedSessions)*10000/float64(metric.EffectiveSessions)) / 100
	}
	return model.ContentAnalyticsOverviewDTO{
		Views:             metric.Views,
		UniqueReaders:     metric.UniqueReaders,
		EffectiveSessions: metric.EffectiveSessions,
		AvgActiveMs:       avgActiveMs,
		CompletionRate:    completionRate,
		Continuation:      contentContinuationMetrics(metric),
	}
}

func contentContinuationMetrics(metric port.ContentDailyMetric) model.ContentContinuationMetricsDTO {
	return model.ContentContinuationMetricsDTO{
		SeriesImpressions:  metric.SeriesImpressions,
		SeriesClicks:       metric.SeriesClicks,
		SeriesClickRate:    contentPercentage(metric.SeriesClicks, metric.SeriesImpressions),
		RelatedImpressions: metric.RelatedImpressions,
		RelatedClicks:      metric.RelatedClicks,
		RelatedClickRate:   contentPercentage(metric.RelatedClicks, metric.RelatedImpressions),
		ContinuationRate:   contentPercentage(metric.SeriesClicks+metric.RelatedClicks, metric.SeriesImpressions+metric.RelatedImpressions),
	}
}

func contentPercentage(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return math.Round(float64(numerator)*10000/float64(denominator)) / 100
}

func articlePerformanceDTO(row port.ContentArticleMetric) model.ContentArticlePerformanceDTO {
	overview := overviewFromContentMetrics(port.ContentDailyMetric{
		Views:              row.Views,
		UniqueReaders:      row.UniqueReaders,
		EffectiveSessions:  row.EffectiveSessions,
		TotalActiveMs:      row.TotalActiveMs,
		CompletedSessions:  row.CompletedSessions,
		SeriesImpressions:  row.SeriesImpressions,
		SeriesClicks:       row.SeriesClicks,
		RelatedImpressions: row.RelatedImpressions,
		RelatedClicks:      row.RelatedClicks,
	})
	return model.ContentArticlePerformanceDTO{
		ArticleID:         row.ArticleId,
		ArticleTitle:      row.ArticleTitle,
		ArticleCover:      row.ArticleCover,
		CategoryName:      row.CategoryName,
		CreateTime:        row.CreateTime,
		Views:             overview.Views,
		UniqueReaders:     overview.UniqueReaders,
		EffectiveSessions: overview.EffectiveSessions,
		AvgActiveMs:       overview.AvgActiveMs,
		CompletionRate:    overview.CompletionRate,
		Continuation:      overview.Continuation,
	}
}

func contentDailyFallback(rows []port.ContentArticleMetric, articleID int) int64 {
	for _, row := range rows {
		if row.ArticleId == articleID {
			return row.UniqueReaders
		}
	}
	return 0
}

func sortContentArticles(items []model.ContentArticlePerformanceDTO, sortBy string) {
	less := func(i, j int) bool { return items[i].Views > items[j].Views }
	switch strings.TrimSpace(sortBy) {
	case "uniqueReaders":
		less = func(i, j int) bool { return items[i].UniqueReaders > items[j].UniqueReaders }
	case "effectiveSessions":
		less = func(i, j int) bool { return items[i].EffectiveSessions > items[j].EffectiveSessions }
	case "avgActiveMs":
		less = func(i, j int) bool { return items[i].AvgActiveMs > items[j].AvgActiveMs }
	case "completionRate":
		less = func(i, j int) bool { return items[i].CompletionRate > items[j].CompletionRate }
	case "continuationRate":
		less = func(i, j int) bool {
			return items[i].Continuation.ContinuationRate > items[j].Continuation.ContinuationRate
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if less(i, j) {
			return true
		}
		if less(j, i) {
			return false
		}
		return items[i].ArticleID > items[j].ArticleID
	})
}

func sortContinuationTargets(items []model.ContentContinuationTargetDTO, sortBy string) {
	less := func(i, j int) bool {
		if items[i].Clicks != items[j].Clicks {
			return items[i].Clicks > items[j].Clicks
		}
		if items[i].ClickRate != items[j].ClickRate {
			return items[i].ClickRate > items[j].ClickRate
		}
		return items[i].TargetID > items[j].TargetID
	}
	if strings.TrimSpace(sortBy) == "clickRate" {
		less = func(i, j int) bool {
			if items[i].ClickRate != items[j].ClickRate {
				return items[i].ClickRate > items[j].ClickRate
			}
			if items[i].Clicks != items[j].Clicks {
				return items[i].Clicks > items[j].Clicks
			}
			return items[i].TargetID > items[j].TargetID
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if less(i, j) {
			return true
		}
		if less(j, i) {
			return false
		}
		if items[i].SourceArticleID != items[j].SourceArticleID {
			return items[i].SourceArticleID > items[j].SourceArticleID
		}
		return items[i].TargetID > items[j].TargetID
	})
}

var _ ContentAnalyticsService = (*MyContentAnalyticsService)(nil)
