package service

import (
	"container/list"
	"context"
	"errors"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

// ArticleStatusScheduled marks an article that the publisher task releases once
// its scheduled_at passes.
const ArticleStatusScheduled = 4

type ArticleService interface {
	ListTopAndFeaturedArticles(c *gin.Context) model.ResultVO
	ListArticles(c *gin.Context) model.ResultVO
	ListArticlesByCategoryId(c *gin.Context) model.ResultVO
	GetArticleById(c *gin.Context) model.ResultVO
	updateArticleViewsCount(ctx context.Context, articleId string)
	ListArticlesByTagId(c *gin.Context) model.ResultVO
	AccessArticle(c *gin.Context) model.ResultVO
	ListArchives(c *gin.Context) model.ResultVO
	ListArticlesAdmin(c *gin.Context) model.ResultVO
	SaveOrUpdateArticle(c *gin.Context) model.ResultVO
	UpdateArticleTopAndFeatured(c *gin.Context) model.ResultVO
	UpdateArticleDelete(c *gin.Context) model.ResultVO
	DeleteArticles(c *gin.Context) model.ResultVO
	SaveArticleImages(c *gin.Context) model.ResultVO
	GetArticleBackById(c *gin.Context) model.ResultVO
	ImportArticles(c *gin.Context) model.ResultVO
	ExportArticles(c *gin.Context) model.ResultVO
	ListArticlesBySearch(c *gin.Context) model.ResultVO
}

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

func (a *MyArticleService) ListTopAndFeaturedArticles(c *gin.Context) model.ResultVO {
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	data, err := a.articleRepository().ListTopAndFeaturedArticles(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	a.attachCardReactionCounts(ctx, data)
	if len(data) == 0 {
		return model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{})
	} else if len(data) > 3 {
		data = data[:3]
		return model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{TopArticle: data[0], FeaturedArticles: data[1:3]})
	} else {
		return model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{TopArticle: data[0], FeaturedArticles: data[1:]})
	}
}

func (a *MyArticleService) ListArticles(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	data, count, err := a.articleRepository().ListArticles(c.Request.Context(), current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	a.attachCardReactionCounts(c.Request.Context(), data)
	if len(data) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: count})
}

func (a *MyArticleService) ListArticlesByCategoryId(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}

	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}

	// Category names are also matched across authors when the caller supplies
	// one; the id path is kept for existing links.
	categoryName := strings.TrimSpace(c.Query("categoryName"))
	if categoryName == "" {
		categoryName = strings.TrimSpace(c.Query("name"))
	}
	if categoryName != "" {
		data, count, err := a.articleRepository().GetArticlesByCategoryName(c.Request.Context(), current, size, categoryName)
		if err != nil {
			return model.ResultFromError(err)
		}
		a.attachCardReactionCounts(c.Request.Context(), data)
		if len(data) == 0 {
			return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
		}
		return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: count})
	}
	categoryID, err := strconv.Atoi(c.Query("categoryId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	data, count, err := a.articleRepository().GetArticlesByCategoryID(c.Request.Context(), current, size, categoryID)
	if err != nil {
		return model.ResultFromError(err)
	}
	a.attachCardReactionCounts(c.Request.Context(), data)
	if len(data) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: count})
}

func (a *MyArticleService) GetArticleById(c *gin.Context) model.ResultVO {
	articleId := c.Param("articleId")
	var err error
	get := ""
	if a.cache != nil {
		get, err = a.cache.Get(c.Request.Context(), articleId)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(c.Request.Context(), "read article cache failed", "error", err)
		}
	}
	if get != "" {
		var dto model.ArticleDTO
		if err := Unmarsh(get, &dto); err == nil {
			// A cached body must never outlive the article's visibility: drafts,
			// scheduled and recycled articles are only readable through the
			// authenticated admin endpoints.
			if !isPubliclyCacheable(dto) {
				a.evictArticleCache(c.Request.Context(), articleId)
			} else {
				sanitizePublicArticle(&dto)
				a.attachArticleReactionCounts(c.Request.Context(), &dto)
				a.updateArticleViewsCount(c.Request.Context(), articleId)
				if a.cache != nil {
					if _, err := a.cache.Expire(c.Request.Context(), articleId, time.Hour*1); err != nil {
						slog.WarnContext(c.Request.Context(), "refresh article cache TTL failed", "error", err)
					}
				}
				return model.ResultOkWithData(dto)
			}
		} else {
			slog.WarnContext(c.Request.Context(), "decode article cache failed", "error", err)
		}
	}
	articleID, err := strconv.Atoi(articleId)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	article, err := a.articleRepository().GetArticleRecord(c.Request.Context(), articleID)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultOk()
		}
		return model.ResultFromError(err)
	}
	if article.Id == 0 {
		return model.ResultOk()
	}
	if article.Status != 1 || article.ModerationStatus == "hidden" {
		return model.ResultOk()
	}
	if article.Password != "" {
		value, ok := c.Get("userInfo")
		if !ok {
			return model.ResultFailWithMessage("无权访问")
		}
		dto, ok := value.(model.UserDetailsDTO)
		if !ok {
			return model.ResultFailWithMessage("无权访问")
		}
		if a.cache == nil {
			return model.ResultFailWithMessage("系统繁忙，请稍后再试")
		}
		isAccess, err := a.cache.SIsMember(c.Request.Context(), ArticleAccess+strconv.Itoa(dto.Id), articleId)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "check article access failed", "error", err)
			return model.ResultFail()
		}
		if !isAccess {
			status := model.ResultInfo(model.ARTICLE_ACCESS_FAIL)
			return model.ResultFailWithCodeAndMessage(52003, status["message"])
		}
	}
	a.updateArticleViewsCount(c.Request.Context(), articleId)
	id := articleID
	data, err := a.articleRepository().GetArticleByID(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	preData, err := a.articleRepository().GetPreArticleByID(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	if preData.Id == 0 {
		preData, err = a.articleRepository().GetLastArticle(c.Request.Context())
		if err != nil {
			return model.ResultFromError(err)
		}
	}
	nextData, err := a.articleRepository().GetNextArticleByID(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	if nextData.Id == 0 {
		nextData, err = a.articleRepository().GetFirstArticle(c.Request.Context())
		if err != nil {
			return model.ResultFromError(err)
		}
	}
	if related, relatedErr := a.articleRepository().ListRelatedArticles(c.Request.Context(), data.Id, article.CategoryId, article.SeriesId, 3); relatedErr == nil {
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
		slog.WarnContext(c.Request.Context(), "load related articles failed", "error", relatedErr)
	}
	if data.Id == 0 {
		return model.ResultOk()
	}
	score := float64(0)
	if a.cache != nil {
		score, err = a.cache.ZScore(c.Request.Context(), ArticleViewsCount, articleId)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(c.Request.Context(), "read article view count failed", "error", err)
		}
	}
	if score != 0 {
		data.ViewCount = int(score)
	}
	data.PreArticleCard = preData
	data.NextArticleCard = nextData
	relatedPointers := make([]*port.ArticleCard, 0, len(data.RelatedArticles))
	for index := range data.RelatedArticles {
		relatedPointers = append(relatedPointers, &data.RelatedArticles[index])
	}
	a.attachCardReactionCounts(c.Request.Context(), relatedPointers)
	sanitizePublicArticle(&data)
	marshal, err := json.Marshal(data)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "marshal article cache failed", "error", err)
		return model.ResultFromError(err)
	}
	if a.cache != nil {
		if err := a.cache.Set(c.Request.Context(), strconv.Itoa(data.Id), marshal, time.Hour*1); err != nil {
			slog.WarnContext(c.Request.Context(), "write article cache failed", "error", err)
		}
	}
	a.attachArticleReactionCounts(c.Request.Context(), &data)
	return model.ResultOkWithData(data)
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

func (a *MyArticleService) ListArticlesByTagId(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	// The discovery surfaces group per-author tags by their trimmed lower-cased
	// name, so a named lookup aggregates across authors. The id lookup stays as
	// the compatibility path for older links.
	tagName := strings.TrimSpace(c.Query("tagName"))
	if tagName != "" {
		data, count, err := a.articleRepository().ListArticlesByTagName(c.Request.Context(), current, size, tagName)
		if err != nil {
			return model.ResultFromError(err)
		}
		a.attachCardReactionCounts(c.Request.Context(), data)
		if len(data) == 0 {
			return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
		}
		return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: count})
	}
	tagId := c.Query("tagId")
	id, err := strconv.Atoi(tagId)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	data, count, err := a.articleRepository().ListArticlesByTagID(c.Request.Context(), current, size, id)
	if err != nil {
		return model.ResultFromError(err)
	}
	a.attachCardReactionCounts(c.Request.Context(), data)
	if len(data) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (a *MyArticleService) AccessArticle(c *gin.Context) model.ResultVO {
	var vo model.ArticlePasswordVO
	err := c.ShouldBind(&vo)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "bind article password failed", "error", err)
		return model.ResultFail()
	}
	article, err := a.articleRepository().GetArticleRecord(c.Request.Context(), vo.ArticleId)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultFailWithMessage("文章不存在")
		}
		return model.ResultFromError(err)
	}
	if article.Id == 0 {
		return model.ResultFailWithMessage("文章不存在")
	}
	if article.Password == vo.ArticlePassword {
		value, ok := c.Get("userInfo")
		if !ok {
			return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "article.access.user", nil))
		}
		dto, ok := value.(model.UserDetailsDTO)
		if !ok {
			return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "article.access.user", nil))
		}
		if a.cache == nil {
			return model.ResultFailWithMessage("系统繁忙，请稍后再试")
		}
		if _, err := a.cache.SAdd(c.Request.Context(), ArticleAccess+strconv.Itoa(dto.Id), vo.ArticleId); err != nil {
			slog.ErrorContext(c.Request.Context(), "record article access failed", "error", err)
			return model.ResultFail()
		}
	} else {
		return model.ResultFailWithMessage("密码错误")
	}
	return model.ResultOk()
}

func (a *MyArticleService) ListArchives(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}

	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}

	articles, count, err := a.articleRepository().ListArchives(c.Request.Context(), current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	archivePointers := make([]*port.ArticleCard, 0, len(articles))
	for index := range articles {
		archivePointers = append(archivePointers, &articles[index])
	}
	a.attachCardReactionCounts(c.Request.Context(), archivePointers)
	type archiveGroup struct {
		date time.Time
		dto  model.ArchiveDTO
	}
	hm := make(map[string]*archiveGroup)
	for _, v := range articles {
		key := v.CreateTime.Format("2006-1-2")
		group, ok := hm[key]
		if !ok {
			group = &archiveGroup{
				date: v.CreateTime,
				dto:  model.ArchiveDTO{Time: key},
			}
			hm[key] = group
		}
		group.dto.Articles = append(group.dto.Articles, v)
	}
	groups := make([]archiveGroup, 0, len(hm))
	for _, group := range hm {
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].date.After(groups[j].date)
	})
	archiveDTOs := make([]model.ArchiveDTO, 0, len(groups))
	for _, group := range groups {
		archiveDTOs = append(archiveDTOs, group.dto)
	}
	if len(archiveDTOs) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: archiveDTOs, Count: int(count)})
}
