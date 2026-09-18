package api

import (
	"net/http"
	"strconv"

	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

// TrackArticleReadSession records a browser-reported reading session. The
// client uses sendBeacon with a small JSON payload and never sends identity
// data beyond the request itself.
// @Router /v1/public/articles/{articleId}/read-sessions [POST]
func TrackArticleReadSession(c *gin.Context) {
	c.JSON(http.StatusOK, contentAnalyticsService.TrackReadSession(c))
}

// TrackArticleContinuationEvent records an anonymous aggregate event from the
// series context or related-reading section on an article page.
// @Router /v1/public/articles/{articleId}/continuation-events [POST]
func TrackArticleContinuationEvent(c *gin.Context) {
	c.JSON(http.StatusOK, contentAnalyticsService.TrackContinuationEvent(c))
}

// GetContentAnalytics returns the site-wide article performance overview.
// @Router /v1/admin/content/analytics [GET]
func GetContentAnalytics(c *gin.Context) {
	c.JSON(http.StatusOK, contentAnalyticsService.GetContentAnalytics(c.Request.Context(), c.DefaultQuery("range", "7d")))
}

// ListContentAnalyticsArticles returns the sortable article performance list.
// @Router /v1/admin/content/analytics/articles [GET]
func ListContentAnalyticsArticles(c *gin.Context) {
	current, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	c.JSON(http.StatusOK, contentAnalyticsService.ListContentAnalyticsArticles(
		c.Request.Context(),
		c.DefaultQuery("range", "7d"),
		c.DefaultQuery("sort", "views"),
		current,
		size,
	))
}

// ListContentContinuationTargets returns the attributed continuation click
// ranking, optionally scoped to one source article.
// @Router /v1/admin/content/analytics/continuation-targets [GET]
func ListContentContinuationTargets(c *gin.Context) {
	current, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	sourceArticleID := 0
	if value := c.Query("sourceArticleId"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
			return
		}
		sourceArticleID = parsed
	}
	c.JSON(http.StatusOK, contentAnalyticsService.ListContinuationTargets(
		c.Request.Context(),
		c.DefaultQuery("range", "7d"),
		c.DefaultQuery("sort", "clicks"),
		sourceArticleID,
		current,
		size,
	))
}

// GetContentAnalyticsArticle returns one article's performance detail.
// @Router /v1/admin/content/analytics/articles/{articleId} [GET]
func GetContentAnalyticsArticle(c *gin.Context) {
	articleID, _ := strconv.Atoi(c.Param("articleId"))
	c.JSON(http.StatusOK, contentAnalyticsService.GetContentAnalyticsArticle(c.Request.Context(), articleID, c.DefaultQuery("range", "7d")))
}
