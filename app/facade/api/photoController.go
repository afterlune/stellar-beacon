package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// SavePhotosAlbumCover
// @Summary		 照片模块
// @Description  上传照片
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos/upload [POST]
func SavePhotosAlbumCover(c *gin.Context) {
	c.JSON(http.StatusOK, photoService.SavePhotosAlbumCover(c))
}

// ListPhotos
// @Summary		 照片模块
// @Description  根据相册id获取照片列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos [GET]
func ListPhotos(c *gin.Context) {
	c.JSON(http.StatusOK, photoService.ListPhotos(c))
}

// UpdatePhoto
// @Summary		 照片模块
// @Description  更新照片信息
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos [PUT]
func UpdatePhoto(c *gin.Context) {
	c.JSON(http.StatusOK, photoService.UpdatePhoto(c))
}

// SavePhotos
// @Summary		 照片模块
// @Description  保存照片
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos [POST]
func SavePhotos(c *gin.Context) {
	c.JSON(http.StatusOK, photoService.SavePhotos(c))
}

// UpdatePhotosAlbum
// @Summary		 照片模块
// @Description  移动照片相册
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos/album [PUT]
func UpdatePhotosAlbum(c *gin.Context) {
	c.JSON(http.StatusOK, photoService.UpdatePhotosAlbum(c))
}

// UpdatePhotoDelete
// @Summary		 照片模块
// @Description  更新照片删除状态
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos/delete [PUT]
func UpdatePhotoDelete(c *gin.Context) {
	c.JSON(http.StatusOK, photoService.UpdatePhotoDelete(c))
}

// DeletePhotos
// @Summary		 照片模块
// @Description  删除照片
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/photos [DELETE]
func DeletePhotos(c *gin.Context) {
	c.JSON(http.StatusOK, photoService.DeletePhotos(c))
}

// ListPhotosByAlbumId
// @Summary		 照片模块
// @Description  根据相册id查看照片列表
// @Success		 200	{object}	model.ResultVO
// @Router       /albums/:albumId/photos [GET]
func ListPhotosByAlbumId(c *gin.Context) {
	c.JSON(http.StatusOK, photoService.ListPhotosByAlbumId(c))
}
