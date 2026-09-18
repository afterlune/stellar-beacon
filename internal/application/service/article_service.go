package service

import (
	"bytes"
	"container/list"
	"context"
	"errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"io"
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
	repo       port.ArticleRepository
	reactions  port.ArticleReactionRepository
	cache      port.Cache
	storage    port.ObjectStorage
	search     port.ArticleSearcher
	newsletter port.NewsletterEnqueuer
}

func NewArticleService(deps ArticleServiceDeps) (*MyArticleService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyArticleService{
		repo:       deps.Repo,
		reactions:  deps.Reactions,
		cache:      deps.Cache,
		storage:    deps.Storage,
		search:     deps.Search,
		newsletter: deps.Newsletter,
	}, nil
}

// isPubliclyCacheable reports whether an article body may live in the shared
// public cache. Only published, non-recycled articles qualify: drafts,
// scheduled releases and password-protected posts must always be resolved
// through the authenticated path.
func isPubliclyCacheable(article port.Article) bool {
	return article.IsDelete == 0 && article.Status == 1
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
	if article.Status == 2 {
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
	if related, _, relatedErr := a.articleRepository().ListArticles(c.Request.Context(), 1, 6); relatedErr == nil {
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
	if a.cache == nil {
		slog.WarnContext(ctx, "article view cache is not configured")
		return
	}
	if _, err := a.cache.ZIncrBy(ctx, ArticleViewsCount, 1, articleId); err != nil {
		slog.ErrorContext(ctx, "increment article view count failed", "error", err)
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

func (a *MyArticleService) ListArticlesAdmin(c *gin.Context) model.ResultVO {
	var conditionVO model.ConditionVO
	err := c.ShouldBindQuery(&conditionVO)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.ArticleFilter{
		Current:  conditionVO.Current,
		Size:     conditionVO.Size,
		Keywords: conditionVO.Keywords,
		IsDelete: conditionVO.IsDelete,
		Status:   conditionVO.Status,
		Category: conditionVO.CategoryId,
		Type:     conditionVO.Type,
		Tag:      conditionVO.TagId,
	}
	count, err := a.articleRepository().CountArticleAdmins(c.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	articleAdminDTOs, err := a.articleRepository().ListArticlesAdmin(c.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	viewsCountMap := map[string]float64{}
	if a.cache != nil {
		viewsCountMap, err = a.cache.ZRangeWithScores(c.Request.Context(), ArticleViewsCount)
		if err != nil {
			slog.WarnContext(c.Request.Context(), "load article view counts failed", "error", err)
			viewsCountMap = map[string]float64{}
		}
	}
	for _, v := range articleAdminDTOs {
		index := strconv.Itoa(v.Id)
		viewsCount := viewsCountMap[index]
		if viewsCount != 0 {
			v.ViewsCount = int(viewsCount)
		}
	}
	a.attachAdminReactionCounts(c.Request.Context(), articleAdminDTOs)
	if len(articleAdminDTOs) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: articleAdminDTOs, Count: count})
}

func (a *MyArticleService) SaveOrUpdateArticle(c *gin.Context) model.ResultVO {
	var articleVO model.ArticleVO
	if err := c.ShouldBind(&articleVO); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	vo, ok := c.Get("articleVO")
	if ok {
		articleVO = vo.(model.ArticleVO)
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return model.ResultFailWithMessage("用户未登录")
	}
	dto, ok := value.(model.UserDetailsDTO)
	if !ok {
		return model.ResultFailWithMessage("用户信息无效")
	}
	if strings.TrimSpace(articleVO.ArticleContentHTML) != "" {
		articleVO.ArticleContentHTML = sanitizeArticleHTML(articleVO.ArticleContentHTML)
		if !articleHTMLHasContent(articleVO.ArticleContentHTML) {
			return model.ResultFailWithMessage("文章内容不能为空")
		}
		// Keep the legacy field populated so older readers and Markdown exports
		// continue to receive a renderable article body.
		articleVO.ArticleContent = articleVO.ArticleContentHTML
	} else if strings.TrimSpace(articleVO.ArticleContent) == "" {
		return model.ResultFailWithMessage("文章内容不能为空")
	}
	var article entity.TArticle
	if err := normalizeScheduledAt(&articleVO); err != nil {
		return model.ResultFromError(err)
	}
	marshal, err := json.Marshal(articleVO)
	if err != nil {
		return model.ResultFromError(err)
	}
	if err := json.Unmarshal(marshal, &article); err != nil {
		return model.ResultFromError(err)
	}
	article.UserId = dto.UserInfoId
	previousStatus := 0
	if article.Id != 0 {
		if previous, previousErr := a.articleRepository().GetArticleRecord(c.Request.Context(), article.Id); previousErr == nil {
			previousStatus = previous.Status
		} else if !apperrors.IsKind(previousErr, apperrors.KindNotFound) {
			return model.ResultFromError(previousErr)
		}
	}
	articlebase, err := a.articleRepository().SaveOrUpdate(c.Request.Context(), article, articleVO.CategoryName, articleVO.TagNames)
	if err != nil {
		return model.ResultFromError(err)
	}
	if articlebase.Id != 0 {
		if a.newsletter != nil && previousStatus != 1 && articlebase.Status == 1 && articlebase.IsDelete == 0 {
			if err := a.newsletter.EnqueueArticle(c.Request.Context(), articlebase.Id); err != nil {
				// Publishing the article must not fail because the notification
				// outbox is temporarily unavailable; the admin can re-enqueue later.
				slog.ErrorContext(c.Request.Context(), "enqueue newsletter article failed", "articleId", articlebase.Id, "error", err)
			}
		}
		a.cacheArticle(c.Request.Context(), articlebase.Id, articlebase.Status, articlebase.IsDelete, articlebase)
	}
	return model.ResultOk()
}

func (a *MyArticleService) UpdateArticleTopAndFeatured(c *gin.Context) model.ResultVO {
	var articleTopFeaturedVO model.ArticleTopFeaturedVO
	if err := c.ShouldBind(&articleTopFeaturedVO); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	articlebase, err := a.articleRepository().UpdateTopAndFeatured(c.Request.Context(), articleTopFeaturedVO.Id, articleTopFeaturedVO.IsTop, articleTopFeaturedVO.IsFeatured)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultOk()
		}
		return model.ResultFromError(err)
	}
	if articlebase.Id != 0 {
		a.cacheArticle(c.Request.Context(), articlebase.Id, articlebase.Status, articlebase.IsDelete, articlebase)
	}
	return model.ResultOk()
}

func (a *MyArticleService) UpdateArticleDelete(c *gin.Context) model.ResultVO {
	var deleteVO model.DeleteVO
	if err := c.ShouldBind(&deleteVO); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := a.articleRepository().UpdateDelete(c.Request.Context(), deleteVO.Ids, deleteVO.IsDelete); err != nil {
		return model.ResultFromError(err)
	}
	// Recycled or restored articles must not keep a stale public body.
	for _, id := range deleteVO.Ids {
		a.evictArticleCache(c.Request.Context(), strconv.Itoa(id))
	}
	return model.ResultOk()
}

func (a *MyArticleService) DeleteArticles(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := a.articleRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (a *MyArticleService) SaveArticleImages(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "read article image failed", "error", err)
		return model.ResultFail()
	}
	ref, err := uploadMultipart(c.Request.Context(), a.storage, file, "articles/")
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(ref.URL)
}

// normalizeScheduledAt keeps the scheduled release consistent with status 4:
// a scheduled article needs a future timestamp, and every other status must not
// carry a stale one. The value is rewritten as RFC3339 because the VO reaches
// the entity through a JSON round-trip.
func normalizeScheduledAt(vo *model.ArticleVO) error {
	raw := strings.TrimSpace(vo.ScheduledAt)
	if vo.Status != ArticleStatusScheduled {
		vo.ScheduledAt = ""
		return nil
	}
	if raw == "" {
		return apperrors.Invalid("article.schedule", "scheduled articles need a release time")
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		parsed, err = time.Parse("2006-01-02 15:04:05", raw)
	}
	if err != nil {
		return apperrors.Invalid("article.schedule", "release time must be an RFC3339 timestamp")
	}
	if !parsed.After(time.Now()) {
		return apperrors.Invalid("article.schedule", "release time must be in the future")
	}
	vo.ScheduledAt = parsed.Format(time.RFC3339)
	return nil
}

func (a *MyArticleService) GetArticleBackById(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("articleId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	article, categoryName, tagNames, err := a.articleRepository().GetAdminArticle(c.Request.Context(), id)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultOkWithData(model.ArticleAdminViewDTO{})
		}
		return model.ResultFromError(err)
	}
	var articleAdminViewDTO model.ArticleAdminViewDTO
	marshal, err := json.Marshal(article)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "marshal admin article failed", "error", err)
		return model.ResultFail()
	}
	err = json.Unmarshal(marshal, &articleAdminViewDTO)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "decode admin article failed", "error", err)
		return model.ResultFail()
	}
	articleAdminViewDTO.CategoryName = categoryName
	if len(tagNames) == 0 {
		articleAdminViewDTO.TagNames = list.New()
	} else {
		articleAdminViewDTO.TagNames = tagNames
	}
	return model.ResultOkWithData(articleAdminViewDTO)
}

func (a *MyArticleService) ImportArticles(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	filename := file.Filename
	index := strings.LastIndex(filename, ".")
	if index <= 0 || index == len(filename)-1 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	articleTitle := filename[:index]
	content, err := file.Open()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "open imported article failed", "error", err)
		return model.ResultFail()
	}
	defer content.Close()
	all, err := io.ReadAll(content)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "read imported article failed", "error", err)
		return model.ResultFail()
	}
	articleVO := model.ArticleVO{
		ArticleTitle:   articleTitle,
		ArticleContent: string(all),
		Status:         3,
	}
	c.Set("articleVO", articleVO)
	return a.SaveOrUpdateArticle(c)
}

func (a *MyArticleService) ExportArticles(c *gin.Context) model.ResultVO {
	var iDs []int
	err := c.ShouldBind(&iDs)
	if err != nil {
		return model.ResultFailWithMessage("导出文章失败")
	}
	articles, err := a.articleRepository().Export(c.Request.Context(), iDs)
	if err != nil {
		return model.ResultFromError(err)
	}
	var urls []string
	for _, v := range articles {
		ref, err := uploadNamed(c.Request.Context(), a.storage, bytes.NewReader([]byte(v.ArticleContent)), v.ArticleTitle+".md", "markdown/")
		if err != nil {
			return model.ResultFromError(err)
		}
		urls = append(urls, ref.URL)
	}
	return model.ResultOkWithData(urls)
}

func (a *MyArticleService) ListArticlesBySearch(c *gin.Context) model.ResultVO {
	keywords := c.Query("keywords")
	if keywords == "" {
		return model.ResultOk()
	}
	if a.search == nil {
		return model.ResultFromError(apperrors.Unavailable("article.search", nil))
	}
	hits, err := a.search.Search(c.Request.Context(), keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	articleSearchDTOs := make([]model.ArticleSearchDTO, 0, len(hits))
	for _, hit := range hits {
		dto := model.ArticleSearchDTO(hit.ArticleSearch)
		if hit.HighlightedTitle != "" {
			dto.ArticleTitle = hit.HighlightedTitle
		}
		if hit.HighlightedContent != "" {
			dto.ArticleContent = hit.HighlightedContent
		}
		articleSearchDTOs = append(articleSearchDTOs, dto)
	}

	return model.ResultOkWithData(articleSearchDTOs)
}
