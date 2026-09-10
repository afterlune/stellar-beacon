package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Report
// @Summary		 benetnasch信息
// @Description 上报访客信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/reports/visit [POST]
func Report(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.Report(c.Request))
}

// GetBlogHomeInfo
// @Summary		 benetnasch信息
// @Description 获取系统信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/ [GET]
func GetBlogHomeInfo(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.GetBlogHomeInfo(c.Request.Context()))
}

// GetBlogBackInfo
// @Summary		 benetnasch信息
// @Description 获取系统后台信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/dashboard [GET]
func GetBlogBackInfo(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.GetBlogBackInfo(c.Request.Context()))
}

// GetDashboardAnalytics returns the data used by the new admin dashboard.
// @Router /v1/admin/dashboard/analytics [GET]
func GetDashboardAnalytics(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.GetDashboardAnalytics(c.Request.Context(), c.DefaultQuery("range", "7d"), c.DefaultQuery("areaType", "users")))
}

// UpdateWebsiteConfig
// @Summary		 benetnasch信息
// @Description 更新网站配置
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/site [PUT]
func UpdateWebsiteConfig(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.UpdateWebsiteConfig(c))
}

// GetWebsiteConfig
// @Summary		 benetnasch信息
// @Description 获取网站配置
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/site [GET]
func GetWebsiteConfig(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.GetWebsiteConfig(c.Request.Context()))
}

// GetAbout
// @Summary		 benetnasch信息
// @Description 查看关于我信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/about [GET]
func GetAbout(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.GetAbout(c.Request.Context()))
}

// UpdateAbout
// @Summary		 benetnasch信息
// @Description 修改关于我信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/about [PUT]
func UpdateAbout(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.UpdateAbout(c))
}

// SaveBlogPhotoAlbumCover
// @Summary		 benetnasch信息
// @Description 上传博客配置图片
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/site/images [POST]
func SaveBlogPhotoAlbumCover(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.SaveBlogPhotoAlbumCover(c))
}
