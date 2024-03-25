package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetAllTags
// @Summary		 标签模块
// @Description  获取所有标签
// @Success		 200	{object}	model.ResultVO
// @Router       /tags/all [GET]
func GetAllTags(c *gin.Context) {
	res := service.ListTags()
	c.JSON(http.StatusOK, res)
}

// GetTopTenTags
// @Summary		 标签模块
// @Description  获取前十个标签
// @Success		 200	{object}	model.ResultVO
// @Router       /tags/topTen [GET]
func GetTopTenTags(c *gin.Context) {
	res := service.ListTopTenTags()
	c.JSON(http.StatusOK, res)
}

// ListTagsAdmin
// @Summary		 标签模块
// @Description  查询后台标签列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/tags [GET]
func ListTagsAdmin(c *gin.Context) {
	res := service.ListTagsAdmin(c)
	c.JSON(http.StatusOK, res)
}

// ListTagsAdminBySearch
// @Summary		 标签模块
// @Description  搜索文章标签
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/tags/search [GET]
func ListTagsAdminBySearch(c *gin.Context) {
	res := service.ListTagsAdminBySearch(c)
	c.JSON(http.StatusOK, res)
}

// SaveOrUpdateTag
// @Summary		 标签模块
// @Description  添加或修改标签
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/tags [POST]
func SaveOrUpdateTag(c *gin.Context) {
	res := service.SaveOrUpdateTag(c)
	c.JSON(http.StatusOK, res)
}

// DeleteTag
// @Summary		 标签模块
// @Description  删除标签
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/tags [DELETE]
func DeleteTag(c *gin.Context) {
	res := service.DeleteTag(c)
	c.JSON(http.StatusOK, res)
}
