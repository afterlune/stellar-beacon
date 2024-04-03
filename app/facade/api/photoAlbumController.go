package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// SavePhotoAlbumCover
// @Summary		 相册模块
// @Description  上传相册封面
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos/albums/upload [POST]
func SavePhotoAlbumCover(c *gin.Context) {
	c.JSON(http.StatusOK, photoAlbumService.SavePhotoAlbumCover(c))
}

// SaveOrUpdatePhotoAlbum
// @Summary		 相册模块
// @Description  保存或更新相册
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos/albums [POST]
func SaveOrUpdatePhotoAlbum(c *gin.Context) {
	c.JSON(http.StatusOK, photoAlbumService.SaveOrUpdatePhotoAlbum(c))
}

// ListPhotoAlbumBacks
// @Summary		 相册模块
// @Description  查看后台相册列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos/albums [GET]
func ListPhotoAlbumBacks(c *gin.Context) {
	c.JSON(http.StatusOK, photoAlbumService.ListPhotoAlbumBacks(c))
}

// ListPhotoAlbumBackInfos
// @Summary		 相册模块
// @Description  获取后台相册列表信息
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos/albums/info [GET]
func ListPhotoAlbumBackInfos(c *gin.Context) {
	c.JSON(http.StatusOK, photoAlbumService.ListPhotoAlbumBackInfos())
}

// GetPhotoAlbumBackById
// @Summary		 相册模块
// @Description  根据id获取后台相册信息
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos/albums/:albumId/info [GET]
func GetPhotoAlbumBackById(c *gin.Context) {
	c.JSON(http.StatusOK, photoAlbumService.GetPhotoAlbumBackById(c))
}

// DeletePhotoAlbumById
// @Summary		 相册模块
// @Description  根据id删除相册
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos/albums/:albumId [GET]
func DeletePhotoAlbumById(c *gin.Context) {
	c.JSON(http.StatusOK, photoAlbumService.DeletePhotoAlbumById(c))
}

// ListPhotoAlbums
// @Summary		 相册模块
// @Description  获取相册列表
// @Success		 200	{object}	model.ResultVO
// @Router       /photos/albums [GET]
func ListPhotoAlbums(c *gin.Context) {
	c.JSON(http.StatusOK, photoAlbumService.ListPhotoAlbums())
}
