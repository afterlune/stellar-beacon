package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListTopAndFeaturedArticles
// @Summary		 文章模块
// @Description  获取置顶和推荐文章
// @Success		 200	{object}	model.ResultVO
// @Router       /articles/topAndFeatured [GET]
func ListTopAndFeaturedArticles(c *gin.Context) {
	res := service.ListTopAndFeaturedArticles()
	c.JSON(http.StatusOK, res)
}

// ListArticles
// @Summary		 文章模块
// @Description  获取所有文章
// @Success		 200	{object}	model.ResultVO
// @Router       /articles/all [GET]
func ListArticles(c *gin.Context) {
	res := service.ListArticles(c)
	c.JSON(http.StatusOK, res)
}

// GetArticlesByCategoryId
// @Summary		 文章模块
// @Description  根据分类id获取文章
// @Success		 200	{object}	model.ResultVO
// @Router       /articles/categoryId [GET]
func GetArticlesByCategoryId(c *gin.Context) {
	res := service.ListArticlesByCategoryId(c)
	c.JSON(http.StatusOK, res)
}

// GetArticleById
// @Summary		 文章模块
// @Description 根据id获取文章
// @Success		 200	{object}	model.ResultVO
// @Router       /articles/:articleId [GET]
func GetArticleById(c *gin.Context) {
	res := service.GetArticleById(c)
	c.JSON(http.StatusOK, res)
}

// AccessArticle
// @Summary		 文章模块
// @Description 校验文章访问密码
// @Success		 200	{object}	model.ResultVO
// @Router       /articles/access [POST]
func AccessArticle(c *gin.Context) {
	res := service.AccessArticle(c)
	c.JSON(http.StatusOK, res)
}

// ListArticlesByTagId
// @Summary		 文章模块
// @Description 根据标签id获取文章
// @Success		 200	{object}	model.ResultVO
// @Router       /articles/tagId [GET]
func ListArticlesByTagId(c *gin.Context) {
	res := service.ListArticlesByTagId(c)
	c.JSON(http.StatusOK, res)
}

// ListArchives
// @Summary		 文章模块
// @Description 获取所有文章归档
// @Success		 200	{object}	model.ResultVO
// @Router       /archives/all [GET]
func ListArchives(c *gin.Context) {
	res := service.ListArchives(c)
	c.JSON(http.StatusOK, res)
}

// ListArticlesAdmin
// @Summary		 文章模块
// @Description 获取后台文章
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/articles [GET]
func ListArticlesAdmin(c *gin.Context) {
	res := service.ListArticlesAdmin(c)
	c.JSON(http.StatusOK, res)
}

// SaveOrUpdateArticle
// @Summary		 文章模块
// @Description 保存和修改文章
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/articles [POST]
func SaveOrUpdateArticle(c *gin.Context) {
	res := service.SaveOrUpdateArticle(c)
	c.JSON(http.StatusOK, res)
}

// UpdateArticleTopAndFeatured
// @Summary		 文章模块
// @Description 修改文章是否置顶和推荐
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/articles/topAndFeatured [PUT]
func UpdateArticleTopAndFeatured(c *gin.Context) {
	res := service.UpdateArticleTopAndFeatured(c)
	c.JSON(http.StatusOK, res)
}

// UpdateArticleDelete
// @Summary		 文章模块
// @Description 删除或者恢复文章
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/articles [PUT]
func UpdateArticleDelete(c *gin.Context) {
	res := service.UpdateArticleDelete(c)
	c.JSON(http.StatusOK, res)
}

// DeleteArticles
// @Summary		 文章模块
// @Description 物理删除文章
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/articles/delete [DELETE]
func DeleteArticles(c *gin.Context) {
	res := service.DeleteArticles(c)
	c.JSON(http.StatusOK, res)
}

// SaveArticleImages
// @Summary		 文章模块
// @Description 上传文章图片
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/articles/images [POST]
func SaveArticleImages(c *gin.Context) {
	res := service.SaveArticleImages(c)
	c.JSON(http.StatusOK, res)
}

// GetArticleBackById
// @Summary		 文章模块
// @Description 根据id查看后台文章
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/articles/:articleId [GET]
func GetArticleBackById(c *gin.Context) {
	res := service.GetArticleBackById(c)
	c.JSON(http.StatusOK, res)
}

// ImportArticles
// @Summary		 文章模块
// @Description 导入文章
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/articles/import [POST]
func ImportArticles(c *gin.Context) {
	res := service.ImportArticles(c)
	c.JSON(http.StatusOK, res)
}

// ExportArticles
// @Summary		 文章模块
// @Description 导出文章
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/articles/export [POST]
func ExportArticles(c *gin.Context) {
	res := service.ExportArticles(c)
	c.JSON(http.StatusOK, res)
}

// ListArticlesBySearch
// @Summary		 文章模块
// @Description 搜索文章
// @Success		 200	{object}	model.ResultVO
// @Router       /articles/search [GET]
func ListArticlesBySearch(c *gin.Context) {
	res := service.ListArticlesBySearch(c)
	c.JSON(http.StatusOK, res)
}
