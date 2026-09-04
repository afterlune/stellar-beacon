package api

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListMenus
// @Summary		 菜单模块
// @Description  查看菜单列表
// @Success		 200	{object} model.ResultVO
// @Router       /admin/menus [GET]
func ListMenus(c *gin.Context) {
	c.JSON(http.StatusOK, menuService.ListMenus(applicationRequest(c)))
}

// SaveOrUpdateMenu
// @Summary		 菜单模块
// @Description  新增或修改菜单
// @Success		 200	{object} model.ResultVO
// @Router       /admin/menus [POST]
func SaveOrUpdateMenu(c *gin.Context) {
	c.JSON(http.StatusOK, menuService.SaveOrUpdateMenu(applicationRequest(c)))
}

// UpdateMenuIsHidden
// @Summary		 菜单模块
// @Description  修改目录是否隐藏
// @Success		 200	{object} model.ResultVO
// @Router       /admin/menus/isHidden [PUT]
func UpdateMenuIsHidden(c *gin.Context) {
	c.JSON(http.StatusOK, menuService.UpdateMenuIsHidden(applicationRequest(c)))
}

// DeleteMenu
// @Summary		 菜单模块
// @Description  删除菜单
// @Success		 200	{object} model.ResultVO
// @Router       /admin/menus/:menuId [DELETE]
func DeleteMenu(c *gin.Context) {
	c.JSON(http.StatusOK, menuService.DeleteMenu(applicationRequest(c)))
}

// ListMenuOptions
// @Summary		 菜单模块
// @Description  查看角色菜单选项
// @Success		 200	{object} model.ResultVO
// @Router       /admin/role/menus [GET]
func ListMenuOptions(c *gin.Context) {
	c.JSON(http.StatusOK, menuService.ListMenuOptions(c.Request.Context()))
}

// ListUserMenus
// @Summary		 菜单模块
// @Description  查看当前用户菜单
// @Success		 200	{object} model.ResultVO
// @Router       /admin/user/menus [GET]
func ListUserMenus(c *gin.Context) {
	value, ok := c.Get("userInfo")
	dto, ok := value.(port.UserDetailsDTO)
	if !ok || dto.UserInfoId <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}

	c.JSON(http.StatusOK, menuService.ListUserMenus(c.Request.Context(), dto.UserInfoId))
}
