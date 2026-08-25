package api

import (
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
	c.JSON(http.StatusOK, menuService.ListMenus(c))
}

// SaveOrUpdateMenu
// @Summary		 菜单模块
// @Description  新增或修改菜单
// @Success		 200	{object} model.ResultVO
// @Router       /admin/menus [POST]
func SaveOrUpdateMenu(c *gin.Context) {
	c.JSON(http.StatusOK, menuService.SaveOrUpdateMenu(c))
}

// UpdateMenuIsHidden
// @Summary		 菜单模块
// @Description  修改目录是否隐藏
// @Success		 200	{object} model.ResultVO
// @Router       /admin/menus/isHidden [PUT]
func UpdateMenuIsHidden(c *gin.Context) {
	c.JSON(http.StatusOK, menuService.UpdateMenuIsHidden(c))
}

// DeleteMenu
// @Summary		 菜单模块
// @Description  删除菜单
// @Success		 200	{object} model.ResultVO
// @Router       /admin/menus/:menuId [DELETE]
func DeleteMenu(c *gin.Context) {
	c.JSON(http.StatusOK, menuService.DeleteMenu(c))
}

// ListMenuOptions
// @Summary		 菜单模块
// @Description  查看角色菜单选项
// @Success		 200	{object} model.ResultVO
// @Router       /admin/role/menus [GET]
func ListMenuOptions(c *gin.Context) {
	c.JSON(http.StatusOK, menuService.ListMenuOptions())
}

// ListUserMenus
// @Summary		 菜单模块
// @Description  查看当前用户菜单
// @Success		 200	{object} model.ResultVO
// @Router       /admin/user/menus [GET]
func ListUserMenus(c *gin.Context) {
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	c.JSON(http.StatusOK, menuService.ListUserMenus(dto.UserInfoId))
}
