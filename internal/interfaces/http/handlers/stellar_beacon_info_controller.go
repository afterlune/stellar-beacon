package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Report
// @Summary		 星际信标信息
// @Description 上报访客信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/reports/visit [POST]
func Report(c *gin.Context) {
	c.JSON(http.StatusOK, stellarBeaconService.Report(c.Request))
}

// GetBlogHomeInfo
// @Summary		 星际信标信息
// @Description 获取系统信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/ [GET]
func GetBlogHomeInfo(c *gin.Context) {
	c.JSON(http.StatusOK, stellarBeaconService.GetBlogHomeInfo(c.Request.Context()))
}

// GetBlogBackInfo
// @Summary		 星际信标信息
// @Description 获取系统后台信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/dashboard [GET]
func GetBlogBackInfo(c *gin.Context) {
	c.JSON(http.StatusOK, stellarBeaconService.GetBlogBackInfo(c.Request.Context()))
}

// GetDashboardAnalytics returns the data used by the new admin dashboard.
// @Router /v1/admin/dashboard/analytics [GET]
func GetDashboardAnalytics(c *gin.Context) {
	c.JSON(http.StatusOK, stellarBeaconService.GetDashboardAnalytics(c.Request.Context(), c.DefaultQuery("range", "7d"), c.DefaultQuery("areaType", "users")))
}

// UpdateWebsiteConfig
// @Summary		 星际信标信息
// @Description 更新网站配置
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/site [PUT]
func UpdateWebsiteConfig(c *gin.Context) {
	c.JSON(http.StatusOK, stellarBeaconService.UpdateWebsiteConfig(c))
}

// GetWebsiteConfig
// @Summary		 星际信标信息
// @Description 获取网站配置
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/site [GET]
func GetWebsiteConfig(c *gin.Context) {
	c.JSON(http.StatusOK, stellarBeaconService.GetWebsiteConfig(c.Request.Context()))
}

// GetAbout
// @Summary		 星际信标信息
// @Description 查看关于我信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/about [GET]
func GetAbout(c *gin.Context) {
	c.JSON(http.StatusOK, stellarBeaconService.GetAbout(c.Request.Context()))
}

// UpdateAbout
// @Summary		 星际信标信息
// @Description 修改关于我信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/about [PUT]
func UpdateAbout(c *gin.Context) {
	c.JSON(http.StatusOK, stellarBeaconService.UpdateAbout(c))
}
