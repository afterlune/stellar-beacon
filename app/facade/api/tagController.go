package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetAllTags
// @Summary		 标签模块
// @Description  获取所有标签
// @Success		 200	{object}	model.ResultVO
// @Router       /tags/all [GET]
func GetAllTags(c *gin.Context) {
	c.JSON(http.StatusOK, tagService.ListTags())
}

// GetTopTenTags
// @Summary		 标签模块
// @Description  获取前十个标签
// @Success		 200	{object}	model.ResultVO
// @Router       /tags/topTen [GET]
func GetTopTenTags(c *gin.Context) {
	c.JSON(http.StatusOK, tagService.ListTopTenTags())
}

// ListTagsAdmin
// @Summary		 标签模块
// @Description  查询后台标签列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/tags [GET]
func ListTagsAdmin(c *gin.Context) {
	c.JSON(http.StatusOK, tagService.ListTagsAdmin(c))
}

// ListTagsAdminBySearch
// @Summary		 标签模块
// @Description  搜索文章标签
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/tags/search [GET]
func ListTagsAdminBySearch(c *gin.Context) {
	c.JSON(http.StatusOK, tagService.ListTagsAdminBySearch(c))
}

// SaveOrUpdateTag
// @Summary		 标签模块
// @Description  添加或修改标签
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/tags [POST]
func SaveOrUpdateTag(c *gin.Context) {
	c.JSON(http.StatusOK, tagService.SaveOrUpdateTag(c))
}

// DeleteTag
// @Summary		 标签模块
// @Description  删除标签
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/tags [DELETE]
func DeleteTag(c *gin.Context) {
	c.JSON(http.StatusOK, tagService.DeleteTag(c))
}
