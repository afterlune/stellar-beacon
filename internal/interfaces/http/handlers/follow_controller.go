package api

import (
	"net/http"
	"strconv"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func FollowAuthor(c *gin.Context) {
	userID, authorID, ok := followRequestIDs(c)
	if !ok {
		return
	}
	writeFollowError(c, followService.Follow(c.Request.Context(), userID, authorID))
}

func UnfollowAuthor(c *gin.Context) {
	userID, authorID, ok := followRequestIDs(c)
	if !ok {
		return
	}
	writeFollowError(c, followService.Unfollow(c.Request.Context(), userID, authorID))
}

func ListFollowingAuthors(c *gin.Context) {
	userID, current, size, ok := followPageRequest(c)
	if !ok {
		return
	}
	records, count, err := followService.ListFollowing(c.Request.Context(), userID, current, size)
	if err != nil {
		writeFollowError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size}))
}

func ListAuthorFollowers(c *gin.Context) {
	userID, current, size, ok := followPageRequest(c)
	if !ok {
		return
	}
	records, count, err := followService.ListFollowers(c.Request.Context(), userID, current, size)
	if err != nil {
		writeFollowError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size}))
}

func ListFollowingFeed(c *gin.Context) {
	userID, current, size, ok := followPageRequest(c)
	if !ok {
		return
	}
	records, count, err := followService.ListFeed(c.Request.Context(), userID, c.Query("type"), current, size)
	if err != nil {
		writeFollowError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size}))
}

func ListFollowNotifications(c *gin.Context) {
	userID, current, size, ok := followPageRequest(c)
	if !ok {
		return
	}
	result, err := followService.ListNotifications(c.Request.Context(), userID, c.Query("group"), current, size)
	if err != nil {
		writeFollowError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(struct {
		Records          []port.NotificationItem `json:"records"`
		Count            int                     `json:"count"`
		Page             int                     `json:"page"`
		PageSize         int                     `json:"pageSize"`
		UnreadCount      int                     `json:"unreadCount"`
		TotalUnreadCount int                     `json:"totalUnreadCount"`
		ReadCursor       port.NotificationCursor `json:"readCursor"`
	}{Records: result.Records, Count: result.Count, Page: current, PageSize: size, UnreadCount: result.UnreadCount, TotalUnreadCount: result.TotalUnreadCount, ReadCursor: result.ReadCursor}))
}

func GetUnreadFollowNotifications(c *gin.Context) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		writeFollowUnauthorized(c)
		return
	}
	count, err := followService.UnreadNotificationCount(c.Request.Context(), userID)
	if err != nil {
		writeFollowError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(map[string]int{"count": count}))
}

func MarkFollowNotificationsRead(c *gin.Context) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		writeFollowUnauthorized(c)
		return
	}
	cursor := port.NotificationCursor{}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&cursor); err != nil {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
			return
		}
	}
	if err := followService.MarkNotificationsRead(c.Request.Context(), userID, cursor); err != nil {
		writeFollowError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(map[string]int{"unreadCount": 0}))
}

func followRequestIDs(c *gin.Context) (int, int, bool) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		writeFollowUnauthorized(c)
		return 0, 0, false
	}
	authorID, err := strconv.Atoi(c.Param("authorId"))
	if err != nil || authorID <= 0 {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("关注对象不正确"))
		return 0, 0, false
	}
	return userID, authorID, true
}

func followPageRequest(c *gin.Context) (int, int, int, bool) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		writeFollowUnauthorized(c)
		return 0, 0, 0, false
	}
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return 0, 0, 0, false
	}
	return userID, current, size, true
}

func writeFollowUnauthorized(c *gin.Context) {
	c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
}

func writeFollowError(c *gin.Context, err error) {
	if err == nil {
		c.JSON(http.StatusOK, model.ResultOk())
		return
	}
	switch apperrors.Op(err) {
	case "follow.target":
		c.JSON(http.StatusOK, model.ResultFailWithMessage("关注对象不正确"))
	case "follow.feed.type":
		c.JSON(http.StatusOK, model.ResultFailWithMessage("内容类型不正确"))
	case "follow.notification.group":
		c.JSON(http.StatusOK, model.ResultFailWithMessage("通知类型不正确"))
	default:
		c.JSON(http.StatusOK, model.ResultFromError(err))
	}
}
