package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/goccy/go-json"
)

// PublicArticleReader contains the application use cases used by anonymous
// article discovery and reading endpoints. HTTP request parsing and response
// envelopes stay in the HTTP adapter.
type PublicArticleReader interface {
	ListFeatured(context.Context) (FeaturedArticles, error)
	List(context.Context, PageQuery) (ArticlePage[*port.ArticleCard], error)
	ListByCategory(context.Context, CategoryArticleQuery) (ArticlePage[*port.ArticleCard], error)
	Get(context.Context, int, int) (*port.Article, error)
	GrantPasswordAccess(context.Context, ArticlePasswordAccess) error
	ListByTag(context.Context, TagArticleQuery) (ArticlePage[*port.ArticleCard], error)
	ListArchives(context.Context, PageQuery) (ArticlePage[ArticleArchiveGroup], error)
	Search(context.Context, ArticleSearchQuery) (ArticlePage[port.ArticleSearchHit], error)
}

type PageQuery struct {
	Current int
	Size    int
}

type ArticlePage[T any] struct {
	Items    []T
	Total    int
	Page     int
	PageSize int
}

type FeaturedArticles struct {
	TopArticle       *port.ArticleCard
	FeaturedArticles []*port.ArticleCard
}

type CategoryArticleQuery struct {
	Page PageQuery
	ID   int
	Name string
}

type TagArticleQuery struct {
	Page PageQuery
	ID   int
	Name string
}

type ArticleSearchQuery struct {
	Page     PageQuery
	Keywords string
}

type ArticlePasswordAccess struct {
	ArticleID int
	Password  string
	UserID    int
}

type ArticleArchiveGroup struct {
	Time     string
	Articles []port.ArticleCard
}

type PublicArticleFailure string

const (
	PublicArticleAccessDenied      PublicArticleFailure = "access_denied"
	PublicArticlePasswordRequired  PublicArticleFailure = "password_required"
	PublicArticleAccessCheckFailed PublicArticleFailure = "access_check_failed"
	PublicArticleNotFoundForAccess PublicArticleFailure = "not_found_for_access"
	PublicArticlePasswordInvalid   PublicArticleFailure = "password_invalid"
	PublicArticleGrantFailed       PublicArticleFailure = "grant_failed"
)

// PublicArticleError identifies legacy public article failures without
// coupling the application service to HTTP result types or localized text.
type PublicArticleError struct {
	Failure PublicArticleFailure
	Cause   error
}

func (e *PublicArticleError) Error() string {
	if e == nil {
		return "public article operation failed"
	}
	if e.Cause != nil {
		return fmt.Sprintf("public article %s: %v", e.Failure, e.Cause)
	}
	return fmt.Sprintf("public article %s", e.Failure)
}

func (e *PublicArticleError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func publicArticleFailure(failure PublicArticleFailure, cause error) error {
	return &PublicArticleError{Failure: failure, Cause: cause}
}

func (a *MyArticleService) ListFeatured(ctx context.Context) (FeaturedArticles, error) {
	data, err := a.articleRepository().ListTopAndFeaturedArticles(ctx)
	if err != nil {
		return FeaturedArticles{}, err
	}
	a.attachCardReactionCounts(ctx, data)
	if len(data) == 0 {
		return FeaturedArticles{}, nil
	}
	if len(data) > 3 {
		data = data[:3]
	}
	return FeaturedArticles{TopArticle: data[0], FeaturedArticles: data[1:]}, nil
}

func (a *MyArticleService) List(ctx context.Context, query PageQuery) (ArticlePage[*port.ArticleCard], error) {
	data, count, err := a.articleRepository().ListArticles(ctx, query.Current, query.Size)
	if err != nil {
		return ArticlePage[*port.ArticleCard]{}, err
	}
	a.attachCardReactionCounts(ctx, data)
	return articlePage(data, count, query), nil
}

func (a *MyArticleService) ListByCategory(ctx context.Context, query CategoryArticleQuery) (ArticlePage[*port.ArticleCard], error) {
	name := strings.TrimSpace(query.Name)
	var data []*port.ArticleCard
	var count int
	var err error
	if name != "" {
		data, count, err = a.articleRepository().GetArticlesByCategoryName(ctx, query.Page.Current, query.Page.Size, name)
	} else {
		data, count, err = a.articleRepository().GetArticlesByCategoryID(ctx, query.Page.Current, query.Page.Size, query.ID)
	}
	if err != nil {
		return ArticlePage[*port.ArticleCard]{}, err
	}
	a.attachCardReactionCounts(ctx, data)
	return articlePage(data, count, query.Page), nil
}

func (a *MyArticleService) Get(ctx context.Context, articleID, userID int) (*port.Article, error) {
	articleKey := strconv.Itoa(articleID)
	if a.cache != nil {
		cached, err := a.cache.Get(ctx, articleKey)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(ctx, "read article cache failed", "error", err)
		}
		if cached != "" {
			var dto port.Article
			if err := json.Unmarshal([]byte(cached), &dto); err == nil {
				if !isPubliclyCacheable(dto) {
					a.evictArticleCache(ctx, articleKey)
				} else {
					sanitizePublicArticle(&dto)
					a.attachArticleReactionCounts(ctx, &dto)
					a.updateArticleViewsCount(ctx, articleKey)
					if _, err := a.cache.Expire(ctx, articleKey, time.Hour); err != nil {
						slog.WarnContext(ctx, "refresh article cache TTL failed", "error", err)
					}
					return &dto, nil
				}
			} else {
				slog.WarnContext(ctx, "decode article cache failed", "error", err)
			}
		}
	}

	article, err := a.articleRepository().GetArticleRecord(ctx, articleID)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if article.Id == 0 || article.Status != 1 || article.ModerationStatus == "hidden" {
		return nil, nil
	}
	if article.Password != "" {
		if userID <= 0 {
			return nil, publicArticleFailure(PublicArticleAccessDenied, nil)
		}
		if a.cache == nil {
			return nil, apperrors.Unavailable("article.read.access", nil)
		}
		allowed, err := a.cache.SIsMember(ctx, ArticleAccess+strconv.Itoa(userID), articleKey)
		if err != nil {
			slog.ErrorContext(ctx, "check article access failed", "error", err)
			return nil, publicArticleFailure(PublicArticleAccessCheckFailed, err)
		}
		if !allowed {
			return nil, publicArticleFailure(PublicArticlePasswordRequired, nil)
		}
	}

	a.updateArticleViewsCount(ctx, articleKey)
	data, err := a.articleRepository().GetArticleByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	preData, err := a.articleRepository().GetPreArticleByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if preData.Id == 0 {
		preData, err = a.articleRepository().GetLastArticle(ctx)
		if err != nil {
			return nil, err
		}
	}
	nextData, err := a.articleRepository().GetNextArticleByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if nextData.Id == 0 {
		nextData, err = a.articleRepository().GetFirstArticle(ctx)
		if err != nil {
			return nil, err
		}
	}
	if related, relatedErr := a.articleRepository().ListRelatedArticles(ctx, data.Id, article.CategoryId, article.SeriesId, 3); relatedErr == nil {
		data.RelatedArticles = make([]port.ArticleCard, 0, 3)
		for _, candidate := range related {
			if candidate == nil || candidate.Id == data.Id || candidate.Status != 1 {
				continue
			}
			data.RelatedArticles = append(data.RelatedArticles, *candidate)
			if len(data.RelatedArticles) == 3 {
				break
			}
		}
	} else {
		slog.WarnContext(ctx, "load related articles failed", "error", relatedErr)
	}
	if data.Id == 0 {
		return nil, nil
	}
	if a.cache != nil {
		score, err := a.cache.ZScore(ctx, ArticleViewsCount, articleKey)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(ctx, "read article view count failed", "error", err)
		}
		if score != 0 {
			data.ViewCount = int(score)
		}
	}
	data.PreArticleCard = preData
	data.NextArticleCard = nextData
	relatedPointers := make([]*port.ArticleCard, 0, len(data.RelatedArticles))
	for index := range data.RelatedArticles {
		relatedPointers = append(relatedPointers, &data.RelatedArticles[index])
	}
	a.attachCardReactionCounts(ctx, relatedPointers)
	sanitizePublicArticle(&data)
	encoded, err := json.Marshal(data)
	if err != nil {
		slog.ErrorContext(ctx, "marshal article cache failed", "error", err)
		return nil, err
	}
	if a.cache != nil {
		if err := a.cache.Set(ctx, articleKey, encoded, time.Hour); err != nil {
			slog.WarnContext(ctx, "write article cache failed", "error", err)
		}
	}
	a.attachArticleReactionCounts(ctx, &data)
	return &data, nil
}

func (a *MyArticleService) GrantPasswordAccess(ctx context.Context, input ArticlePasswordAccess) error {
	article, err := a.articleRepository().GetArticleRecord(ctx, input.ArticleID)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return publicArticleFailure(PublicArticleNotFoundForAccess, err)
		}
		return err
	}
	if article.Id == 0 {
		return publicArticleFailure(PublicArticleNotFoundForAccess, nil)
	}
	if article.Password != input.Password {
		return publicArticleFailure(PublicArticlePasswordInvalid, nil)
	}
	if input.UserID <= 0 {
		return apperrors.New(apperrors.KindUnauthorized, "article.access.user", nil)
	}
	if a.cache == nil {
		return apperrors.Unavailable("article.access.cache", nil)
	}
	if _, err := a.cache.SAdd(ctx, ArticleAccess+strconv.Itoa(input.UserID), input.ArticleID); err != nil {
		slog.ErrorContext(ctx, "record article access failed", "error", err)
		return publicArticleFailure(PublicArticleGrantFailed, err)
	}
	return nil
}

func (a *MyArticleService) ListByTag(ctx context.Context, query TagArticleQuery) (ArticlePage[*port.ArticleCard], error) {
	name := strings.TrimSpace(query.Name)
	var data []*port.ArticleCard
	var count int
	var err error
	if name != "" {
		data, count, err = a.articleRepository().ListArticlesByTagName(ctx, query.Page.Current, query.Page.Size, name)
	} else {
		data, count, err = a.articleRepository().ListArticlesByTagID(ctx, query.Page.Current, query.Page.Size, query.ID)
	}
	if err != nil {
		return ArticlePage[*port.ArticleCard]{}, err
	}
	a.attachCardReactionCounts(ctx, data)
	return articlePage(data, count, query.Page), nil
}

func (a *MyArticleService) ListArchives(ctx context.Context, query PageQuery) (ArticlePage[ArticleArchiveGroup], error) {
	articles, count, err := a.articleRepository().ListArchives(ctx, query.Current, query.Size)
	if err != nil {
		return ArticlePage[ArticleArchiveGroup]{}, err
	}
	articlePointers := make([]*port.ArticleCard, 0, len(articles))
	for index := range articles {
		articlePointers = append(articlePointers, &articles[index])
	}
	a.attachCardReactionCounts(ctx, articlePointers)
	type archiveGroup struct {
		date  time.Time
		value ArticleArchiveGroup
	}
	byMonth := make(map[string]*archiveGroup)
	for _, article := range articles {
		key := article.CreateTime.Format("2006-1")
		group, ok := byMonth[key]
		if !ok {
			month := time.Date(article.CreateTime.Year(), article.CreateTime.Month(), 1, 0, 0, 0, 0, article.CreateTime.Location())
			group = &archiveGroup{date: month, value: ArticleArchiveGroup{Time: key, Articles: []port.ArticleCard{}}}
			byMonth[key] = group
		}
		group.value.Articles = append(group.value.Articles, article)
	}
	groups := make([]archiveGroup, 0, len(byMonth))
	for _, group := range byMonth {
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].date.After(groups[j].date) })
	items := make([]ArticleArchiveGroup, 0, len(groups))
	for _, group := range groups {
		items = append(items, group.value)
	}
	return ArticlePage[ArticleArchiveGroup]{Items: items, Total: count, Page: query.Current, PageSize: query.Size}, nil
}

func (a *MyArticleService) Search(ctx context.Context, query ArticleSearchQuery) (ArticlePage[port.ArticleSearchHit], error) {
	keywords := strings.TrimSpace(query.Keywords)
	if keywords == "" {
		return ArticlePage[port.ArticleSearchHit]{Items: []port.ArticleSearchHit{}, Page: query.Page.Current, PageSize: query.Page.Size}, nil
	}
	if a.search == nil {
		return ArticlePage[port.ArticleSearchHit]{}, apperrors.Unavailable("article.search", nil)
	}
	page, err := a.search.Search(ctx, keywords, (query.Page.Current-1)*query.Page.Size, query.Page.Size)
	if err != nil {
		return ArticlePage[port.ArticleSearchHit]{}, err
	}
	return ArticlePage[port.ArticleSearchHit]{
		Items: page.Hits, Total: int(page.Total), Page: query.Page.Current, PageSize: query.Page.Size,
	}, nil
}

func articlePage[T any](items []T, total int, query PageQuery) ArticlePage[T] {
	if items == nil {
		items = []T{}
	}
	return ArticlePage[T]{Items: items, Total: total, Page: query.Current, PageSize: query.Size}
}

func (a *MyArticleService) updateArticleViewsCount(ctx context.Context, articleId string) {
	if a.cache != nil {
		if _, err := a.cache.ZIncrBy(ctx, ArticleViewsCount, 1, articleId); err != nil {
			slog.ErrorContext(ctx, "increment article view count failed", "error", err)
		}
	}
	if a.contentAnalytics == nil {
		return
	}
	id, err := strconv.Atoi(articleId)
	if err != nil || id <= 0 {
		return
	}
	if err := a.contentAnalytics.RecordView(ctx, id, timeNow()); err != nil {
		slog.WarnContext(ctx, "record article daily view failed", "articleId", id, "error", err)
	}
}

var _ PublicArticleReader = (*MyArticleService)(nil)
