package api

import (
	"benetnasch/app/application/service"
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type createTimeCapsuleRequest struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	DeliverAt string `json:"deliverAt"`
}

// CreateTimeCapsule creates a private draft. The owner is taken from the
// authenticated stable account ID; it is never accepted from JSON.
// @Summary      创建时间胶囊
// @Description  创建只绑定稳定用户 ID 的私密时间胶囊草稿
// @Success      200 {object} model.ResultVO
// @Router       /capsules [POST]
func CreateTimeCapsule(c *gin.Context) {
	if !agentFeatureFlags.Capsules {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("时间胶囊暂未开启"))
		return
	}
	ownerID, err := authenticatedCapsuleOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ResultFromError(err))
		return
	}
	var request createTimeCapsuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.ResultFromError(errors.Invalid("time_capsule.request", "request body is invalid")))
		return
	}
	c.JSON(http.StatusOK, capsuleService.Create(c.Request.Context(), ownerID, service.TimeCapsuleCreateInput{
		Title:     request.Title,
		Content:   request.Content,
		DeliverAt: request.DeliverAt,
	}))
}

// GetTimeCapsule returns the caller's capsule. Sealed content stays hidden
// until delivery; reading a due capsule acknowledges its HTTP delivery and
// moves it to delivered atomically.
// @Summary      查询时间胶囊
// @Description  仅允许所有者查询，封存内容在到期前不会返回
// @Success      200 {object} model.ResultVO
// @Router       /capsules/{id} [GET]
func GetTimeCapsule(c *gin.Context) {
	if !agentFeatureFlags.Capsules {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("时间胶囊暂未开启"))
		return
	}
	ownerID, err := authenticatedCapsuleOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, capsuleService.Get(c.Request.Context(), ownerID, c.Param("id")))
}

// SealTimeCapsule freezes a draft. There is intentionally no edit endpoint:
// once sealed, the user-authored content and recipient identity cannot be
// changed as part of a later Agent or profile update.
// @Summary      封存时间胶囊
// @Description  封存后内容不可编辑，只有到期后所有者可以读取
// @Success      200 {object} model.ResultVO
// @Router       /capsules/{id}/seal [POST]
func SealTimeCapsule(c *gin.Context) {
	if !agentFeatureFlags.Capsules {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("时间胶囊暂未开启"))
		return
	}
	ownerID, err := authenticatedCapsuleOwnerID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, capsuleService.Seal(c.Request.Context(), ownerID, c.Param("id")))
}

func authenticatedCapsuleOwnerID(c *gin.Context) (int, error) {
	if c == nil {
		return 0, errors.New(errors.KindUnauthorized, "time_capsule.owner", nil)
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return 0, errors.New(errors.KindUnauthorized, "time_capsule.owner", nil)
	}
	switch user := value.(type) {
	case port.UserDetailsDTO:
		if user.UserInfoId > 0 {
			return user.UserInfoId, nil
		}
	case *port.UserDetailsDTO:
		if user != nil && user.UserInfoId > 0 {
			return user.UserInfoId, nil
		}
	case map[string]interface{}:
		if raw, ok := user["userInfoId"]; ok {
			switch value := raw.(type) {
			case float64:
				if int(value) > 0 && value == float64(int(value)) {
					return int(value), nil
				}
			case string:
				if id, parseErr := strconv.Atoi(strings.TrimSpace(value)); parseErr == nil && id > 0 {
					return id, nil
				}
			}
		}
	}
	return 0, errors.New(errors.KindUnauthorized, "time_capsule.owner", nil)
}
