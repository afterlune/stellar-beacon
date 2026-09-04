package api

import (
	"benetnasch/app/application/service"
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type externalVideoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// GetVideos returns only public, non-deleted video metadata. The CSP is
// emitted together with the allowlisted frame origins so a browser never has
// to trust an arbitrary iframe URL from the request body.
// @Summary      视频列表
// @Description  返回本地视频和白名单外链视频
// @Success      200 {object} model.ResultVO
// @Router       /videos [GET]
func GetVideos(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if agentSafetySwitch != nil {
		stopped, err := agentSafetySwitch.IsStopped(c.Request.Context())
		if err != nil || stopped {
			setVideoSecurityHeaders(c, nil)
			c.JSON(http.StatusOK, model.ResultFailWithMessage("视频暂未公开"))
			return
		}
	}
	if !agentFeatureFlags.Videos {
		setVideoSecurityHeaders(c, nil)
		c.JSON(http.StatusOK, model.ResultFailWithMessage("视频暂未公开"))
		return
	}
	if videoService == nil {
		setVideoSecurityHeaders(c, nil)
		c.JSON(http.StatusOK, model.ResultFailWithMessage("视频暂未公开"))
		return
	}
	setVideoSecurityHeaders(c, videoService.AllowedFrameOrigins())
	c.JSON(http.StatusOK, videoService.List(c.Request.Context(), service.VideoQuery{
		Current: c.Query("current"),
		Size:    c.Query("size"),
	}))
}

// UploadVideo stores a validated local video and then creates its public
// metadata. The request body is bounded before multipart parsing.
// @Summary      上传视频
// @Description  上传不超过 500MB 的视频文件
// @Accept       multipart/form-data
// @Param        file formData file true "video file"
// @Param        title formData string false "title"
// @Param        description formData string false "description"
// @Success      200 {object} model.ResultVO
// @Router       /admin/videos/upload [POST]
func UploadVideo(c *gin.Context) {
	if !agentFeatureFlags.Videos || videoService == nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("视频暂未开放上传"))
		return
	}
	if c.Request != nil && c.Request.Body != nil {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, port.MaxVideoBytes+1<<20)
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResultFromError(errors.Invalid("video.upload.request", "video file is required")))
		return
	}
	c.JSON(http.StatusOK, videoService.Upload(c.Request.Context(), service.VideoUploadInput{
		File:        file,
		Title:       c.PostForm("title"),
		Description: c.PostForm("description"),
	}))
}

// CreateExternalVideo registers an HTTPS URL only when its origin is in the
// configured allowlist. It never fetches the URL server-side.
// @Summary      添加外链视频
// @Description  添加 HTTPS 白名单外链，不在服务端抓取远程内容
// @Accept       json
// @Success      200 {object} model.ResultVO
// @Router       /admin/videos/external [POST]
func CreateExternalVideo(c *gin.Context) {
	if !agentFeatureFlags.Videos || videoService == nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("视频暂未开放管理"))
		return
	}
	var request externalVideoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.ResultFromError(errors.Invalid("video.external.request", "request body is invalid")))
		return
	}
	c.JSON(http.StatusOK, videoService.CreateExternal(c.Request.Context(), service.ExternalVideoInput{
		Title:       request.Title,
		Description: request.Description,
		URL:         request.URL,
	}))
}

// DeleteVideo soft-deletes a video so old object URLs are not silently
// reassigned to another record.
// @Summary      删除视频
// @Description  软删除视频元数据
// @Success      200 {object} model.ResultVO
// @Router       /admin/videos/{id} [DELETE]
func DeleteVideo(c *gin.Context) {
	if !agentFeatureFlags.Videos || videoService == nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("视频暂未开放管理"))
		return
	}
	c.JSON(http.StatusOK, videoService.Delete(c.Request.Context(), c.Param("id")))
}

func setVideoSecurityHeaders(c *gin.Context, origins []string) {
	if c == nil {
		return
	}
	frameSources := []string{"'self'"}
	for _, origin := range origins {
		if strings.TrimSpace(origin) != "" {
			frameSources = append(frameSources, origin)
		}
	}
	c.Header("Content-Security-Policy", fmt.Sprintf("default-src 'none'; frame-src %s; object-src 'none'; base-uri 'none'", strings.Join(frameSources, " ")))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "no-referrer")
}
