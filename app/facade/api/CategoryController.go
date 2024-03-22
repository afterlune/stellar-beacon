package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListCategories
// @Summary		 分类模块
// @Description 获取所有分类
// @Success		 200	{object}	model.ResultVO
// @Router       /categories/all [GET]
func ListCategories(c *gin.Context) {
	res := service.ListCategories()
	c.JSON(http.StatusOK, res)
}

// ListCategoriesAdmin
// @Summary		 分类模块
// @Description 查看后台分类列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/categories [GET]
func ListCategoriesAdmin(c *gin.Context) {
	res := service.ListCategoriesAdmin(c)
	c.JSON(http.StatusOK, res)
}

// ListCategoriesAdminBySearch
// @Summary		 分类模块
// @Description 搜索文章分类
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/categories/search [GET]
func ListCategoriesAdminBySearch(c *gin.Context) {
	res := service.ListCategoriesAdminBySearch(c)
	c.JSON(http.StatusOK, res)
}

// DeleteCategories
// @Summary		 分类模块
// @Description 删除分类
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/categories [DELETE]
func DeleteCategories(c *gin.Context) {
	res := service.DeleteCategories(c)
	c.JSON(http.StatusOK, res)
}

// SaveOrUpdateCategory
// @Summary		 分类模块
// @Description 添加或修改分类
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/categories [POST]
func SaveOrUpdateCategory(c *gin.Context) {
	res := service.SaveOrUpdateCategory(c)
	c.JSON(http.StatusOK, res)
}
