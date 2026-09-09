package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListUserRoles
// @Summary		 角色模块
// @Description  查询用户角色选项
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/users/roles [GET]
func ListUserRoles(c *gin.Context) {
	c.JSON(http.StatusOK, roleService.ListUserRoles())
}

// ListRoles
// @Summary		 角色模块
// @Description  查询角色列表
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/roles [GET]
func ListRoles(c *gin.Context) {
	c.JSON(http.StatusOK, roleService.ListRoles(c))
}

// SaveOrUpdateRole
// @Summary		 角色模块
// @Description  保存或更新角色
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/roles [POST]
func SaveOrUpdateRole(c *gin.Context) {
	c.JSON(http.StatusOK, roleService.SaveOrUpdateRole(c))
}

// DeleteRoles
// @Summary		 角色模块
// @Description  删除角色
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/roles [DELETE]
func DeleteRoles(c *gin.Context) {
	c.JSON(http.StatusOK, roleService.DeleteRoles(c))
}
