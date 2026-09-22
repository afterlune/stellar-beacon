package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SubscribeCollection(c *gin.Context) {
	c.JSON(http.StatusOK, collectionSubService.Subscribe(c))
}
func UnsubscribeCollection(c *gin.Context) {
	c.JSON(http.StatusOK, collectionSubService.Unsubscribe(c))
}
func MuteCollectionSubscription(c *gin.Context) {
	c.JSON(http.StatusOK, collectionSubService.SetMuted(c))
}
func GetCollectionSubscriptionStatus(c *gin.Context) {
	c.JSON(http.StatusOK, collectionSubService.GetStatus(c))
}
func ListCollectionSubscriptions(c *gin.Context) {
	c.JSON(http.StatusOK, collectionSubService.ListSubscriptions(c))
}
func ListCollectionSubscriptionFeed(c *gin.Context) {
	c.JSON(http.StatusOK, collectionSubService.ListFeed(c))
}
