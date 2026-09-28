package service

import (
	"strconv"
	"strings"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func (s *MyPlatformService) BatchPreviewContent(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioBatchPreviewVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	contentType, ok := studioBatchContentType(vo.Kind)
	if !ok {
		return model.ResultFailWithMessage("内容类型不正确")
	}
	if !validStudioStatus(contentType, vo.Status) {
		return model.ResultFailWithMessage("状态筛选不正确")
	}
	preview, err := s.platformRepo().PreviewOwnedContent(c.Request.Context(), user.UserInfoId, contentType, port.StudioFilter{
		Status: vo.Status, Keywords: strings.TrimSpace(vo.Keywords), SeriesID: vo.SeriesId,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(preview)
}

func (s *MyPlatformService) BatchUpdateContentStatus(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioBatchStatusVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	contentType, ok := studioBatchContentType(vo.Kind)
	if !ok {
		return model.ResultFailWithMessage("内容类型不正确")
	}
	scope, ok := studioBatchScope(vo.Scope, contentType)
	if !ok {
		return model.ResultFailWithMessage("批量选择范围不正确")
	}
	status, ok := visibilityStatus(vo.Visibility, false)
	if !ok {
		return model.ResultFailWithMessage("批量状态仅支持公开、私有或草稿")
	}
	mutation, err := s.platformRepo().BatchUpdateOwnedContentStatus(c.Request.Context(), user.UserInfoId, contentType, scope, status, studioAuditActor(c, user))
	if err != nil {
		return model.ResultFromError(err)
	}
	if contentType == port.StudioContentArticle && s.cache != nil {
		for _, id := range mutation.ContentIDs {
			_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(id))
		}
	}
	if contentType == port.StudioContentArticle {
		s.syncArticleSearch(c.Request.Context(), mutation.ContentIDs...)
	}
	return model.ResultOkWithData(mutation)
}

func (s *MyPlatformService) BatchDeleteContent(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioBatchDeleteVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	contentType, ok := studioBatchContentType(vo.Kind)
	if !ok {
		return model.ResultFailWithMessage("内容类型不正确")
	}
	scope, ok := studioBatchScope(vo.Scope, contentType)
	if !ok {
		return model.ResultFailWithMessage("批量选择范围不正确")
	}
	mutation, err := s.platformRepo().BatchDeleteOwnedContent(c.Request.Context(), user.UserInfoId, contentType, scope, studioAuditActor(c, user))
	if err != nil {
		return model.ResultFromError(err)
	}
	if contentType == port.StudioContentArticle && s.cache != nil {
		for _, id := range mutation.ContentIDs {
			_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(id))
		}
	}
	if contentType == port.StudioContentArticle {
		s.syncArticleSearch(c.Request.Context(), mutation.ContentIDs...)
	}
	return model.ResultOkWithData(mutation)
}

func (s *MyPlatformService) RetryScheduledPublication(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	articleID, err := pathID(c, "articleId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	record, err := s.platformRepo().RetryScheduledPublication(c.Request.Context(), user.UserInfoId, articleID, studioAuditActor(c, user))
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(record)
}

func optionalUserID(c *gin.Context) int {
	if user, ok := currentUser(c); ok {
		return user.UserInfoId
	}
	return 0
}
func studioAuditActor(c *gin.Context, user model.UserDetailsDTO) port.StudioAuditActor {
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = user.IpAddress
	}
	return port.StudioAuditActor{UserID: user.UserInfoId, Nickname: user.Nickname, IPAddress: ip, IPSource: user.IpSource}
}

func studioBatchContentType(value string) (port.StudioContentType, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(port.StudioContentArticle):
		return port.StudioContentArticle, true
	case string(port.StudioContentTalk):
		return port.StudioContentTalk, true
	case string(port.StudioContentSeries):
		return port.StudioContentSeries, true
	default:
		return "", false
	}
}

func studioBatchScope(vo model.StudioBatchScopeVO, contentType port.StudioContentType) (port.StudioBatchScope, bool) {
	mode := strings.ToLower(strings.TrimSpace(vo.Mode))
	scope := port.StudioBatchScope{
		Mode: mode, Status: vo.Status, Keywords: strings.TrimSpace(vo.Keywords), SeriesID: vo.SeriesId,
		MaxID: vo.MaxId, ExcludeIDs: vo.ExcludeIds, ExpectedCount: vo.ExpectedCount,
	}
	switch mode {
	case port.StudioBatchScopeIDs:
		ids, ok := studioBatchIDs(vo.Ids, 500)
		if !ok {
			return port.StudioBatchScope{}, false
		}
		scope.IDs = ids
		return scope, true
	case port.StudioBatchScopeFilter:
		if scope.ExpectedCount <= 0 || scope.MaxID <= 0 || !validStudioStatus(contentType, scope.Status) {
			return port.StudioBatchScope{}, false
		}
		excludes, ok := studioBatchIDs(vo.ExcludeIds, 500)
		if len(vo.ExcludeIds) > 0 && !ok {
			return port.StudioBatchScope{}, false
		}
		scope.ExcludeIDs = excludes
		if contentType != port.StudioContentArticle && scope.SeriesID != 0 {
			return port.StudioBatchScope{}, false
		}
		return scope, true
	default:
		return port.StudioBatchScope{}, false
	}
}

func validStudioStatus(contentType port.StudioContentType, status int) bool {
	if status < 0 {
		return false
	}
	if contentType == port.StudioContentArticle {
		return status <= 4
	}
	return status <= 3
}

func studioBatchIDs(values []int, limit int) ([]int, bool) {
	if len(values) == 0 {
		return []int{}, true
	}
	if limit <= 0 || len(values) > limit {
		return nil, false
	}
	seen := make(map[int]struct{}, len(values))
	ids := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			return nil, false
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		return nil, false
	}
	return ids, true
}
