package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListMedia lists image objects for the administration media center.
// @Router /v1/admin/media [GET]
func ListMedia(c *gin.Context) {
	c.JSON(http.StatusOK, mediaService.List(c))
}

// UploadMedia uploads an image into the reusable media area.
// @Router /v1/admin/media/upload [POST]
func UploadMedia(c *gin.Context) {
	c.JSON(http.StatusOK, mediaService.Upload(c))
}

// DeleteMedia removes images that were uploaded into the reusable media area.
// @Router /v1/admin/media [DELETE]
func DeleteMedia(c *gin.Context) {
	c.JSON(http.StatusOK, mediaService.Delete(c))
}

// ProxyMedia renders legacy public OSS images with inline response headers.
// @Router /v1/public/media/proxy [GET]
func ProxyMedia(c *gin.Context) {
	mediaService.Proxy(c)
}
