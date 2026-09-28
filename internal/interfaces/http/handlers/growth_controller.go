package api

import (
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func TrackGrowthEvent(c *gin.Context) {
	var request model.GrowthEventVO
	if err := c.ShouldBindJSON(&request); err != nil {
		writeGrowthResult(c, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	path := strings.TrimSpace(request.Path)
	if path == "" {
		path = c.Request.URL.Path
	}
	rateLimited, err := growthService.Track(c.Request.Context(), request.EventName, request.ArticleId, path, c.ClientIP())
	if err != nil {
		if apperrors.Op(err) == "growth.track.event" {
			writeGrowthResult(c, model.ResultFailWithMessage("不支持的事件类型"))
			return
		}
		writeGrowthResult(c, model.ResultFromError(err))
		return
	}
	if rateLimited {
		writeGrowthResult(c, model.ResultFailWithCodeAndMessage(42900, "请求过于频繁，请稍后再试"))
		return
	}
	writeGrowthResult(c, model.ResultOk())
}

func GetGrowthSummary(c *gin.Context) {
	days := 30
	if raw := c.Query("days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}
	items, err := growthService.Summary(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(items))
}

func SubscribeNewsletter(c *gin.Context) {
	writeGrowthResult(c, newsletterService.Subscribe(c))
}

func ConfirmNewsletter(c *gin.Context) {
	c.JSON(http.StatusOK, newsletterService.Confirm(c))
}

func UnsubscribeNewsletter(c *gin.Context) {
	c.JSON(http.StatusOK, newsletterService.Unsubscribe(c))
}

func ListNewsletterSubscribers(c *gin.Context) {
	c.JSON(http.StatusOK, newsletterService.ListSubscribers(c))
}

func UpdateNewsletterSubscriberStatus(c *gin.Context) {
	c.JSON(http.StatusOK, newsletterService.UpdateSubscriberStatus(c))
}

func ResendNewsletterConfirmation(c *gin.Context) {
	c.JSON(http.StatusOK, newsletterService.ResendConfirmation(c))
}

func ListNewsletterDeliveries(c *gin.Context) {
	c.JSON(http.StatusOK, newsletterService.ListDeliveries(c))
}

func RetryNewsletterDelivery(c *gin.Context) {
	c.JSON(http.StatusOK, newsletterService.RetryDelivery(c))
}

func RetryFailedNewsletterDeliveries(c *gin.Context) {
	c.JSON(http.StatusOK, newsletterService.RetryFailed(c))
}

func GetNewsletterHealth(c *gin.Context) {
	c.JSON(http.StatusOK, newsletterService.Health(c))
}

func GetSystemHealth(c *gin.Context) {
	c.JSON(http.StatusOK, model.ResultOkWithData(systemMonitorService.Snapshot(c.Request.Context())))
}

func GetSystemHealthTrends(c *gin.Context) {
	trends, err := systemMonitorService.Trends(c.Request.Context(), c.Query("range"))
	writeSystemMonitorResult(c, trends, err)
}

func GetSystemHealthTimeline(c *gin.Context) {
	timeline, err := systemMonitorService.Timeline(c.Request.Context(), c.Query("range"))
	writeSystemMonitorResult(c, timeline, err)
}

func ListSystemMonitorIncidents(c *gin.Context) {
	limit := 0
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeSystemMonitorResult(c, nil, apperrors.Invalid("system_monitor.incidents.limit", "limit must be between 1 and 200"))
			return
		}
		limit = parsed
	}
	incidents, err := systemMonitorService.Incidents(c.Request.Context(), c.Query("range"), limit)
	writeSystemMonitorResult(c, incidents, err)
}

func GetSystemMonitorIncident(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("incidentId"), 10, 64)
	if err != nil {
		writeSystemMonitorResult(c, nil, apperrors.Invalid("system_monitor.incident.id", "incidentId must be a positive integer"))
		return
	}
	incident, err := systemMonitorService.Incident(c.Request.Context(), id)
	writeSystemMonitorResult(c, incident, err)
}

func AddSystemMonitorIncidentUpdate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("incidentId"), 10, 64)
	if err != nil {
		writeSystemMonitorResult(c, nil, apperrors.Invalid("system_monitor.incident.id", "incidentId must be a positive integer"))
		return
	}
	var request struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeSystemMonitorResult(c, nil, apperrors.Invalid("system_monitor.incident_update.body", "content is required"))
		return
	}
	user := model.UserDetailsDTO{}
	if value, ok := c.Get("userInfo"); ok {
		user, _ = value.(model.UserDetailsDTO)
	}
	update, err := systemMonitorService.AddIncidentUpdate(c.Request.Context(), id, user, request.Content)
	writeSystemMonitorResult(c, update, err)
}

func writeSystemMonitorResult(c *gin.Context, data any, err error) {
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(data))
}

func RenderAuthorSEO(c *gin.Context) {
	seoService.RenderAuthorHTML(c)
}

func RenderArticleSEO(c *gin.Context) {
	seoService.RenderArticleHTML(c)
}

func RenderSitemap(c *gin.Context) {
	seoService.RenderSitemap(c)
}

func RenderRobots(c *gin.Context) {
	seoService.RenderRobots(c)
}

func RenderFeed(c *gin.Context) {
	seoService.RenderFeed(c)
}

func writeGrowthResult(c *gin.Context, result model.ResultVO) {
	status := http.StatusOK
	if result.Code == 42900 {
		status = http.StatusTooManyRequests
		c.Header("Retry-After", "900")
	}
	c.JSON(status, result)
}
