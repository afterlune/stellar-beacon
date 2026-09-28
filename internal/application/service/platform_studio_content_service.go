package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func (s *MyPlatformService) ListOwnedArticles(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	filter, err := studioFilter(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	articles, count, err := s.platformRepo().ListOwnedArticles(c.Request.Context(), user.UserInfoId, filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: articles, Count: count})
}

func (s *MyPlatformService) GetOwnedArticle(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "articleId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	article, err := s.platformRepo().GetOwnedArticle(c.Request.Context(), user.UserInfoId, id)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(article)
}

func (s *MyPlatformService) SaveOwnedArticle(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioArticleVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	status, ok := visibilityStatus(vo.Visibility, true)
	if !ok {
		return model.ResultFailWithMessage("可见性参数不正确")
	}
	title := strings.TrimSpace(vo.ArticleTitle)
	if title == "" {
		return model.ResultFailWithMessage("文章标题不能为空")
	}
	if len([]rune(title)) > 50 {
		return model.ResultFailWithMessage("文章标题不能超过 50 字")
	}
	if len([]rune(vo.ArticleCover)) > 1024 {
		return model.ResultFailWithMessage("文章封面地址过长")
	}
	if len([]rune(vo.OriginalUrl)) > 255 {
		return model.ResultFailWithMessage("原文链接不能超过 255 字")
	}
	if len([]rune(vo.Password)) > 255 {
		return model.ResultFailWithMessage("访问密码不能超过 255 字")
	}
	if vo.Type != 0 && (vo.Type < 1 || vo.Type > 3) {
		return model.ResultFailWithMessage("文章类型不正确")
	}
	if strings.TrimSpace(vo.ArticleContentHTML) != "" {
		vo.ArticleContentHTML = sanitizeArticleHTML(vo.ArticleContentHTML)
		if !articleHTMLHasContent(vo.ArticleContentHTML) {
			return model.ResultFailWithMessage("文章内容不能为空")
		}
		vo.ArticleContent = vo.ArticleContentHTML
	} else if strings.TrimSpace(vo.ArticleContent) == "" {
		return model.ResultFailWithMessage("文章内容不能为空")
	}
	if len([]rune(vo.ArticleContent)) > 100000 {
		return model.ResultFailWithMessage("文章内容不能超过 100000 字")
	}
	article := entity.TArticle{
		Id: vo.Id, UserId: user.UserInfoId, ArticleCover: strings.TrimSpace(vo.ArticleCover), ArticleTitle: title,
		ArticleContent: vo.ArticleContent, ArticleContentHTML: vo.ArticleContentHTML, SeriesId: vo.SeriesId,
		SeriesOrder: vo.SeriesOrder, Status: status, Type: vo.Type, Password: strings.TrimSpace(vo.Password), OriginalUrl: strings.TrimSpace(vo.OriginalUrl),
	}
	if article.Type == 0 {
		article.Type = 1
	}
	if status != 1 {
		article.Password = ""
	}
	if status == 4 {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(vo.ScheduledAt))
		if err != nil || !parsed.After(time.Now()) {
			return model.ResultFailWithMessage("定时发布时间必须晚于当前时间")
		}
		article.ScheduledAt = parsed
	}
	saved, err := s.platformRepo().SaveOwnedArticle(c.Request.Context(), user.UserInfoId, article, vo.CategoryId, vo.TagIds)
	if err != nil {
		return model.ResultFromError(err)
	}
	if s.cache != nil && saved.Id > 0 {
		_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(saved.Id))
	}
	return model.ResultOkWithData(map[string]int{"id": saved.Id})
}

func (s *MyPlatformService) DeleteOwnedArticles(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedArticles(c.Request.Context(), user.UserInfoId, ids); err != nil {
		return model.ResultFromError(err)
	}
	for _, id := range ids {
		if s.cache != nil {
			_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(id))
		}
	}
	return model.ResultOk()
}

func (s *MyPlatformService) Upload(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	kind := strings.TrimSpace(c.Query("kind"))
	prefix := "studio/articles/"
	switch kind {
	case "article-cover", "cover":
		prefix = "articles/covers/"
	case "article-inline":
		prefix = "articles/inline/"
	case "talk-image", "talk":
		prefix = "talks/"
	case "series-cover":
		prefix = "series/covers/"
	case "photo":
		prefix = "photos/"
	case "album-cover":
		prefix = "photos/albums/"
	case "avatar":
		prefix = "avatar/"
	}
	ref, err := uploadMultipart(c.Request.Context(), s.storage, file, prefix)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(ref.URL)
}

func (s *MyPlatformService) ListOwnedTalks(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	filter, err := studioFilter(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	talks, count, err := s.platformRepo().ListOwnedTalks(c.Request.Context(), user.UserInfoId, filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: talks, Count: count})
}

func (s *MyPlatformService) GetOwnedTalk(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "talkId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	talk, err := s.platformRepo().GetOwnedTalk(c.Request.Context(), user.UserInfoId, id)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(talk)
}

func (s *MyPlatformService) SaveOwnedTalk(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioTalkVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	status, ok := visibilityStatus(vo.Visibility, false)
	if !ok {
		return model.ResultFailWithMessage("可见性参数不正确")
	}
	content := strings.TrimSpace(vo.Content)
	if content == "" {
		return model.ResultFailWithMessage("说说内容不能为空")
	}
	if len([]rune(content)) > 2000 {
		return model.ResultFailWithMessage("说说内容不能超过 2000 字")
	}
	images, err := normalizeTalkImages(strings.TrimSpace(vo.Images))
	if err != nil {
		return model.ResultFailWithMessage("说说图片格式不正确")
	}
	if len([]rune(images)) > 2500 {
		return model.ResultFailWithMessage("说说图片地址过长")
	}
	talk, err := s.platformRepo().SaveOwnedTalk(c.Request.Context(), entity.TTalk{
		Id: vo.Id, UserId: user.UserInfoId, Content: content, Images: images,
		IsTop: vo.IsTop, Status: status,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(map[string]int{"id": talk.Id})
}

func (s *MyPlatformService) DeleteOwnedTalks(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedTalks(c.Request.Context(), user.UserInfoId, ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyPlatformService) ListOwnedSeries(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	filter, err := studioFilter(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	series, count, err := s.platformRepo().ListOwnedSeries(c.Request.Context(), user.UserInfoId, filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: series, Count: count})
}

func (s *MyPlatformService) GetOwnedSeries(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "seriesId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	series, err := s.platformRepo().GetOwnedSeries(c.Request.Context(), user.UserInfoId, id)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(series)
}

func (s *MyPlatformService) SaveOwnedSeries(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioSeriesVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	status, ok := visibilityStatus(vo.Visibility, false)
	if !ok {
		return model.ResultFailWithMessage("可见性参数不正确")
	}
	name := strings.TrimSpace(vo.SeriesName)
	if name == "" {
		return model.ResultFailWithMessage("系列名称不能为空")
	}
	if len([]rune(name)) > 50 {
		return model.ResultFailWithMessage("系列名称不能超过 50 字")
	}
	description := strings.TrimSpace(vo.SeriesDesc)
	if len([]rune(description)) > 255 {
		return model.ResultFailWithMessage("系列简介不能超过 255 字")
	}
	if len([]rune(vo.Cover)) > 1024 {
		return model.ResultFailWithMessage("系列封面地址过长")
	}
	series, err := s.platformRepo().SaveOwnedSeries(c.Request.Context(), entity.TSeries{
		Id: vo.Id, UserId: user.UserInfoId, SeriesName: name,
		SeriesDesc: description, Cover: strings.TrimSpace(vo.Cover), Status: status,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(map[string]int{"id": series.Id})
}

func (s *MyPlatformService) DeleteOwnedSeries(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "seriesId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedSeries(c.Request.Context(), user.UserInfoId, id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}
