package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func (s *MyContentAnalyticsService) ListContinuationTargets(ctx context.Context, rangeValue, sortBy string, sourceArticleID, current, size int) model.ResultVO {
	if current < 1 {
		current = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	window := resolveContentAnalyticsRange(rangeValue, timeNow())
	rows, err := s.repo.ListContinuationTargetMetrics(ctx, window.Start.Format("2006-01-02"), window.End.Format("2006-01-02"), sourceArticleID)
	if err != nil {
		return model.ResultFromError(err)
	}
	items := make([]model.ContentContinuationTargetDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.ContentContinuationTargetDTO{
			RowKey:             fmt.Sprintf("%d-%s-%d-%s-%d", row.SourceArticleId, row.TargetType, row.TargetId, row.Placement, row.Position),
			SourceArticleID:    row.SourceArticleId,
			SourceArticleTitle: row.SourceArticleTitle,
			TargetType:         string(row.TargetType),
			TargetID:           row.TargetId,
			TargetTitle:        row.TargetTitle,
			Placement:          string(row.Placement),
			Position:           row.Position,
			Clicks:             row.Clicks,
			ModuleImpressions:  row.ModuleImpressions,
			ClickRate:          contentPercentage(row.Clicks, row.ModuleImpressions),
		})
	}
	sortContinuationTargets(items, sortBy)
	start := (current - 1) * size
	if start > len(items) {
		start = len(items)
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: items[start:end], Count: len(items)})
}

func (s *MyContentAnalyticsService) GetStudioAnalytics(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	if s.studio == nil {
		return model.ResultFailWithMessage("运营统计暂不可用")
	}
	rangeValue := strings.TrimSpace(c.Query("range"))
	if rangeValue != "7d" && rangeValue != "90d" {
		rangeValue = "30d"
	}
	window := resolveContentAnalyticsRange(rangeValue, timeNow())
	endExclusive := window.End.AddDate(0, 0, 1)
	summary, err := s.studio.StudioOperationsSummary(c.Request.Context(), user.UserInfoId, window.Start, endExclusive)
	if err != nil {
		return model.ResultFromError(err)
	}
	trend, err := s.studio.StudioAnalyticsTrend(c.Request.Context(), user.UserInfoId, window.Start, window.End)
	if err != nil {
		return model.ResultFromError(err)
	}
	topArticles, err := s.studio.StudioTopArticles(c.Request.Context(), user.UserInfoId, window.Start, window.End, 5)
	if err != nil {
		return model.ResultFromError(err)
	}
	totals := port.ContentDailyMetric{}
	for index := range trend {
		totals.Views += trend[index].Views
		totals.EffectiveSessions += trend[index].EffectiveSessions
		totals.TotalActiveMs += trend[index].TotalActiveMs
		totals.CompletedSessions += trend[index].CompletedSessions
		totals.SeriesImpressions += trend[index].SeriesImpressions
		totals.SeriesClicks += trend[index].SeriesClicks
		totals.RelatedImpressions += trend[index].RelatedImpressions
		totals.RelatedClicks += trend[index].RelatedClicks
	}
	for index := range trend {
		totals.UniqueReaders += trend[index].UniqueReaders
		if s.cache != nil {
			key := contentUniqueReadersAuthorPrefix + strconv.Itoa(user.UserInfoId) + ":" + trend[index].Date
			if count, countErr := s.cache.PFCount(c.Request.Context(), key); countErr == nil && count > 0 {
				trend[index].UniqueReaders = count
			}
		}
	}
	totals.UniqueReaders = s.uniqueReadersForAuthorDays(c.Request.Context(), user.UserInfoId, window.Days, totals.UniqueReaders)
	return model.ResultOkWithData(model.StudioAnalyticsDTO{
		Range: rangeValue, Unit: window.Unit, Operations: summary, Performance: overviewFromContentMetrics(totals), Trend: trend,
		TopArticles: topArticles, GeneratedAt: timeNow(),
	})
}

func (s *MyContentAnalyticsService) GetStudioCalendar(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	if s.studio == nil {
		return model.ResultFailWithMessage("运营日历暂不可用")
	}
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(c.Query("start")))
	if err != nil {
		return model.ResultFailWithMessage("开始时间格式不正确")
	}
	end, err := time.Parse(time.RFC3339, strings.TrimSpace(c.Query("end")))
	if err != nil || !end.After(start) || end.Sub(start) > 62*24*time.Hour {
		return model.ResultFailWithMessage("日历时间范围不正确")
	}
	events, err := s.studio.StudioCalendar(c.Request.Context(), user.UserInfoId, start, end)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.StudioCalendarDTO{Events: events})
}

func (s *MyContentAnalyticsService) GetContentAnalytics(ctx context.Context, rangeValue string) model.ResultVO {
	window := resolveContentAnalyticsRange(rangeValue, timeNow())
	rows, err := s.repo.ListDailyMetrics(ctx, window.Start.Format("2006-01-02"), window.End.Format("2006-01-02"))
	if err != nil {
		return model.ResultFromError(err)
	}
	byDate := contentMetricsByDate(rows)
	trend := s.contentTrend(ctx, window, byDate, contentUniqueReadersAllPrefix)
	overview := overviewFromContentMetrics(contentTotals(rows))
	overview.UniqueReaders = s.uniqueReadersForDays(ctx, contentUniqueReadersAllPrefix, window.Days, byDate)
	return model.ResultOkWithData(model.ContentAnalyticsDTO{
		Range:       window.Value,
		Unit:        window.Unit,
		Overview:    overview,
		Trend:       trend,
		GeneratedAt: timeNow(),
	})
}

func (s *MyContentAnalyticsService) ListContentAnalyticsArticles(ctx context.Context, rangeValue, sortBy string, current, size int) model.ResultVO {
	if current < 1 {
		current = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	window := resolveContentAnalyticsRange(rangeValue, timeNow())
	rows, err := s.repo.ListArticleMetrics(ctx, window.Start.Format("2006-01-02"), window.End.Format("2006-01-02"))
	if err != nil {
		return model.ResultFromError(err)
	}
	items := make([]model.ContentArticlePerformanceDTO, 0, len(rows))
	for _, row := range rows {
		item := articlePerformanceDTO(row)
		item.UniqueReaders = s.uniqueReadersForArticleDays(ctx, row.ArticleId, window.Days, contentDailyFallback(rows, row.ArticleId))
		items = append(items, item)
	}
	sortContentArticles(items, sortBy)
	start := (current - 1) * size
	if start > len(items) {
		start = len(items)
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	records := items[start:end]
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: len(items)})
}

func (s *MyContentAnalyticsService) GetContentAnalyticsArticle(ctx context.Context, articleID int, rangeValue string) model.ResultVO {
	if articleID <= 0 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	article, err := s.articles.GetArticleRecord(ctx, articleID)
	if err != nil {
		return model.ResultFromError(err)
	}
	if article.Id == 0 {
		return model.ResultFailWithMessage("文章不存在")
	}
	window := resolveContentAnalyticsRange(rangeValue, timeNow())
	rows, err := s.repo.GetArticleDailyMetrics(ctx, articleID, window.Start.Format("2006-01-02"), window.End.Format("2006-01-02"))
	if err != nil {
		return model.ResultFromError(err)
	}
	byDate := contentMetricsByDate(rows)
	overview := overviewFromContentMetrics(contentTotals(rows))
	overview.UniqueReaders = s.uniqueReadersForArticleDays(ctx, articleID, window.Days, contentTotals(rows).UniqueReaders)
	return model.ResultOkWithData(model.ContentAnalyticsArticleDetailDTO{
		ArticleID:    article.Id,
		ArticleTitle: article.ArticleTitle,
		ArticleCover: article.ArticleCover,
		CreateTime:   article.CreateTime,
		Overview:     overview,
		Trend:        s.contentTrend(ctx, window, byDate, contentUniqueReadersArticlePrefix+strconv.Itoa(articleID)+":"),
	})
}
