package api

import (
	"net/http"
	"strconv"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func SubscribeCollection(c *gin.Context) {
	userID, collectionID, ok := collectionSubscriptionRequestIDs(c)
	if !ok {
		return
	}
	err := collectionSubService.Subscribe(c.Request.Context(), userID, collectionID)
	if apperrors.IsKind(err, apperrors.KindNotFound) {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("书单不存在或不可订阅"))
		return
	}
	if apperrors.IsKind(err, apperrors.KindValidation) {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("不能订阅自己的书单"))
		return
	}
	writeCollectionSubscriptionError(c, err)
}

func UnsubscribeCollection(c *gin.Context) {
	userID, collectionID, ok := collectionSubscriptionRequestIDs(c)
	if !ok {
		return
	}
	err := collectionSubService.Unsubscribe(c.Request.Context(), userID, collectionID)
	writeCollectionSubscriptionError(c, err)
}

func MuteCollectionSubscription(c *gin.Context) {
	userID, collectionID, ok := collectionSubscriptionRequestIDs(c)
	if !ok {
		return
	}
	var request struct {
		Muted *int `json:"muted" form:"muted"`
	}
	if err := c.ShouldBind(&request); err != nil || request.Muted == nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	err := collectionSubService.SetMuted(c.Request.Context(), userID, collectionID, *request.Muted == 1)
	if apperrors.IsKind(err, apperrors.KindNotFound) {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("还没有订阅这个书单"))
		return
	}
	writeCollectionSubscriptionError(c, err)
}

func GetCollectionSubscriptionStatus(c *gin.Context) {
	userID, collectionID, ok := collectionSubscriptionRequestIDs(c)
	if !ok {
		return
	}
	status, err := collectionSubService.GetStatus(c.Request.Context(), userID, collectionID)
	if err != nil {
		writeCollectionSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(status))
}

func ListCollectionSubscriptions(c *gin.Context) {
	listCollectionSubscriptionPage(c, false)
}

func ListCollectionSubscriptionFeed(c *gin.Context) {
	listCollectionSubscriptionPage(c, true)
}

func listCollectionSubscriptionPage(c *gin.Context, feed bool) {
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
	if feed {
		records, count, err := collectionSubService.ListFeed(c.Request.Context(), userID, current, size)
		if err != nil {
			writeCollectionSubscriptionError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size}))
		return
	}
	records, count, err := collectionSubService.ListSubscriptions(c.Request.Context(), userID, current, size)
	if err != nil {
		writeCollectionSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size}))
}

func collectionSubscriptionRequestIDs(c *gin.Context) (int, int, bool) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return 0, 0, false
	}
	collectionID, err := strconv.Atoi(c.Param("collectionId"))
	if err != nil || collectionID <= 0 {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return 0, 0, false
	}
	return userID, collectionID, true
}

func writeCollectionSubscriptionError(c *gin.Context, err error) {
	if err == nil {
		c.JSON(http.StatusOK, model.ResultOk())
		return
	}
	c.JSON(http.StatusOK, model.ResultFromError(err))
}
