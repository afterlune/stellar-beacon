package service

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

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
	if article.UserId > 0 {
		authorKey := contentUniqueReadersAuthorPrefix + strconv.Itoa(article.UserId) + ":" + now.Format("2006-01-02")
		if _, err := s.cache.PFAdd(c.Request.Context(), authorKey, fingerprintHash); err != nil {
			s.removeSessionKey(c.Request.Context(), sessionKey)
			return model.ResultFromError(err)
		}
		if _, err := s.cache.Expire(c.Request.Context(), authorKey, contentReadSessionTTL); err != nil {
			slog.WarnContext(c.Request.Context(), "expire author reader hyperloglog failed", "authorId", article.UserId, "error", err)
		}
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
