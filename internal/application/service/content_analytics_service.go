package service

import (
	"context"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

const (
	contentUniqueReadersAllPrefix       = "content:unique_readers:all:"
	contentUniqueReadersArticlePrefix   = "content:unique_readers:article:"
	contentUniqueReadersAuthorPrefix    = "content:unique_readers:author:"
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
	GetStudioAnalytics(c *gin.Context) model.ResultVO
	GetStudioCalendar(c *gin.Context) model.ResultVO
}

type MyContentAnalyticsService struct {
	repo     port.ContentAnalyticsRepository
	articles port.ArticleRepository
	studio   port.StudioOperationsRepository
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
		studio:   deps.Studio,
		cache:    deps.Cache,
		visitor:  deps.Visitor,
		limiter:  deps.Limiter,
	}, nil
}
