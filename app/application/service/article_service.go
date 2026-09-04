package service

import (
	"benetnasch/app/application/support"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"bytes"
	"container/list"
	"context"
	"errors"
	"io"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

type ArticleService interface {
	ListTopAndFeaturedArticles(c port.Request) port.ResultVO
	ListArticles(c port.Request) port.ResultVO
	ListArticlesByCategoryId(c port.Request) port.ResultVO
	GetArticleById(c port.Request) port.ResultVO
	updateArticleViewsCount(ctx context.Context, articleId string)
	ListArticlesByTagId(c port.Request) port.ResultVO
	AccessArticle(c port.Request) port.ResultVO
	ListArchives(c port.Request) port.ResultVO
	ListArticlesAdmin(c port.Request) port.ResultVO
	SaveOrUpdateArticle(c port.Request) port.ResultVO
	UpdateArticleTopAndFeatured(c port.Request) port.ResultVO
	UpdateArticleDelete(c port.Request) port.ResultVO
	DeleteArticles(c port.Request) port.ResultVO
	SaveArticleImages(c port.Request) port.ResultVO
	GetArticleBackById(c port.Request) port.ResultVO
	ImportArticles(c port.Request) port.ResultVO
	ExportArticles(c port.Request) port.ResultVO
	ListArticlesBySearch(c port.Request) port.ResultVO
}

type MyArticleService struct {
	repo                     port.ArticleRepository
	cache                    port.Cache
	storage                  port.ObjectStorage
	search                   port.ArticleSearcher
	aiJobs                   port.AIJobRepository
	contentUnderstandingJobs port.AIJobRepository
}

func NewArticleService(deps ArticleServiceDeps) (*MyArticleService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyArticleService{
		repo:                     deps.Repo,
		cache:                    deps.Cache,
		storage:                  deps.Storage,
		search:                   deps.Search,
		aiJobs:                   deps.AIJobs,
		contentUnderstandingJobs: deps.ContentUnderstandingJobs,
	}, nil
}

func (a *MyArticleService) articleRepository() port.ArticleRepository {
	return a.repo
}

func (a *MyArticleService) ListTopAndFeaturedArticles(c port.Request) port.ResultVO {
	ctx := context.Background()
	if c != nil && c.HTTPRequest() != nil {
		ctx = c.Context()
	}
	data, err := a.articleRepository().ListTopAndFeaturedArticles(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	if len(data) == 0 {
		return port.ResultOkWithData(port.TopAndFeaturedArticlesDTO{})
	} else if len(data) > 3 {
		data = data[:3]
		return port.ResultOkWithData(port.TopAndFeaturedArticlesDTO{TopArticle: data[0], FeaturedArticles: data[1:3]})
	} else {
		return port.ResultOkWithData(port.TopAndFeaturedArticlesDTO{TopArticle: data[0], FeaturedArticles: data[1:]})
	}
}

func (a *MyArticleService) ListArticles(c port.Request) port.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	data, count, err := a.articleRepository().ListArticles(c.Context(), current, size)
	if err != nil {
		return port.ResultFromError(err)
	}
	if len(data) == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: data, Count: count})
}

func (a *MyArticleService) ListArticlesByCategoryId(c port.Request) port.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}

	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}

	categoryID, err := strconv.Atoi(c.Query("categoryId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	data, count, err := a.articleRepository().GetArticlesByCategoryID(c.Context(), current, size, categoryID)
	if err != nil {
		return port.ResultFromError(err)
	}
	if len(data) == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: data, Count: count})
}

func (a *MyArticleService) GetArticleById(c port.Request) port.ResultVO {
	articleId := c.Param("articleId")
	articleID, err := strconv.Atoi(articleId)
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	article, err := a.articleRepository().GetArticleRecord(c.Context(), articleID)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return port.ResultOk()
		}
		return port.ResultFromError(err)
	}
	if article.Id == 0 {
		return port.ResultOk()
	}
	// Visibility must be checked against the current record before reading the
	// article cache. A previously public cached response is not proof that the
	// article is still public after a status change.
	if article.IsDelete != 0 || (article.Status != port.PublicArticleStatus && article.Status != 2) {
		return port.ResultOk()
	}
	if article.Status == 2 {
		value, ok := c.Get("userInfo")
		if !ok {
			return port.ResultFailWithMessage("无权访问")
		}
		dto, ok := value.(port.UserDetailsDTO)
		if !ok {
			return port.ResultFailWithMessage("无权访问")
		}
		if a.cache == nil {
			return port.ResultFailWithMessage("系统繁忙，请稍后再试")
		}
		isAccess, err := a.cache.SIsMember(c.Context(), support.ArticleAccess+strconv.Itoa(dto.Id), articleId)
		if err != nil {
			slog.ErrorContext(c.Context(), "check article access failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFail()
		}
		if !isAccess {
			status := port.ResultInfo(port.ARTICLE_ACCESS_FAIL)
			return port.ResultFailWithCodeAndMessage(52003, status["desc"])
		}
	}
	if a.cache != nil {
		get, err := a.cache.Get(c.Context(), articleId)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(c.Context(), "read article cache failed", "error_code", apperrors.SafeCode(err))
		}
		if get != "" {
			var dto port.ArticleDTO
			if err := support.Unmarsh(get, &dto); err == nil {
				if dto.Id == articleID && dto.Status == article.Status && dto.IsDelete == article.IsDelete {
					if _, err := a.cache.Expire(c.Context(), articleId, time.Hour*1); err != nil {
						slog.WarnContext(c.Context(), "refresh article cache TTL failed", "error_code", apperrors.SafeCode(err))
					}
					return port.ResultOkWithData(dto)
				}
				slog.WarnContext(c.Context(), "article cache visibility metadata mismatch", "article_id", articleID)
			} else {
				slog.WarnContext(c.Context(), "decode article cache failed", "error_code", apperrors.SafeCode(err))
			}
		}
	}
	a.updateArticleViewsCount(c.Context(), articleId)
	id := articleID
	data, err := a.articleRepository().GetArticleByID(c.Context(), id)
	if err != nil {
		return port.ResultFromError(err)
	}
	preData, err := a.articleRepository().GetPreArticleByID(c.Context(), id)
	if err != nil {
		return port.ResultFromError(err)
	}
	if preData.Id == 0 {
		preData, err = a.articleRepository().GetLastArticle(c.Context())
		if err != nil {
			return port.ResultFromError(err)
		}
	}
	nextData, err := a.articleRepository().GetNextArticleByID(c.Context(), id)
	if err != nil {
		return port.ResultFromError(err)
	}
	if nextData.Id == 0 {
		nextData, err = a.articleRepository().GetFirstArticle(c.Context())
		if err != nil {
			return port.ResultFromError(err)
		}
	}
	if data.Id == 0 {
		return port.ResultOk()
	}
	score := float64(0)
	if a.cache != nil {
		score, err = a.cache.ZScore(c.Context(), support.ArticleViewsCount, articleId)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(c.Context(), "read article view count failed", "error_code", apperrors.SafeCode(err))
		}
	}
	if score != 0 {
		data.ViewCount = int(score)
	}
	data.PreArticleCard = preData
	data.NextArticleCard = nextData
	marshal, err := json.Marshal(data)
	if err != nil {
		slog.ErrorContext(c.Context(), "marshal article cache failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFromError(err)
	}
	if a.cache != nil {
		if err := a.cache.Set(c.Context(), strconv.Itoa(data.Id), marshal, time.Hour*1); err != nil {
			slog.WarnContext(c.Context(), "write article cache failed", "error_code", apperrors.SafeCode(err))
		}
	}
	return port.ResultOkWithData(data)
}

func (a *MyArticleService) updateArticleViewsCount(ctx context.Context, articleId string) {
	if a.cache == nil {
		slog.WarnContext(ctx, "article view cache is not configured")
		return
	}
	if _, err := a.cache.ZIncrBy(ctx, support.ArticleViewsCount, 1, articleId); err != nil {
		slog.ErrorContext(ctx, "increment article view count failed", "error_code", apperrors.SafeCode(err))
	}
}

func (a *MyArticleService) ListArticlesByTagId(c port.Request) port.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	tagId := c.Query("tagId")
	id, err := strconv.Atoi(tagId)
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	data, count, err := a.articleRepository().ListArticlesByTagID(c.Context(), current, size, id)
	if err != nil {
		return port.ResultFromError(err)
	}
	if len(data) == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: data, Count: int(count)})
}

func (a *MyArticleService) AccessArticle(c port.Request) port.ResultVO {
	var vo port.ArticlePasswordVO
	err := c.Bind(&vo)
	if err != nil {
		slog.ErrorContext(c.Context(), "bind article password failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	article, err := a.articleRepository().GetArticleRecord(c.Context(), vo.ArticleId)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return port.ResultFailWithMessage("文章不存在")
		}
		return port.ResultFromError(err)
	}
	if article.Id == 0 {
		return port.ResultFailWithMessage("文章不存在")
	}
	if article.Password == vo.ArticlePassword {
		value, ok := c.Get("userInfo")
		if !ok {
			return port.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "article.access.user", nil))
		}
		dto, ok := value.(port.UserDetailsDTO)
		if !ok {
			return port.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "article.access.user", nil))
		}
		if a.cache == nil {
			return port.ResultFailWithMessage("系统繁忙，请稍后再试")
		}
		if _, err := a.cache.SAdd(c.Context(), support.ArticleAccess+strconv.Itoa(dto.Id), vo.ArticleId); err != nil {
			slog.ErrorContext(c.Context(), "record article access failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFail()
		}
	} else {
		return port.ResultFailWithMessage("密码错误")
	}
	return port.ResultOk()
}

func (a *MyArticleService) ListArchives(c port.Request) port.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}

	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}

	articles, count, err := a.articleRepository().ListArchives(c.Context(), current, size)
	if err != nil {
		return port.ResultFromError(err)
	}
	type archiveGroup struct {
		date time.Time
		dto  port.ArchiveDTO
	}
	hm := make(map[string]*archiveGroup)
	for _, v := range articles {
		key := v.CreateTime.Format("2006-1-2")
		group, ok := hm[key]
		if !ok {
			group = &archiveGroup{
				date: v.CreateTime,
				dto:  port.ArchiveDTO{Time: key},
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
	archiveDTOs := make([]port.ArchiveDTO, 0, len(groups))
	for _, group := range groups {
		archiveDTOs = append(archiveDTOs, group.dto)
	}
	if len(archiveDTOs) == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: archiveDTOs, Count: int(count)})
}

func (a *MyArticleService) ListArticlesAdmin(c port.Request) port.ResultVO {
	var conditionVO port.ConditionVO
	err := c.BindQuery(&conditionVO)
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
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
	count, err := a.articleRepository().CountArticleAdmins(c.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	articleAdminDTOs, err := a.articleRepository().ListArticlesAdmin(c.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	viewsCountMap := map[string]float64{}
	if a.cache != nil {
		viewsCountMap, err = a.cache.ZRangeWithScores(c.Context(), support.ArticleViewsCount)
		if err != nil {
			slog.WarnContext(c.Context(), "load article view counts failed", "error_code", apperrors.SafeCode(err))
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
	if len(articleAdminDTOs) == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: articleAdminDTOs, Count: count})
}

func (a *MyArticleService) SaveOrUpdateArticle(c port.Request) port.ResultVO {
	var articleVO port.ArticleVO
	if err := c.Bind(&articleVO); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	vo, ok := c.Get("articleVO")
	if ok {
		parsed, ok := vo.(port.ArticleVO)
		if !ok {
			return port.ResultFailWithMessage("参数格式不正确")
		}
		articleVO = parsed
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return port.ResultFailWithMessage("用户未登录")
	}
	dto, ok := value.(port.UserDetailsDTO)
	if !ok {
		return port.ResultFailWithMessage("用户信息无效")
	}
	var article port.TArticle
	marshal, err := json.Marshal(articleVO)
	if err != nil {
		return port.ResultFromError(err)
	}
	if err := json.Unmarshal(marshal, &article); err != nil {
		return port.ResultFromError(err)
	}
	wasExisting := article.Id != 0
	article.UserId = dto.UserInfoId
	articlebase, err := a.articleRepository().SaveOrUpdate(c.Context(), article, articleVO.CategoryName, articleVO.TagNames)
	if err != nil {
		return port.ResultFromError(err)
	}
	action, event := articleIndexMutation(articlebase, !wasExisting)
	if err := a.enqueueArticleIndexJob(c.Context(), articlebase.Id, action, event, articlebase.Status, articlebase.IsDelete, articlebase.UpdateTime); err != nil {
		slog.ErrorContext(c.Context(), "enqueue article index job failed", "article_id", articlebase.Id, "error_code", apperrors.SafeCode(err))
		return port.ResultFromError(err)
	}
	if port.IsPublicArticle(articlebase.Status, articlebase.IsDelete) {
		if err := a.enqueueContentUnderstandingJob(c.Context(), articlebase.Id, articlebase.UpdateTime); err != nil {
			slog.ErrorContext(c.Context(), "enqueue content understanding job failed", "article_id", articlebase.Id, "error_code", apperrors.SafeCode(err))
			return port.ResultFromError(err)
		}
	}
	if articlebase.Id != 0 {
		marsha, err := json.Marshal(articlebase)
		if err != nil {
			slog.ErrorContext(c.Context(), "marshal article cache failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFail()
		}
		if a.cache != nil {
			if err := a.cache.Set(c.Context(), strconv.Itoa(articlebase.Id), marsha, 0); err != nil {
				slog.WarnContext(c.Context(), "cache article failed", "error_code", apperrors.SafeCode(err))
			}
		}
	}
	return port.ResultOk()
}

func (a *MyArticleService) UpdateArticleTopAndFeatured(c port.Request) port.ResultVO {
	var articleTopFeaturedVO port.ArticleTopFeaturedVO
	if err := c.Bind(&articleTopFeaturedVO); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	articlebase, err := a.articleRepository().UpdateTopAndFeatured(c.Context(), articleTopFeaturedVO.Id, articleTopFeaturedVO.IsTop, articleTopFeaturedVO.IsFeatured)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return port.ResultOk()
		}
		return port.ResultFromError(err)
	}
	if articlebase.Id != 0 {
		marsha, err := json.Marshal(articlebase)
		if err != nil {
			slog.ErrorContext(c.Context(), "marshal article cache failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFail()
		}
		if a.cache != nil {
			if err := a.cache.Set(c.Context(), strconv.Itoa(articlebase.Id), marsha, 0); err != nil {
				slog.WarnContext(c.Context(), "cache article failed", "error_code", apperrors.SafeCode(err))
			}
		}
	}
	return port.ResultOk()
}

func (a *MyArticleService) UpdateArticleDelete(c port.Request) port.ResultVO {
	var deleteVO port.DeleteVO
	if err := c.Bind(&deleteVO); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if deleteVO.IsDelete != 0 && deleteVO.IsDelete != 1 {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := validateArticleIndexJobIDs(deleteVO.Ids); err != nil {
		return port.ResultFromError(err)
	}
	if err := a.articleRepository().UpdateDelete(c.Context(), deleteVO.Ids, deleteVO.IsDelete); err != nil {
		return port.ResultFromError(err)
	}
	action, event := articleIndexMutationForDelete(deleteVO.IsDelete)
	if err := a.enqueueArticleIndexJobs(c.Context(), deleteVO.Ids, action, event, 0, deleteVO.IsDelete, time.Now().UTC()); err != nil {
		slog.ErrorContext(c.Context(), "enqueue article delete-state jobs failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFromError(err)
	}
	if deleteVO.IsDelete == 0 {
		if err := a.enqueueContentUnderstandingJobs(c.Context(), deleteVO.Ids, time.Now().UTC()); err != nil {
			slog.ErrorContext(c.Context(), "enqueue restored content understanding jobs failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFromError(err)
		}
	}
	return port.ResultOk()
}

func (a *MyArticleService) DeleteArticles(c port.Request) port.ResultVO {
	var ids []int
	if err := c.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := validateArticleIndexJobIDs(ids); err != nil {
		return port.ResultFromError(err)
	}
	if err := a.articleRepository().Delete(c.Context(), ids); err != nil {
		return port.ResultFromError(err)
	}
	if err := a.enqueueArticleIndexJobs(c.Context(), ids, port.ArticleIndexDelete, port.ArticleDeleted, 0, 1, time.Now().UTC()); err != nil {
		slog.ErrorContext(c.Context(), "enqueue article deletion jobs failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (a *MyArticleService) SaveArticleImages(c port.Request) port.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		slog.ErrorContext(c.Context(), "read article image failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	ref, err := uploadMultipart(c.Context(), a.storage, file, "articles/")
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(ref.URL)
}

func (a *MyArticleService) GetArticleBackById(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("articleId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	article, categoryName, tagNames, err := a.articleRepository().GetAdminArticle(c.Context(), id)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return port.ResultOkWithData(port.ArticleAdminViewDTO{})
		}
		return port.ResultFromError(err)
	}
	var articleAdminViewDTO port.ArticleAdminViewDTO
	marshal, err := json.Marshal(article)
	if err != nil {
		slog.ErrorContext(c.Context(), "marshal admin article failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	err = json.Unmarshal(marshal, &articleAdminViewDTO)
	if err != nil {
		slog.ErrorContext(c.Context(), "decode admin article failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	articleAdminViewDTO.CategoryName = categoryName
	if len(tagNames) == 0 {
		articleAdminViewDTO.TagNames = list.New()
	} else {
		articleAdminViewDTO.TagNames = tagNames
	}
	return port.ResultOkWithData(articleAdminViewDTO)
}

func (a *MyArticleService) ImportArticles(c port.Request) port.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	filename := file.Filename
	index := strings.LastIndex(filename, ".")
	if index <= 0 || index == len(filename)-1 {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	articleTitle := filename[:index]
	content, err := file.Open()
	if err != nil {
		slog.ErrorContext(c.Context(), "open imported article failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	defer content.Close()
	all, err := io.ReadAll(content)
	if err != nil {
		slog.ErrorContext(c.Context(), "read imported article failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	articleVO := port.ArticleVO{
		ArticleTitle:   articleTitle,
		ArticleContent: string(all),
		Status:         3,
	}
	c.Set("articleVO", articleVO)
	return a.SaveOrUpdateArticle(c)
}

func (a *MyArticleService) ExportArticles(c port.Request) port.ResultVO {
	var iDs []int
	err := c.Bind(&iDs)
	if err != nil {
		return port.ResultFailWithMessage("导出文章失败")
	}
	articles, err := a.articleRepository().Export(c.Context(), iDs)
	if err != nil {
		return port.ResultFromError(err)
	}
	var urls []string
	for _, v := range articles {
		ref, err := uploadNamed(c.Context(), a.storage, bytes.NewReader([]byte(v.ArticleContent)), v.ArticleTitle+".md", "markdown/")
		if err != nil {
			return port.ResultFromError(err)
		}
		urls = append(urls, ref.URL)
	}
	return port.ResultOkWithData(urls)
}

func (a *MyArticleService) ListArticlesBySearch(c port.Request) port.ResultVO {
	keywords := strings.TrimSpace(c.Query("keywords"))
	if keywords == "" {
		return port.ResultOk()
	}
	if a.search == nil {
		return port.ResultFromError(apperrors.Unavailable("article.search", nil))
	}
	mode, err := port.NormalizeSearchMode(c.Query("mode"))
	if err != nil {
		return port.ResultFromError(apperrors.Invalid("article.search.mode", err.Error()))
	}
	filter, err := articleSearchFilter(c)
	if err != nil {
		return port.ResultFromError(apperrors.Invalid("article.search.filter", err.Error()))
	}
	var hits []port.ArticleSearchHit
	if filteredSearcher, ok := a.search.(port.ArticleFilteredModeSearcher); ok {
		hits, err = filteredSearcher.SearchWithModeAndFilter(c.Context(), keywords, mode, filter)
	} else if modeSearcher, ok := a.search.(port.ArticleModeSearcher); ok && filter.Empty() {
		hits, err = modeSearcher.SearchWithMode(c.Context(), keywords, mode)
	} else if mode == port.SearchModeKeyword && filter.Empty() {
		hits, err = a.search.Search(c.Context(), keywords)
	} else {
		err = apperrors.Unavailable("article.search.mode", nil)
	}
	if err != nil {
		return port.ResultFromError(err)
	}
	articleSearchDTOs := make([]port.ArticleSearchDTO, 0, len(hits))
	for _, hit := range hits {
		dto := port.ArticleSearchDTO{
			ArticleSearch:      hit.ArticleSearch,
			HighlightedTitle:   hit.HighlightedTitle,
			HighlightedContent: hit.HighlightedContent,
			Source:             hit.Source,
			Relevance:          hit.Relevance,
		}
		if hit.HighlightedTitle != "" {
			dto.ArticleTitle = hit.HighlightedTitle
		}
		if hit.HighlightedContent != "" {
			dto.ArticleContent = hit.HighlightedContent
		}
		articleSearchDTOs = append(articleSearchDTOs, dto)
	}

	return port.ResultOkWithData(articleSearchDTOs)
}

func articleSearchFilter(c port.Request) (port.KnowledgeFilter, error) {
	category := strings.TrimSpace(c.Query("category"))
	if category == "" {
		category = c.Query("categoryName")
	}
	tags := append([]string(nil), c.QueryArray("tag")...)
	tags = append(tags, c.QueryArray("tags")...)
	if len(tags) == 0 {
		if tag := c.Query("tagName"); tag != "" {
			tags = []string{tag}
		}
	}
	return port.ParseKnowledgeFilter(port.KnowledgeFilterInput{
		Category: category,
		Tags:     tags,
		Year:     c.Query("year"),
		From:     c.Query("from"),
		To:       c.Query("to"),
	})
}
