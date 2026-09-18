package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

const (
	contentUniqueReadersAllPrefix       = "content:unique_readers:all:"
	contentUniqueReadersArticlePrefix   = "content:unique_readers:article:"
	contentReadSessionIdempotencyPrefix = "content:read-session:"
	contentReadSessionTTL               = 400 * 24 * time.Hour
	contentReadSessionIdempotencyTTL    = 48 * time.Hour
	minEffectiveReadMs                  = 3000
	maxReadSessionMs                    = 2 * 60 * 60 * 1000
)

type ContentAnalyticsService interface {
	TrackReadSession(c *gin.Context) model.ResultVO
	TrackContinuationEvent(c *gin.Context) model.ResultVO
	GetContentAnalytics(ctx context.Context, rangeValue string) model.ResultVO
	ListContentAnalyticsArticles(ctx context.Context, rangeValue, sortBy string, current, size int) model.ResultVO
	ListContinuationTargets(ctx context.Context, rangeValue, sortBy string, sourceArticleID, current, size int) model.ResultVO
	GetContentAnalyticsArticle(ctx context.Context, articleID int, rangeValue string) model.ResultVO
}

type MyContentAnalyticsService struct {
	repo     port.ContentAnalyticsRepository
	articles port.ArticleRepository
	cache    port.Cache
	visitor  port.VisitorResolver
	limiter  port.RateLimiter
}

func NewContentAnalyticsService(deps ContentAnalyticsServiceDeps) (*MyContentAnalyticsService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyContentAnalyticsService{
		repo:     deps.Repo,
		articles: deps.Articles,
		cache:    deps.Cache,
		visitor:  deps.Visitor,
		limiter:  deps.Limiter,
	}, nil
}

func (s *MyContentAnalyticsService) TrackReadSession(c *gin.Context) model.ResultVO {
	articleID, err := strconv.Atoi(c.Param("articleId"))
	if err != nil || articleID <= 0 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.ArticleReadSessionVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if !validReadSessionID(vo.SessionID) {
		return model.ResultFailWithMessage("阅读会话标识不正确")
	}
	if vo.ActiveMs < minEffectiveReadMs || vo.ActiveMs > maxReadSessionMs {
		return model.ResultFailWithMessage("阅读时长不正确")
	}
	if math.IsNaN(vo.MaxScrollPercent) || math.IsInf(vo.MaxScrollPercent, 0) || vo.MaxScrollPercent < 0 || vo.MaxScrollPercent > 100 {
		return model.ResultFailWithMessage("阅读进度不正确")
	}

	article, err := s.articles.GetArticleRecord(c.Request.Context(), articleID)
	if err != nil {
		return model.ResultFromError(err)
	}
	if article.Id == 0 || article.IsDelete != 0 || article.Status != 1 {
		return model.ResultFailWithMessage("文章不存在")
	}

	identity, err := s.visitor.Resolve(c.Request.Context(), c.Request)
	if err != nil {
		return model.ResultFailWithMessage("无法识别阅读会话")
	}
	if identity.IsBot {
		return model.ResultOk()
	}
	readerKey := strings.TrimSpace(identity.Fingerprint)
	if readerKey == "" {
		readerKey = c.ClientIP() + "\x00" + c.Request.UserAgent()
	}
	if s.limiter != nil {
		allowed, limitErr := allowRateLimit(c.Request.Context(), s.limiter, "content:read-session:", readerKey, 120, time.Minute)
		if limitErr != nil {
			return model.ResultFromError(limitErr)
		}
		if !allowed {
			return model.ResultFailWithCodeAndMessage(42900, "请求过于频繁，请稍后再试")
		}
	}

	sessionKey := hashContentValue(strconv.Itoa(articleID) + "\x00" + strings.TrimSpace(vo.SessionID))
	created, err := s.cache.SetNX(c.Request.Context(), contentReadSessionIdempotencyPrefix+sessionKey, "1", contentReadSessionIdempotencyTTL)
	if err != nil {
		return model.ResultFromError(err)
	}
	if !created {
		// SetNX returns false when the same browser report is retried. The
		// existing aggregate already contains the session.
		return model.ResultOk()
	}

	fingerprintHash := hashContentValue(readerKey)
	now := timeNow()
	articleKey := contentUniqueReadersArticlePrefix + strconv.Itoa(articleID) + ":" + now.Format("2006-01-02")
	allKey := contentUniqueReadersAllPrefix + now.Format("2006-01-02")
	if _, err := s.cache.PFAdd(c.Request.Context(), articleKey, fingerprintHash); err != nil {
		s.removeSessionKey(c.Request.Context(), sessionKey)
		return model.ResultFromError(err)
	}
	if _, err := s.cache.PFAdd(c.Request.Context(), allKey, fingerprintHash); err != nil {
		s.removeSessionKey(c.Request.Context(), sessionKey)
		return model.ResultFromError(err)
	}
	if _, err := s.cache.Expire(c.Request.Context(), articleKey, contentReadSessionTTL); err != nil {
		slog.WarnContext(c.Request.Context(), "expire article reader hyperloglog failed", "error", err)
	}
	if _, err := s.cache.Expire(c.Request.Context(), allKey, contentReadSessionTTL); err != nil {
		slog.WarnContext(c.Request.Context(), "expire site reader hyperloglog failed", "error", err)
	}
	uniqueReaders, err := s.cache.PFCount(c.Request.Context(), articleKey)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "count article readers failed", "error", err)
		uniqueReaders = 0
	}
	if err := s.repo.RecordReadSession(c.Request.Context(), articleID, now, vo.ActiveMs, int(math.Round(vo.MaxScrollPercent)), uniqueReaders); err != nil {
		s.removeSessionKey(c.Request.Context(), sessionKey)
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyContentAnalyticsService) TrackContinuationEvent(c *gin.Context) model.ResultVO {
	articleID, err := strconv.Atoi(c.Param("articleId"))
	if err != nil || articleID <= 0 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.ArticleContinuationEventVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	eventType := port.ContinuationEventType(strings.TrimSpace(vo.EventType))
	if !eventType.Valid() {
		return model.ResultFailWithMessage("不支持的续读事件类型")
	}
	article, err := s.articles.GetArticleRecord(c.Request.Context(), articleID)
	if err != nil {
		return model.ResultFromError(err)
	}
	if article.Id == 0 || article.IsDelete != 0 || article.Status != 1 {
		return model.ResultFailWithMessage("文章不存在")
	}
	target, err := s.validateContinuationTarget(c.Request.Context(), article, eventType, vo)
	if err != nil {
		return model.ResultFromError(err)
	}
	identity, err := s.visitor.Resolve(c.Request.Context(), c.Request)
	if err != nil {
		return model.ResultFailWithMessage("无法识别续读事件")
	}
	if identity.IsBot {
		return model.ResultOk()
	}
	readerKey := strings.TrimSpace(identity.Fingerprint)
	if readerKey == "" {
		readerKey = c.ClientIP() + "\x00" + c.Request.UserAgent()
	}
	if s.limiter != nil {
		allowed, limitErr := allowRateLimit(c.Request.Context(), s.limiter, "content:continuation-event:", readerKey, 120, time.Minute)
		if limitErr != nil {
			return model.ResultFromError(limitErr)
		}
		if !allowed {
			return model.ResultFailWithCodeAndMessage(42900, "请求过于频繁，请稍后再试")
		}
	}
	if err := s.repo.RecordContinuationEvent(c.Request.Context(), articleID, timeNow(), eventType, target); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyContentAnalyticsService) validateContinuationTarget(ctx context.Context, article entity.TArticle, eventType port.ContinuationEventType, vo model.ArticleContinuationEventVO) (*port.ContinuationTarget, error) {
	if eventType.IsImpression() {
		if strings.TrimSpace(vo.TargetType) != "" || vo.TargetId != 0 || strings.TrimSpace(vo.Placement) != "" || vo.Position != 0 {
			return nil, apperrors.Invalid("content_analytics.continuation_target", "impression events must not carry a target")
		}
		return nil, nil
	}
	if !eventType.IsClick() {
		return nil, apperrors.Invalid("content_analytics.continuation_target", "unsupported continuation event")
	}
	target := &port.ContinuationTarget{
		Type:      port.ContinuationTargetType(strings.TrimSpace(vo.TargetType)),
		Id:        vo.TargetId,
		Placement: port.ContinuationPlacement(strings.TrimSpace(vo.Placement)),
		Position:  vo.Position,
	}
	if !target.Type.Valid() || target.Id <= 0 || target.Position < 0 {
		return nil, apperrors.Invalid("content_analytics.continuation_target", "invalid continuation target")
	}
	switch eventType {
	case port.ContinuationEventRelatedClick:
		if target.Type != port.ContinuationTargetArticle || target.Placement != port.ContinuationPlacementRelated || target.Position < 1 || target.Position > 3 {
			return nil, apperrors.Invalid("content_analytics.continuation_target", "invalid related continuation target")
		}
		related, err := s.articles.ListRelatedArticles(ctx, article.Id, article.CategoryId, article.SeriesId, 3)
		if err != nil {
			return nil, err
		}
		if len(related) < target.Position || related[target.Position-1] == nil || related[target.Position-1].Id != target.Id {
			return nil, apperrors.Invalid("content_analytics.continuation_target", "related target is no longer recommended")
		}
	case port.ContinuationEventSeriesClick:
		if article.SeriesId <= 0 {
			return nil, apperrors.Invalid("content_analytics.continuation_target", "source article has no series")
		}
		switch target.Placement {
		case port.ContinuationPlacementSeriesPrevious, port.ContinuationPlacementSeriesNext:
			if target.Type != port.ContinuationTargetArticle || target.Position != 0 {
				return nil, apperrors.Invalid("content_analytics.continuation_target", "invalid series article target")
			}
			articles, err := s.articles.ListArticleCardsBySeries(ctx, article.SeriesId)
			if err != nil {
				return nil, err
			}
			index := -1
			for i, candidate := range articles {
				if candidate != nil && candidate.Id == article.Id {
					index = i
					break
				}
			}
			if index < 0 {
				return nil, apperrors.Invalid("content_analytics.continuation_target", "source article is not in its series")
			}
			neighborIndex := index - 1
			if target.Placement == port.ContinuationPlacementSeriesNext {
				neighborIndex = index + 1
			}
			if neighborIndex < 0 || neighborIndex >= len(articles) || articles[neighborIndex] == nil || articles[neighborIndex].Id != target.Id || articles[neighborIndex].Status != 1 {
				return nil, apperrors.Invalid("content_analytics.continuation_target", "target is not the adjacent public article")
			}
		case port.ContinuationPlacementSeriesIndex:
			if target.Type != port.ContinuationTargetSeries || target.Id != article.SeriesId || target.Position != 0 {
				return nil, apperrors.Invalid("content_analytics.continuation_target", "invalid series index target")
			}
		default:
			return nil, apperrors.Invalid("content_analytics.continuation_target", "invalid series placement")
		}
	default:
		return nil, apperrors.Invalid("content_analytics.continuation_target", "unsupported continuation target")
	}
	return target, nil
}

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
