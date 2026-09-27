package service

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"log/slog"
	"strconv"
	"time"

	"github.com/goccy/go-json"
)

// ArticleStatusScheduled marks an article that the publisher task releases once
// its scheduled_at passes.
const ArticleStatusScheduled = 4

type MyArticleService struct {
	repo             port.ArticleRepository
	reactions        port.ArticleReactionRepository
	contentAnalytics port.ContentAnalyticsRepository
	cache            port.Cache
	storage          port.ObjectStorage
	search           port.ArticleSearcher
	searchIndex      ArticleSearchMaintainer
	newsletter       port.NewsletterEnqueuer
}

func NewArticleService(deps ArticleServiceDeps) (*MyArticleService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyArticleService{
		repo:             deps.Repo,
		reactions:        deps.Reactions,
		contentAnalytics: deps.ContentAnalytics,
		cache:            deps.Cache,
		storage:          deps.Storage,
		search:           deps.Search,
		searchIndex:      deps.SearchIndex,
		newsletter:       deps.Newsletter,
	}, nil
}

func (a *MyArticleService) syncArticleSearch(ctx context.Context, articleIDs ...int) {
	if a.searchIndex == nil || len(articleIDs) == 0 {
		return
	}
	if err := a.searchIndex.Sync(ctx, articleIDs...); err != nil {
		slog.WarnContext(ctx, "sync article search index failed", "articleIds", articleIDs, "error", err)
	}
}

// isPubliclyCacheable reports whether an article body may live in the shared
// public cache. Only published, non-recycled articles qualify: drafts,
// scheduled releases and password-protected posts must always be resolved
// through the authenticated path.
func isPubliclyCacheable(article port.Article) bool {
	return article.IsDelete == 0 && article.Status == 1 && article.ModerationStatus != "hidden"
}

// cacheArticle stores a published body and evicts every other state so a stale
// public copy can never survive an unpublish, a recycle or a reschedule.
func (a *MyArticleService) cacheArticle(ctx context.Context, id, status, isDelete int, value any) {
	if a.cache == nil {
		return
	}
	key := strconv.Itoa(id)
	if isDelete != 0 || status != 1 {
		a.evictArticleCache(ctx, key)
		return
	}
	marshal, err := json.Marshal(value)
	if err != nil {
		slog.ErrorContext(ctx, "marshal article cache failed", "error", err)
		return
	}
	if err := a.cache.Set(ctx, key, marshal, time.Hour*1); err != nil {
		slog.WarnContext(ctx, "cache article failed", "error", err)
	}
}

func (a *MyArticleService) evictArticleCache(ctx context.Context, articleID string) {
	if a.cache == nil {
		return
	}
	if err := a.cache.Delete(ctx, articleID); err != nil {
		slog.WarnContext(ctx, "evict article cache failed", "error", err)
	}
}

func (a *MyArticleService) articleRepository() port.ArticleRepository {
	return a.repo
}

// attachCardReactionCounts decorates article cards with like/favourite totals.
// Counts are presentational: a ledger failure is logged and leaves the zero
// value instead of failing the whole read.
func (a *MyArticleService) attachCardReactionCounts(ctx context.Context, cards []*port.ArticleCard) {
	if a.reactions == nil || len(cards) == 0 {
		return
	}
	ids := make([]int, 0, len(cards))
	for _, card := range cards {
		if card != nil && card.Id > 0 {
			ids = append(ids, card.Id)
		}
	}
	counts, err := a.reactions.Counts(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "load article reaction counts failed", "error", err)
		return
	}
	for _, card := range cards {
		if card == nil {
			continue
		}
		totals := counts[card.Id]
		card.LikeCount = totals.LikeCount
		card.FavoriteCount = totals.FavoriteCount
	}
}

func (a *MyArticleService) attachArticleReactionCounts(ctx context.Context, article *port.Article) {
	if a.reactions == nil || article == nil || article.Id <= 0 {
		return
	}
	counts, err := a.reactions.Counts(ctx, []int{article.Id})
	if err != nil {
		slog.WarnContext(ctx, "load article reaction counts failed", "error", err)
		return
	}
	totals := counts[article.Id]
	article.LikeCount = totals.LikeCount
	article.FavoriteCount = totals.FavoriteCount
}

func (a *MyArticleService) attachAdminReactionCounts(ctx context.Context, rows []*port.ArticleAdmin) {
	if a.reactions == nil || len(rows) == 0 {
		return
	}
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.Id > 0 {
			ids = append(ids, row.Id)
		}
	}
	counts, err := a.reactions.Counts(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "load article reaction counts failed", "error", err)
		return
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		totals := counts[row.Id]
		row.LikeCount = totals.LikeCount
		row.FavoriteCount = totals.FavoriteCount
	}
}
