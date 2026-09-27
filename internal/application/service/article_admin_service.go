package service

import (
	"bytes"
	"container/list"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

func (a *MyArticleService) ListArticlesAdmin(c *gin.Context) model.ResultVO {
	var conditionVO model.ConditionVO
	err := c.ShouldBindQuery(&conditionVO)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.ArticleFilter{
		Current:          conditionVO.Current,
		Size:             conditionVO.Size,
		Keywords:         conditionVO.Keywords,
		IsDelete:         conditionVO.IsDelete,
		Status:           conditionVO.Status,
		ModerationStatus: conditionVO.ModerationStatus,
		Category:         conditionVO.CategoryId,
		Type:             conditionVO.Type,
		Tag:              conditionVO.TagId,
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
		previous, previousErr := a.articleRepository().GetArticleRecord(c.Request.Context(), article.Id)
		if previousErr != nil {
			return model.ResultFromError(previousErr)
		}
		if previous.UserId != dto.UserInfoId {
			return model.ResultFromError(apperrors.New(apperrors.KindForbidden, "article.save", nil))
		}
		previousStatus = previous.Status
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
		a.syncArticleSearch(c.Request.Context(), articlebase.Id)
	}
	return model.ResultOk()
}

func (a *MyArticleService) UpdateArticleTopAndFeatured(c *gin.Context) model.ResultVO {
	var articleTopFeaturedVO model.ArticleTopFeaturedVO
	if err := c.ShouldBind(&articleTopFeaturedVO); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if articleTopFeaturedVO.IsTop == 1 || articleTopFeaturedVO.IsFeatured == 1 {
		current, err := a.articleRepository().GetArticleRecord(c.Request.Context(), articleTopFeaturedVO.Id)
		if err != nil {
			return model.ResultFromError(err)
		}
		if current.IsDelete != 0 || current.Status != 1 || current.ModerationStatus != "visible" {
			return model.ResultFailWithMessage("只有公开且审核可见的文章可以置顶或推荐")
		}
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
		a.syncArticleSearch(c.Request.Context(), articlebase.Id)
	}
	return model.ResultOk()
}

func (a *MyArticleService) UpdateArticleDelete(c *gin.Context) model.ResultVO {
	var deleteVO model.DeleteVO
	if err := c.ShouldBind(&deleteVO); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	if err := a.requireOwnedArticles(c, user.UserInfoId, deleteVO.Ids); err != nil {
		return model.ResultFromError(err)
	}
	if err := a.articleRepository().UpdateDelete(c.Request.Context(), deleteVO.Ids, deleteVO.IsDelete); err != nil {
		return model.ResultFromError(err)
	}
	// Recycled or restored articles must not keep a stale public body.
	for _, id := range deleteVO.Ids {
		a.evictArticleCache(c.Request.Context(), strconv.Itoa(id))
	}
	a.syncArticleSearch(c.Request.Context(), deleteVO.Ids...)
	return model.ResultOk()
}

func (a *MyArticleService) DeleteArticles(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	if err := a.requireOwnedArticles(c, user.UserInfoId, ids); err != nil {
		return model.ResultFromError(err)
	}
	if err := a.articleRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	a.syncArticleSearch(c.Request.Context(), ids...)
	return model.ResultOk()
}

func (a *MyArticleService) requireOwnedArticles(c *gin.Context, userID int, ids []int) error {
	if len(ids) == 0 {
		return apperrors.Invalid("article.owner", "article id is required")
	}
	for _, id := range ids {
		if id <= 0 {
			return apperrors.Invalid("article.owner", "article id is invalid")
		}
		article, err := a.articleRepository().GetArticleRecord(c.Request.Context(), id)
		if err != nil {
			return err
		}
		if article.UserId != userID {
			return apperrors.New(apperrors.KindForbidden, "article.owner", nil)
		}
	}
	return nil
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
