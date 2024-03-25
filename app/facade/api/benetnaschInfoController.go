package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Report
// @Summary		 benetnasch信息
// @Description 上报访客信息
// @Success		 200	{object}	model.ResultVO
// @Router       /report [POST]
func Report(c *gin.Context) {
	res := service.Report(c.Request)
	c.JSON(http.StatusOK, res)
}

// GetBlogHomeInfo
// @Summary		 benetnasch信息
// @Description 获取系统信息
// @Success		 200	{object}	model.ResultVO
// @Router       / [GET]
func GetBlogHomeInfo(c *gin.Context) {
	res := service.GetBlogHomeInfo()
	c.JSON(http.StatusOK, res)
}

// GetBlogBackInfo
// @Summary		 benetnasch信息
// @Description 获取系统后台信息
// @Success		 200	{object}	model.ResultVO
// @Router       /admin [GET]
func GetBlogBackInfo(c *gin.Context) {
	res := service.GetBlogBackInfo()
	c.JSON(http.StatusOK, res)
}

// UpdateWebsiteConfig
// @Summary		 benetnasch信息
// @Description 更新网站配置
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/website/config [PUT]
func UpdateWebsiteConfig(c *gin.Context) {
	res := service.UpdateWebsiteConfig(c)
	c.JSON(http.StatusOK, res)
}

// GetWebsiteConfig
// @Summary		 benetnasch信息
// @Description 获取网站配置
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/website/config [GET]
func GetWebsiteConfig(c *gin.Context) {
	res := service.GetWebsiteConfig()
	c.JSON(http.StatusOK, res)
}

// GetAbout
// @Summary		 benetnasch信息
// @Description 查看关于我信息
// @Success		 200	{object}	model.ResultVO
// @Router       /about [GET]
func GetAbout(c *gin.Context) {
	res := service.GetAbout()
	c.JSON(http.StatusOK, res)
}

// UpdateAbout
// @Summary		 benetnasch信息
// @Description 修改关于我信息
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/about [PUT]
func UpdateAbout(c *gin.Context) {
	res := service.UpdateAbout(c)
	c.JSON(http.StatusOK, res)
}

// SaveBlogPhotoAlbumCover
// @Summary		 benetnasch信息
// @Description 上传博客配置图片
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/config/images [POST]
func SaveBlogPhotoAlbumCover(c *gin.Context) {
	res := service.SaveBlogPhotoAlbumCover(c)
	c.JSON(http.StatusOK, res)
}
