package service

import (
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type MediaService interface {
	List(c *gin.Context) model.ResultVO
	Upload(c *gin.Context) model.ResultVO
	Delete(c *gin.Context) model.ResultVO
	Proxy(c *gin.Context)
}

type MyMediaService struct {
	storage port.ObjectStorage
}

func NewMediaService(storage port.ObjectStorage) *MyMediaService {
	return &MyMediaService{storage: storage}
}

func (m *MyMediaService) mediaStorage() (port.MediaStorage, bool) {
	storage, ok := m.storage.(port.MediaStorage)
	return storage, ok
}

func (m *MyMediaService) List(c *gin.Context) model.ResultVO {
	storage, ok := m.mediaStorage()
	if !ok {
		return model.ResultFailWithMessage("当前对象存储不支持图片资源列表")
	}
	prefix := strings.TrimSpace(c.Query("prefix"))
	objects, err := storage.List(c.Request.Context(), prefix)
	if err != nil {
		return model.ResultFromError(err)
	}
	assets := make([]model.MediaAssetDTO, 0, len(objects))
	for _, object := range objects {
		if !isImageObject(object.Key, object.ContentType) {
			continue
		}
		assets = append(assets, model.MediaAssetDTO{
			Key:          object.Key,
			URL:          object.URL,
			Name:         filepath.Base(object.Key),
			Size:         object.Size,
			ContentType:  object.ContentType,
			LastModified: object.LastModified,
			Deletable:    strings.HasPrefix(object.Key, "media/"),
		})
	}
	sort.SliceStable(assets, func(i, j int) bool {
		return assets[i].LastModified.After(assets[j].LastModified)
	})
	current := positiveInt(c.Query("current"), 1)
	size := positiveInt(c.Query("size"), 24)
	if size > 100 {
		size = 100
	}
	start := (current - 1) * size
	if start > len(assets) {
		start = len(assets)
	}
	end := start + size
	if end > len(assets) {
		end = len(assets)
	}
	page := assets[start:end]
	return model.ResultOkWithData(model.PageResultDTO{Records: page, Count: len(assets), Page: current, PageSize: size})
}

func (m *MyMediaService) Upload(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return model.ResultFailWithMessage("请选择图片文件")
	}
	if !isImageObject(file.Filename, file.Header.Get("Content-Type")) {
		return model.ResultFailWithMessage("只支持图片文件")
	}
	ref, err := uploadMultipart(c.Request.Context(), m.storage, file, "media/")
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.MediaAssetDTO{
		Key:       ref.Key,
		URL:       ref.URL,
		Name:      filepath.Base(ref.Key),
		Deletable: true,
	})
}

func (m *MyMediaService) Delete(c *gin.Context) model.ResultVO {
	storage, ok := m.mediaStorage()
	if !ok {
		return model.ResultFailWithMessage("当前对象存储不支持图片资源删除")
	}
	var keys []string
	if err := c.ShouldBindJSON(&keys); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if len(keys) == 0 {
		return model.ResultFailWithMessage("请选择要删除的图片")
	}
	for _, key := range keys {
		if !strings.HasPrefix(strings.TrimSpace(key), "media/") {
			return model.ResultFailWithMessage("已被业务内容使用的图片不可删除")
		}
	}
	if err := storage.Delete(c.Request.Context(), keys); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

// Proxy makes legacy public OSS images renderable in browsers that block an
// attachment response as CORB. The host allowlist is intentionally narrow:
// this endpoint must never become a general-purpose SSRF proxy.
func (m *MyMediaService) Proxy(c *gin.Context) {
	rawURL := strings.TrimSpace(c.Query("url"))
	parsed, err := url.Parse(rawURL)
	if err != nil || !isAllowedMediaProxyHost(parsed) {
		c.Status(http.StatusBadRequest)
		return
	}
	target := normalizeLegacyMediaURL(parsed)
	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	// The legacy bucket uses a referer allowlist. Forward the browser's
	// already-established page origin so old images remain readable without
	// turning this endpoint into an unrestricted header relay.
	if referer := c.GetHeader("Referer"); isHTTPURL(referer) {
		request.Header.Set("Referer", referer)
	} else if origin := c.GetHeader("Origin"); isHTTPURL(origin) {
		request.Header.Set("Referer", origin+"/")
	}
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		c.Status(http.StatusBadGateway)
		return
	}
	contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		contentType = mime.TypeByExtension(filepath.Ext(parsed.Path))
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		c.Status(http.StatusUnsupportedMediaType)
		return
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(c.Writer, io.LimitReader(response.Body, 32<<20))
}

func isAllowedMediaProxyHost(value *url.URL) bool {
	if value == nil || (value.Scheme != "http" && value.Scheme != "https") {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(value.Hostname(), "."))
	return host == "i.example.invalid" || host == "aliyuncs.com" || strings.HasSuffix(host, ".aliyuncs.com")
}

func normalizeLegacyMediaURL(value *url.URL) *url.URL {
	copyValue := *value
	host := strings.ToLower(strings.TrimSuffix(copyValue.Hostname(), "."))
	if host == "i.example.invalid" || host == "benetnasch.oss-cn-shanghai.aliyuncs.com" {
		copyValue.Scheme = "http"
		copyValue.Host = "example-bucket.oss-cn-shanghai.aliyuncs.com"
	}
	return &copyValue
}

func isHTTPURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsed.Hostname() != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func isImageObject(name, contentType string) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "image/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	return mime.TypeByExtension(ext) != "" && strings.HasPrefix(mime.TypeByExtension(ext), "image/")
}

func positiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

var _ MediaService = (*MyMediaService)(nil)
