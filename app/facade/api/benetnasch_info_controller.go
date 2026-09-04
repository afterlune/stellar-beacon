package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Report
// @Summary		 benetnasch信息
// @Description 上报访客信息
// @Success		 200	{object} model.ResultVO
// @Router       /report [POST]
func Report(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.Report(c.Request))
}

// GetBlogHomeInfo
// @Summary		 benetnasch信息
// @Description 获取系统信息
// @Success		 200	{object} model.ResultVO
// @Router       / [GET]
func GetBlogHomeInfo(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.GetBlogHomeInfo(c.Request.Context()))
}

// GetBlogBackInfo
// @Summary		 benetnasch信息
// @Description 获取系统后台信息
// @Success		 200	{object} model.ResultVO
// @Router       /admin [GET]
func GetBlogBackInfo(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.GetBlogBackInfo(c.Request.Context()))
}

// UpdateWebsiteConfig
// @Summary		 benetnasch信息
// @Description 更新网站配置
// @Success		 200	{object} model.ResultVO
// @Router       /admin/website/config [PUT]
func UpdateWebsiteConfig(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.UpdateWebsiteConfig(applicationRequest(c)))
}

// GetWebsiteConfig
// @Summary		 benetnasch信息
// @Description 获取网站配置
// @Success		 200	{object} model.ResultVO
// @Router       /admin/website/config [GET]
func GetWebsiteConfig(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.GetWebsiteConfig(c.Request.Context()))
}

// GetAbout
// @Summary		 benetnasch信息
// @Description 查看关于我信息
// @Success		 200	{object} model.ResultVO
// @Router       /about [GET]
func GetAbout(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.GetAbout(c.Request.Context()))
}

// UpdateAbout
// @Summary		 benetnasch信息
// @Description 修改关于我信息
// @Success		 200	{object} model.ResultVO
// @Router       /admin/about [PUT]
func UpdateAbout(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.UpdateAbout(applicationRequest(c)))
}

// SaveBlogPhotoAlbumCover
// @Summary		 benetnasch信息
// @Description 上传博客配置图片
// @Success		 200	{object} model.ResultVO
// @Router       /admin/config/images [POST]
func SaveBlogPhotoAlbumCover(c *gin.Context) {
	c.JSON(http.StatusOK, benetnaschService.SaveBlogPhotoAlbumCover(applicationRequest(c)))
}
