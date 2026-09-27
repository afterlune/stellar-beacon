package service

import (
	"strconv"
	"strings"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func (s *MyPlatformService) Moderate(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.ModerationVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if vo.Id <= 0 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	reason := strings.TrimSpace(vo.Reason)
	if vo.Hidden && reason == "" {
		return model.ResultFailWithMessage("隐藏内容时必须填写审核理由")
	}
	if len([]rune(reason)) > 255 {
		return model.ResultFailWithMessage("审核理由不能超过 255 字")
	}
	contentType := strings.TrimSpace(vo.ContentType)
	if err := s.platformRepo().ModerateContent(c.Request.Context(), contentType, vo.Id, user.UserInfoId, vo.Hidden, reason); err != nil {
		return model.ResultFromError(err)
	}
	if s.cache != nil {
		_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(vo.Id))
	}
	if contentType == "article" {
		s.syncArticleSearch(c.Request.Context(), vo.Id)
	}
	return model.ResultOk()
}

func (s *MyPlatformService) DistributeArticle(c *gin.Context) model.ResultVO {
	id, err := pathID(c, "articleId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.DistributionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	current, err := s.articles.GetArticleRecord(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	if current.IsDelete != 0 || current.Status != 1 || current.ModerationStatus != "visible" {
		return model.ResultFailWithMessage("只有公开且审核可见的文章可以推荐或分发")
	}
	if vo.Newsletter && s.newsletter == nil {
		return model.ResultFromError(apperrors.Unavailable("article.distribution", nil))
	}
	featured := 0
	if vo.Featured {
		featured = 1
	}
	article, err := s.articles.UpdateTopAndFeatured(c.Request.Context(), id, 0, featured)
	if err != nil {
		return model.ResultFromError(err)
	}
	if article.Id == 0 {
		return model.ResultFailWithMessage("文章不存在")
	}
	if vo.Newsletter {
		if err := s.newsletter.EnqueueArticle(c.Request.Context(), id); err != nil {
			return model.ResultFromError(err)
		}
	}
	if s.cache != nil {
		_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(id))
	}
	return model.ResultOk()
}
