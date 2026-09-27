package api

import (
	"container/list"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/eternallyzzz/stellar-beacon/internal/application/service"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func publicArticleFailure(err error) model.ResultVO {
	var articleErr *service.PublicArticleError
	if !errors.As(err, &articleErr) {
		return model.ResultFromError(err)
	}
	switch articleErr.Failure {
	case service.PublicArticleAccessDenied:
		return model.ResultFailWithMessage("无权访问")
	case service.PublicArticlePasswordRequired:
		return model.ResultFailWithCodeAndMessage(52003, "")
	case service.PublicArticleAccessCheckFailed, service.PublicArticleGrantFailed:
		return model.ResultFail()
	case service.PublicArticleNotFoundForAccess:
		return model.ResultFailWithMessage("文章不存在")
	case service.PublicArticlePasswordInvalid:
		return model.ResultFailWithMessage("密码错误")
	default:
		return model.ResultFromError(err)
	}
}

func publicArticlePageResult[T any](page service.ArticlePage[T]) model.ResultVO {
	return model.ResultOkWithData(model.PageResultDTO{
		Records: page.Items, Count: page.Total, Page: page.Page, PageSize: page.PageSize,
	})
}

func articleSessionID(c *gin.Context) int {
	value, ok := c.Get("userInfo")
	if !ok {
		return 0
	}
	session, ok := value.(model.UserDetailsDTO)
	if !ok {
		return 0
	}
	return session.Id
}

func articleIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("articleId"))
	return id, err == nil
}

// ListTopAndFeaturedArticles
// @Summary		 文章模块
// @Description  获取置顶和推荐文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/featured [GET]
func ListTopAndFeaturedArticles(c *gin.Context) {
	result, err := publicArticleReader.ListFeatured(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, publicArticleFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{
		TopArticle: result.TopArticle, FeaturedArticles: result.FeaturedArticles,
	}))
}

// ListArticles
// @Summary		 文章模块
// @Description  获取所有文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles [GET]
func ListArticles(c *gin.Context) {
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	page, err := publicArticleReader.List(c.Request.Context(), service.PageQuery{Current: current, Size: size})
	if err != nil {
		c.JSON(http.StatusOK, publicArticleFailure(err))
		return
	}
	c.JSON(http.StatusOK, publicArticlePageResult(page))
}

// GetArticlesByCategoryId
// @Summary		 文章模块
// @Description  根据分类id获取文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/by-category [GET]
func GetArticlesByCategoryId(c *gin.Context) {
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	name := strings.TrimSpace(c.Query("categoryName"))
	if name == "" {
		name = strings.TrimSpace(c.Query("name"))
	}
	query := service.CategoryArticleQuery{Page: service.PageQuery{Current: current, Size: size}, Name: name}
	if name == "" {
		id, err := strconv.Atoi(c.Query("categoryId"))
		if err != nil {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
			return
		}
		query.ID = id
	}
	page, err := publicArticleReader.ListByCategory(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusOK, publicArticleFailure(err))
		return
	}
	c.JSON(http.StatusOK, publicArticlePageResult(page))
}

// GetArticleById
// @Summary		 文章模块
// @Description 根据id获取文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/{articleId} [GET]
func GetArticleById(c *gin.Context) {
	id, ok := articleIDParam(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	article, err := publicArticleReader.Get(c.Request.Context(), id, articleSessionID(c))
	if err != nil {
		c.JSON(http.StatusOK, publicArticleFailure(err))
		return
	}
	if article == nil {
		c.JSON(http.StatusOK, model.ResultOk())
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(*article))
}

// AccessArticle
// @Summary		 文章模块
// @Description 校验文章访问密码
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/{articleId}/access [POST]
func AccessArticle(c *gin.Context) {
	var request model.ArticlePasswordVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFail())
		return
	}
	err := publicArticleReader.GrantPasswordAccess(c.Request.Context(), service.ArticlePasswordAccess{
		ArticleID: request.ArticleId, Password: request.ArticlePassword, UserID: articleSessionID(c),
	})
	if err != nil {
		c.JSON(http.StatusOK, publicArticleFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// ListArticlesByTagId
// @Summary		 文章模块
// @Description 根据标签id获取文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/by-tag [GET]
func ListArticlesByTagId(c *gin.Context) {
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	name := strings.TrimSpace(c.Query("tagName"))
	query := service.TagArticleQuery{Page: service.PageQuery{Current: current, Size: size}, Name: name}
	if name == "" {
		id, err := strconv.Atoi(c.Query("tagId"))
		if err != nil {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
			return
		}
		query.ID = id
	}
	page, err := publicArticleReader.ListByTag(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusOK, publicArticleFailure(err))
		return
	}
	c.JSON(http.StatusOK, publicArticlePageResult(page))
}

// ListArchives
// @Summary		 文章模块
// @Description 获取所有文章归档
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/archives [GET]
func ListArchives(c *gin.Context) {
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	page, err := publicArticleReader.ListArchives(c.Request.Context(), service.PageQuery{Current: current, Size: size})
	if err != nil {
		c.JSON(http.StatusOK, publicArticleFailure(err))
		return
	}
	archives := make([]model.ArchiveDTO, 0, len(page.Items))
	for _, item := range page.Items {
		archives = append(archives, model.ArchiveDTO{Time: item.Time, Articles: item.Articles})
	}
	c.JSON(http.StatusOK, publicArticlePageResult(service.ArticlePage[model.ArchiveDTO]{
		Items: archives, Total: page.Total, Page: page.Page, PageSize: page.PageSize,
	}))
}

// ListArticlesAdmin
// @Summary		 文章模块
// @Description 获取后台文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles [GET]
func ListArticlesAdmin(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	page, err := articleAdminUseCases.ListAdminArticles(c.Request.Context(), port.ArticleFilter{
		Current: current, Size: size, Keywords: query.Keywords, IsDelete: query.IsDelete,
		Status: query.Status, ModerationStatus: query.ModerationStatus, Category: query.CategoryId,
		Type: query.Type, Tag: query.TagId,
	})
	if err != nil {
		c.JSON(http.StatusOK, articleAdminFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{
		Records: page.Items, Count: page.Total, Page: page.Page, PageSize: page.PageSize,
	}))
}

// SaveOrUpdateArticle
// @Summary		 文章模块
// @Description 保存和修改文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles [POST]
func SaveOrUpdateArticle(c *gin.Context) {
	var request model.ArticleVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	userID, response, ok := articleSaveUserID(c)
	if !ok {
		c.JSON(http.StatusOK, response)
		return
	}
	if err := articleAdminUseCases.SaveAdminArticle(c.Request.Context(), articleSaveInput(request, userID)); err != nil {
		c.JSON(http.StatusOK, articleAdminFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// UpdateArticleTopAndFeatured
// @Summary		 文章模块
// @Description 修改文章是否置顶和推荐
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/featured [PUT]
func UpdateArticleTopAndFeatured(c *gin.Context) {
	var request model.ArticleTopFeaturedVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := articleAdminUseCases.SetArticleTopAndFeatured(c.Request.Context(), request.Id, request.IsTop, request.IsFeatured); err != nil {
		c.JSON(http.StatusOK, articleAdminFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// UpdateArticleDelete
// @Summary		 文章模块
// @Description 删除或者恢复文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/trash [PUT]
func UpdateArticleDelete(c *gin.Context) {
	var request model.DeleteVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}
	if err := articleAdminUseCases.TrashArticles(c.Request.Context(), userID, request.Ids, request.IsDelete); err != nil {
		c.JSON(http.StatusOK, articleAdminFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// DeleteArticles
// @Summary		 文章模块
// @Description 物理删除文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/batch-delete [DELETE]
func DeleteArticles(c *gin.Context) {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}
	if err := articleAdminUseCases.DeleteArticles(c.Request.Context(), userID, ids); err != nil {
		c.JSON(http.StatusOK, articleAdminFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// SaveArticleImages
// @Summary		 文章模块
// @Description 上传文章图片
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/images [POST]
func SaveArticleImages(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "read article image failed", "error", err)
		c.JSON(http.StatusOK, model.ResultFail())
		return
	}
	content, err := file.Open()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "open article image failed", "error", err)
		c.JSON(http.StatusOK, model.ResultFail())
		return
	}
	defer content.Close()
	ref, err := articleAdminUseCases.UploadArticleImage(c.Request.Context(), service.ArticleImageUpload{
		Filename: file.Filename, ContentType: file.Header.Get("Content-Type"), Size: file.Size, Content: content,
	})
	if err != nil {
		c.JSON(http.StatusOK, articleAdminFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(ref.URL))
}

// GetArticleBackById
// @Summary		 文章模块
// @Description 根据id查看后台文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/{articleId} [GET]
func GetArticleBackById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("articleId"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	article, found, err := articleAdminUseCases.GetAdminArticle(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, articleAdminFailure(err))
		return
	}
	if !found {
		c.JSON(http.StatusOK, model.ResultOkWithData(model.ArticleAdminViewDTO{}))
		return
	}
	if tags, ok := article.TagNames.([]string); !ok || len(tags) == 0 {
		article.TagNames = list.New()
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(article))
}

// ImportArticles
// @Summary		 文章模块
// @Description 导入文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/import [POST]
func ImportArticles(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	content, err := file.Open()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "open imported article failed", "error", err)
		c.JSON(http.StatusOK, model.ResultFail())
		return
	}
	defer content.Close()
	userID, response, ok := articleSaveUserID(c)
	if !ok {
		c.JSON(http.StatusOK, response)
		return
	}
	if err := articleAdminUseCases.ImportAdminArticle(c.Request.Context(), service.ArticleImportInput{
		Filename: file.Filename, Content: content, UserID: userID,
	}); err != nil {
		c.JSON(http.StatusOK, articleAdminFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// ExportArticles
// @Summary		 文章模块
// @Description 导出文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/export [POST]
func ExportArticles(c *gin.Context) {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("导出文章失败"))
		return
	}
	urls, err := articleAdminUseCases.ExportAdminArticles(c.Request.Context(), ids)
	if err != nil {
		c.JSON(http.StatusOK, articleAdminFailure(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(urls))
}

func articleSaveUserID(c *gin.Context) (int, model.ResultVO, bool) {
	value, ok := c.Get("userInfo")
	if !ok {
		return 0, model.ResultFailWithMessage("用户未登录"), false
	}
	user, ok := value.(model.UserDetailsDTO)
	if !ok {
		return 0, model.ResultFailWithMessage("用户信息无效"), false
	}
	return user.UserInfoId, model.ResultVO{}, true
}

func articleSaveInput(request model.ArticleVO, userID int) service.ArticleSaveInput {
	return service.ArticleSaveInput{
		Article: entity.TArticle{
			Id: request.Id, ArticleCover: request.ArticleCover, ArticleTitle: request.ArticleTitle,
			ArticleContent: request.ArticleContent, ArticleContentHTML: request.ArticleContentHTML,
			SeriesId: request.SeriesId, SeriesOrder: request.SeriesOrder, IsTop: request.IsTop,
			IsFeatured: request.IsFeatured, Status: request.Status, Type: request.Type,
			Password: request.Password, OriginalUrl: request.OriginalUrl,
		},
		CategoryName: request.CategoryName, TagNames: request.TagNames,
		ScheduledAt: request.ScheduledAt, UserID: userID,
	}
}

func articleAdminFailure(err error) model.ResultVO {
	var adminErr *service.ArticleAdminError
	if errors.As(err, &adminErr) {
		switch adminErr.Failure {
		case service.ArticleAdminContentRequired:
			return model.ResultFailWithMessage("文章内容不能为空")
		case service.ArticleAdminPublicRequired:
			return model.ResultFailWithMessage("只有公开且审核可见的文章可以置顶或推荐")
		case service.ArticleAdminImportRead:
			return model.ResultFail()
		}
	}
	return model.ResultFromError(err)
}

// ListArticlesBySearch
// @Summary		 文章模块
// @Description 搜索文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/search [GET]
func ListArticlesBySearch(c *gin.Context) {
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	page, err := publicArticleReader.Search(c.Request.Context(), service.ArticleSearchQuery{
		Page: service.PageQuery{Current: current, Size: size}, Keywords: c.Query("keywords"),
	})
	if err != nil {
		c.JSON(http.StatusOK, publicArticleFailure(err))
		return
	}
	items := make([]model.ArticleSearchDTO, 0, len(page.Items))
	for _, hit := range page.Items {
		items = append(items, model.ArticleSearchDTO{
			ArticleSearch: hit.ArticleSearch, HighlightedTitle: hit.HighlightedTitle,
			HighlightedContent: hit.HighlightedContent,
		})
	}
	c.JSON(http.StatusOK, publicArticlePageResult(service.ArticlePage[model.ArticleSearchDTO]{
		Items: items, Total: page.Total, Page: page.Page, PageSize: page.PageSize,
	}))
}
