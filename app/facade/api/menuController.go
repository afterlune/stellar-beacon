package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListMenus
// @Summary		 菜单模块
// @Description  查看菜单列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/menus [GET]
func ListMenus(c *gin.Context) {
	res := service.ListMenus(c)
	c.JSON(http.StatusOK, res)
}

// SaveOrUpdateMenu
// @Summary		 菜单模块
// @Description  新增或修改菜单
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/menus [POST]
func SaveOrUpdateMenu(c *gin.Context) {
	res := service.SaveOrUpdateMenu(c)
	c.JSON(http.StatusOK, res)
}

// UpdateMenuIsHidden
// @Summary		 菜单模块
// @Description  修改目录是否隐藏
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/menus/isHidden [PUT]
func UpdateMenuIsHidden(c *gin.Context) {
	res := service.UpdateMenuIsHidden(c)
	c.JSON(http.StatusOK, res)
}

// DeleteMenu
// @Summary		 菜单模块
// @Description  删除菜单
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/menus/:menuId [DELETE]
func DeleteMenu(c *gin.Context) {
	res := service.DeleteMenu(c)
	c.JSON(http.StatusOK, res)
}

// ListMenuOptions
// @Summary		 菜单模块
// @Description  查看角色菜单选项
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/role/menus [GET]
func ListMenuOptions(c *gin.Context) {
	res := service.ListMenuOptions()
	c.JSON(http.StatusOK, res)
}

// ListUserMenus
// @Summary		 菜单模块
// @Description  查看当前用户菜单
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/user/menus [GET]
func ListUserMenus(c *gin.Context) {
	res := service.ListUserMenus()
	c.JSON(http.StatusOK, res)
}
