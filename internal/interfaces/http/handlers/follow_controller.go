package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func FollowAuthor(c *gin.Context)         { c.JSON(http.StatusOK, followService.Follow(c)) }
func UnfollowAuthor(c *gin.Context)       { c.JSON(http.StatusOK, followService.Unfollow(c)) }
func ListFollowingAuthors(c *gin.Context) { c.JSON(http.StatusOK, followService.ListFollowing(c)) }
func ListAuthorFollowers(c *gin.Context)  { c.JSON(http.StatusOK, followService.ListFollowers(c)) }
func ListFollowingFeed(c *gin.Context)    { c.JSON(http.StatusOK, followService.ListFeed(c)) }
func ListFollowNotifications(c *gin.Context) {
	c.JSON(http.StatusOK, followService.ListNotifications(c))
}
func GetUnreadFollowNotifications(c *gin.Context) {
	c.JSON(http.StatusOK, followService.UnreadNotificationCount(c))
}
func MarkFollowNotificationsRead(c *gin.Context) {
	c.JSON(http.StatusOK, followService.MarkNotificationsRead(c))
}
