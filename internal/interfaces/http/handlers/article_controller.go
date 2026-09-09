package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListTopAndFeaturedArticles
// @Summary		 文章模块
// @Description  获取置顶和推荐文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/featured [GET]
func ListTopAndFeaturedArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListTopAndFeaturedArticles(c))
}

// ListArticles
// @Summary		 文章模块
// @Description  获取所有文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles [GET]
func ListArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticles(c))
}

// GetArticlesByCategoryId
// @Summary		 文章模块
// @Description  根据分类id获取文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/by-category [GET]
func GetArticlesByCategoryId(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticlesByCategoryId(c))
}

// GetArticleById
// @Summary		 文章模块
// @Description 根据id获取文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/{articleId} [GET]
func GetArticleById(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.GetArticleById(c))
}

// AccessArticle
// @Summary		 文章模块
// @Description 校验文章访问密码
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/{articleId}/access [POST]
func AccessArticle(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.AccessArticle(c))
}

// ListArticlesByTagId
// @Summary		 文章模块
// @Description 根据标签id获取文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/by-tag [GET]
func ListArticlesByTagId(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticlesByTagId(c))
}

// ListArchives
// @Summary		 文章模块
// @Description 获取所有文章归档
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/archives [GET]
func ListArchives(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArchives(c))
}

// ListArticlesAdmin
// @Summary		 文章模块
// @Description 获取后台文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles [GET]
func ListArticlesAdmin(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticlesAdmin(c))
}

// SaveOrUpdateArticle
// @Summary		 文章模块
// @Description 保存和修改文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles [POST]
func SaveOrUpdateArticle(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.SaveOrUpdateArticle(c))
}

// UpdateArticleTopAndFeatured
// @Summary		 文章模块
// @Description 修改文章是否置顶和推荐
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/featured [PUT]
func UpdateArticleTopAndFeatured(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.UpdateArticleTopAndFeatured(c))
}

// UpdateArticleDelete
// @Summary		 文章模块
// @Description 删除或者恢复文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/trash [PUT]
func UpdateArticleDelete(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.UpdateArticleDelete(c))
}

// DeleteArticles
// @Summary		 文章模块
// @Description 物理删除文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/batch-delete [DELETE]
func DeleteArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.DeleteArticles(c))
}

// SaveArticleImages
// @Summary		 文章模块
// @Description 上传文章图片
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/images [POST]
func SaveArticleImages(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.SaveArticleImages(c))
}

// GetArticleBackById
// @Summary		 文章模块
// @Description 根据id查看后台文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/{articleId} [GET]
func GetArticleBackById(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.GetArticleBackById(c))
}

// ImportArticles
// @Summary		 文章模块
// @Description 导入文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/import [POST]
func ImportArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ImportArticles(c))
}

// ExportArticles
// @Summary		 文章模块
// @Description 导出文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/articles/export [POST]
func ExportArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ExportArticles(c))
}

// ListArticlesBySearch
// @Summary		 文章模块
// @Description 搜索文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/articles/search [GET]
func ListArticlesBySearch(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticlesBySearch(c))
}
