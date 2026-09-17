package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListPublicSeries
// @Summary		 文章系列模块
// @Description  获取公开系列列表
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/series [GET]
func ListPublicSeries(c *gin.Context) {
	c.JSON(http.StatusOK, seriesService.ListPublicSeries(c))
}

// GetPublicSeries
// @Summary		 文章系列模块
// @Description  获取系列详情与按序文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/series/{seriesId} [GET]
func GetPublicSeries(c *gin.Context) {
	c.JSON(http.StatusOK, seriesService.GetPublicSeries(c))
}

// ListAdminSeries
// @Summary		 文章系列模块
// @Description  后台系列列表
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/series [GET]
func ListAdminSeries(c *gin.Context) {
	c.JSON(http.StatusOK, seriesService.ListAdminSeries(c))
}

// ListSeriesOptions
// @Summary		 文章系列模块
// @Description  系列下拉选项
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/series/options [GET]
func ListSeriesOptions(c *gin.Context) {
	c.JSON(http.StatusOK, seriesService.ListSeriesOptions(c))
}

// SaveOrUpdateSeries
// @Summary		 文章系列模块
// @Description  新增或修改系列
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/series [POST]
func SaveOrUpdateSeries(c *gin.Context) {
	c.JSON(http.StatusOK, seriesService.SaveOrUpdateSeries(c))
}

// DeleteSeries
// @Summary		 文章系列模块
// @Description  删除系列并解绑文章
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/series [DELETE]
func DeleteSeries(c *gin.Context) {
	c.JSON(http.StatusOK, seriesService.DeleteSeries(c))
}
