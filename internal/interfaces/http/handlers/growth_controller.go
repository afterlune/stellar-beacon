package api

import (
	"net/http"

	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func TrackGrowthEvent(c *gin.Context) {
	writeGrowthResult(c, growthService.Track(c))
}

func GetGrowthSummary(c *gin.Context) {
	c.JSON(http.StatusOK, growthService.Summary(c))
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
