package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListCategories
// @Summary		 分类模块
// @Description 获取所有分类
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/categories [GET]
func ListCategories(c *gin.Context) {
	c.JSON(http.StatusOK, categoryService.ListCategories())
}

// ListCategoriesAdmin
// @Summary		 分类模块
// @Description 查看后台分类列表
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/categories [GET]
func ListCategoriesAdmin(c *gin.Context) {
	c.JSON(http.StatusOK, categoryService.ListCategoriesAdmin(c))
}

// ListCategoriesAdminBySearch
// @Summary		 分类模块
// @Description 搜索文章分类
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/categories/search [GET]
func ListCategoriesAdminBySearch(c *gin.Context) {
	c.JSON(http.StatusOK, categoryService.ListCategoriesAdminBySearch(c))
}

// DeleteCategories
// @Summary		 分类模块
// @Description 删除分类
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/categories [DELETE]
func DeleteCategories(c *gin.Context) {
	c.JSON(http.StatusOK, categoryService.DeleteCategories(c))
}

// SaveOrUpdateCategory
// @Summary		 分类模块
// @Description 添加或修改分类
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/categories [POST]
func SaveOrUpdateCategory(c *gin.Context) {
	c.JSON(http.StatusOK, categoryService.SaveOrUpdateCategory(c))
}
