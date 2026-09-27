package api

import (
	"net/http"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ListCategories
// @Summary         分类模块
// @Description    获取所有分类
// @Success        200 {object} model.ResultVO
// @Router         /v1/public/categories [GET]
func ListCategories(c *gin.Context) {
	data, err := categoryService.ListCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(data))
}

// ListCategoriesAdmin
// @Summary         分类模块
// @Description    查看后台分类列表
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/categories [GET]
func ListCategoriesAdmin(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	records, count, err := categoryService.ListCategoriesAdmin(c.Request.Context(), query.Current, query.Size, query.Keywords)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	if records == nil {
		records = []*port.CategoryAdmin{}
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: int(count)}))
}

// ListCategoriesAdminBySearch
// @Summary         分类模块
// @Description    搜索文章分类
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/categories/search [GET]
func ListCategoriesAdminBySearch(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	data, err := categoryService.ListCategoriesAdminBySearch(c.Request.Context(), query.Keywords)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(data))
}

// DeleteCategories
// @Summary         分类模块
// @Description    删除分类
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/categories [DELETE]
func DeleteCategories(c *gin.Context) {
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
	if err := categoryService.DeleteCategories(c.Request.Context(), userID, ids); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("删除失败，该分类下存在文章"))
			return
		}
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// SaveOrUpdateCategory
// @Summary         分类模块
// @Description    添加或修改分类
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/categories [POST]
func SaveOrUpdateCategory(c *gin.Context) {
	var request model.CategoryVO
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
	category := entity.TCategory{Id: request.Id, UserId: user.UserInfoId, CategoryName: request.CategoryName}
	if err := categoryService.SaveOrUpdateCategory(c.Request.Context(), category); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("分类名已存在"))
			return
		}
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}
