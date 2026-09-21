package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SubscribeTopic(c *gin.Context) { c.JSON(http.StatusOK, topicSubscriptionService.Subscribe(c)) }
func UnsubscribeTopic(c *gin.Context) {
	c.JSON(http.StatusOK, topicSubscriptionService.Unsubscribe(c))
}
func MuteTopicSubscription(c *gin.Context) {
	c.JSON(http.StatusOK, topicSubscriptionService.SetMuted(c))
}
func ListTopicSubscriptions(c *gin.Context) {
	c.JSON(http.StatusOK, topicSubscriptionService.ListSubscriptions(c))
}
func ListTopicSubscriptionFeed(c *gin.Context) {
	c.JSON(http.StatusOK, topicSubscriptionService.ListFeed(c))
}
