package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListTopAndFeaturedArticles
// @Summary		 文章模块
// @Description  获取置顶和推荐文章
// @Success		 200	{object} model.ResultVO
// @Router       /articles/topAndFeatured [GET]
func ListTopAndFeaturedArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListTopAndFeaturedArticles(applicationRequest(c)))
}

// ListArticles
// @Summary		 文章模块
// @Description  获取所有文章
// @Success		 200	{object} model.ResultVO
// @Router       /articles/all [GET]
func ListArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticles(applicationRequest(c)))
}

// GetArticlesByCategoryId
// @Summary		 文章模块
// @Description  根据分类id获取文章
// @Success		 200	{object} model.ResultVO
// @Router       /articles/categoryId [GET]
func GetArticlesByCategoryId(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticlesByCategoryId(applicationRequest(c)))
}

// GetArticleById
// @Summary		 文章模块
// @Description 根据id获取文章
// @Success		 200	{object} model.ResultVO
// @Router       /articles/:articleId [GET]
func GetArticleById(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.GetArticleById(applicationRequest(c)))
}

// AccessArticle
// @Summary		 文章模块
// @Description 校验文章访问密码
// @Success		 200	{object} model.ResultVO
// @Router       /articles/access [POST]
func AccessArticle(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.AccessArticle(applicationRequest(c)))
}

// ListArticlesByTagId
// @Summary		 文章模块
// @Description 根据标签id获取文章
// @Success		 200	{object} model.ResultVO
// @Router       /articles/tagId [GET]
func ListArticlesByTagId(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticlesByTagId(applicationRequest(c)))
}

// ListArchives
// @Summary		 文章模块
// @Description 获取所有文章归档
// @Success		 200	{object} model.ResultVO
// @Router       /archives/all [GET]
func ListArchives(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArchives(applicationRequest(c)))
}

// ListArticlesAdmin
// @Summary		 文章模块
// @Description 获取后台文章
// @Success		 200	{object} model.ResultVO
// @Router       /admin/articles [GET]
func ListArticlesAdmin(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticlesAdmin(applicationRequest(c)))
}

// SaveOrUpdateArticle
// @Summary		 文章模块
// @Description 保存和修改文章
// @Success		 200	{object} model.ResultVO
// @Router       /admin/articles [POST]
func SaveOrUpdateArticle(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.SaveOrUpdateArticle(applicationRequest(c)))
}

// UpdateArticleTopAndFeatured
// @Summary		 文章模块
// @Description 修改文章是否置顶和推荐
// @Success		 200	{object} model.ResultVO
// @Router       /admin/articles/topAndFeatured [PUT]
func UpdateArticleTopAndFeatured(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.UpdateArticleTopAndFeatured(applicationRequest(c)))
}

// UpdateArticleDelete
// @Summary		 文章模块
// @Description 删除或者恢复文章
// @Success		 200	{object} model.ResultVO
// @Router       /admin/articles [PUT]
func UpdateArticleDelete(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.UpdateArticleDelete(applicationRequest(c)))
}

// DeleteArticles
// @Summary		 文章模块
// @Description 物理删除文章
// @Success		 200	{object} model.ResultVO
// @Router       /admin/articles/delete [DELETE]
func DeleteArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.DeleteArticles(applicationRequest(c)))
}

// SaveArticleImages
// @Summary		 文章模块
// @Description 上传文章图片
// @Success		 200	{object} model.ResultVO
// @Router       /admin/articles/images [POST]
func SaveArticleImages(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.SaveArticleImages(applicationRequest(c)))
}

// GetArticleBackById
// @Summary		 文章模块
// @Description 根据id查看后台文章
// @Success		 200	{object} model.ResultVO
// @Router       /admin/articles/:articleId [GET]
func GetArticleBackById(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.GetArticleBackById(applicationRequest(c)))
}

// ImportArticles
// @Summary		 文章模块
// @Description 导入文章
// @Success		 200	{object} model.ResultVO
// @Router       /admin/articles/import [POST]
func ImportArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ImportArticles(applicationRequest(c)))
}

// ExportArticles
// @Summary		 文章模块
// @Description 导出文章
// @Success		 200	{object} model.ResultVO
// @Router       /admin/articles/export [POST]
func ExportArticles(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ExportArticles(applicationRequest(c)))
}

// ListArticlesBySearch
// @Summary		 文章模块
// @Description 搜索文章
// @Param        keywords query string false "关键词"
// @Param        mode query string false "搜索模式：keyword、hybrid、semantic"
// @Param        category query string false "分类名"
// @Param        tag query string false "标签名，可重复传入"
// @Param        tags query string false "逗号分隔的标签名"
// @Param        year query int false "发表年份"
// @Param        from query string false "起始时间，RFC3339 或 YYYY-MM-DD"
// @Param        to query string false "结束时间（半开区间），RFC3339 或 YYYY-MM-DD"
// @Success		 200	{object} model.ResultVO
// @Router       /articles/search [GET]
func ListArticlesBySearch(c *gin.Context) {
	c.JSON(http.StatusOK, articleService.ListArticlesBySearch(applicationRequest(c)))
}
