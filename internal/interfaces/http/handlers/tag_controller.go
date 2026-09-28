package api

import (
	"net/http"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// GetAllTags
// @Summary         标签模块
// @Description    获取所有标签
// @Success        200 {object} model.ResultVO
// @Router         /v1/public/tags [GET]
func GetAllTags(c *gin.Context) {
	data, err := tagService.ListTags(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(data))
}

// GetTopTenTags
// @Summary         标签模块
// @Description    获取前十个标签
// @Success        200 {object} model.ResultVO
// @Router         /v1/public/tags/top [GET]
func GetTopTenTags(c *gin.Context) {
	data, err := tagService.ListTopTenTags(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(data))
}

// ListTagsAdmin
// @Summary         标签模块
// @Description    查询后台标签列表
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/tags [GET]
func ListTagsAdmin(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	records, count, err := tagService.ListTagsAdmin(c.Request.Context(), query.Current, query.Size, query.Keywords)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	if records == nil {
		records = []*port.TagAdmin{}
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: int(count)}))
}

// ListTagsAdminBySearch
// @Summary         标签模块
// @Description    搜索文章标签
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/tags/search [GET]
func ListTagsAdminBySearch(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	data, err := tagService.ListTagsAdminBySearch(c.Request.Context(), query.Keywords)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(data))
}

// SaveOrUpdateTag
// @Summary         标签模块
// @Description    添加或修改标签
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/tags [POST]
func SaveOrUpdateTag(c *gin.Context) {
	var request model.TagVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	value, ok := c.Get("userInfo")
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("用户未登录"))
		return
	}
	user, ok := value.(model.UserDetailsDTO)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("用户信息无效"))
		return
	}
	tag := entity.TTag{Id: request.Id, UserId: user.UserInfoId, TagName: request.TagName}
	if err := tagService.SaveOrUpdateTag(c.Request.Context(), tag); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("标签名已存在"))
			return
		}
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// DeleteTag
// @Summary         标签模块
// @Description    删除标签
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/tags [DELETE]
func DeleteTag(c *gin.Context) {
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
	if err := tagService.DeleteTag(c.Request.Context(), userID, ids); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("删除失败，该标签下存在文章"))
			return
		}
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}
