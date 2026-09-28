package api

import (
	"net/http"
	"strings"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type topicMuteRequest struct {
	Muted *int `json:"muted" form:"muted"`
}

func SubscribeTopic(c *gin.Context) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}
	topicType, topicKey, ok := topicPath(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("话题参数不正确"))
		return
	}
	if err := topicSubscriptionService.Subscribe(c.Request.Context(), userID, topicType, topicKey); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("话题不存在或还没有公开内容"))
			return
		}
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

func UnsubscribeTopic(c *gin.Context) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}
	topicType, topicKey, ok := topicPath(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("话题参数不正确"))
		return
	}
	if err := topicSubscriptionService.Unsubscribe(c.Request.Context(), userID, topicType, topicKey); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

func MuteTopicSubscription(c *gin.Context) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}
	topicType, topicKey, ok := topicPath(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("话题参数不正确"))
		return
	}
	var request topicMuteRequest
	if err := c.ShouldBind(&request); err != nil || request.Muted == nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := topicSubscriptionService.SetMuted(c.Request.Context(), userID, topicType, topicKey, *request.Muted == 1); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("还没有订阅这个话题"))
			return
		}
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

func ListTopicSubscriptions(c *gin.Context) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	records, count, err := topicSubscriptionService.ListSubscriptions(c.Request.Context(), userID, current, size)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size}))
}

func ListTopicSubscriptionFeed(c *gin.Context) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	records, count, err := topicSubscriptionService.ListFeed(c.Request.Context(), userID, current, size)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size}))
}

func topicPath(c *gin.Context) (string, string, bool) {
	topicType := strings.ToLower(strings.TrimSpace(c.Param("topicType")))
	topicKey := strings.TrimSpace(c.Param("topicKey"))
	return topicType, topicKey, port.ValidTopicType(topicType) && topicKey != ""
}
