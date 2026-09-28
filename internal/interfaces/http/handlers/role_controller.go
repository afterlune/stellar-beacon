package api

import (
	"net/http"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ListUserRoles
// @Summary         角色模块
// @Description    查询用户角色选项
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/users/roles [GET]
func ListUserRoles(c *gin.Context) {
	roles, err := roleService.ListUserRoles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	dtos, err := mapResponseDTO[[]model.UserRoleDTO](roles)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(dtos))
}

// ListRoles
// @Summary         角色模块
// @Description    查询角色列表
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/roles [GET]
func ListRoles(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	records, count, err := roleService.ListRoles(c.Request.Context(), query.Current, query.Size, query.Keywords)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	dtos, err := mapResponseDTO[[]model.RoleDTO](records)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	if dtos == nil {
		dtos = []model.RoleDTO{}
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)}))
}

// SaveOrUpdateRole
// @Summary         角色模块
// @Description    保存或更新角色
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/roles [POST]
func SaveOrUpdateRole(c *gin.Context) {
	var request model.RoleVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := roleService.SaveOrUpdateRole(c.Request.Context(), request.Id, request.RoleName, request.ResourceIds, request.MenuIds); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("该角色存在"))
			return
		}
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// DeleteRoles
// @Summary         角色模块
// @Description    删除角色
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/roles [DELETE]
func DeleteRoles(c *gin.Context) {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := roleService.DeleteRoles(c.Request.Context(), ids); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("该角色下存在用户"))
			return
		}
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}
