package api

import (
	"benetnasch/app/application/service"
	"benetnasch/app/domain/errors"
	"benetnasch/app/facade/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAgentFeatures
// @Summary      Agent 功能开关
// @Description  返回公开、非敏感的 Agent rollout flags
// @Success      200 {object} model.ResultVO
// @Router       /agent/features [GET]
//
// GetAgentFeatures returns only public, non-secret rollout flags. A failed
// feature lookup must fail closed in the frontend, so this endpoint has no
// provider or infrastructure details in its response.
func GetAgentFeatures(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	flags := agentFeatureFlags
	if agentSafetySwitch != nil {
		stopped, err := agentSafetySwitch.IsStopped(c.Request.Context())
		if err != nil || stopped {
			flags.PublicChat = false
			flags.Vitals = false
			flags.Galaxy = false
			flags.Dreams = false
			flags.Capsules = false
			flags.Radio = false
			flags.Videos = false
			flags.TTSEnabled = false
		}
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(flags))
}

// GetDreams returns only human-approved dream entries. Candidate generation
// and approval remain behind the admin AI Studio routes.
// @Summary      Benetnasch 梦境
// @Description  返回已通过人工审核的梦境内容
// @Success      200 {object} model.ResultVO
// @Router       /dreams [GET]
func GetDreams(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if agentSafetySwitch != nil {
		stopped, err := agentSafetySwitch.IsStopped(c.Request.Context())
		if err != nil || stopped {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("梦境暂未公开"))
			return
		}
	}
	if !agentFeatureFlags.Dreams {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("梦境暂未公开"))
		return
	}
	c.JSON(http.StatusOK, dreamService.List(c.Request.Context(), service.DreamQuery{
		Current: c.Query("current"),
		Size:    c.Query("size"),
	}))
}

// GetGalaxy returns the public, coordinate-only content projection. The
// application service owns feature gating and query validation; the handler
// deliberately keeps the existing ResultVO response envelope.
// @Summary      星河内容投影
// @Description  返回公开文章的二维坐标和生命阶段
// @Success      200 {object} model.ResultVO
// @Router       /galaxy [GET]
func GetGalaxy(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if agentSafetySwitch != nil {
		stopped, err := agentSafetySwitch.IsStopped(c.Request.Context())
		if err != nil || stopped {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("星河暂未公开"))
			return
		}
	}
	if !agentFeatureFlags.Galaxy {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("星河暂未公开"))
		return
	}
	c.JSON(http.StatusOK, contentGalaxyService.List(c.Request.Context(), service.ContentGalaxyQuery{
		Current:      c.Query("current"),
		Size:         c.Query("size"),
		LifeStage:    c.Query("stage"),
		UpdatedAfter: c.Query("since"),
	}))
}

// GetAgentVitals
// @Summary      Agent 生命体征
// @Description  返回公开聚合状态，不包含访客身份信息
// @Success      200 {object} model.ResultVO
// @Router       /agent/vitals [GET]
//
// GetAgentVitals returns the read-only aggregate used by the public space
// widget. It is independently feature-gated from public chat.
func GetAgentVitals(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if agentSafetySwitch != nil {
		stopped, err := agentSafetySwitch.IsStopped(c.Request.Context())
		if err != nil || stopped {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("生命体征暂未公开"))
			return
		}
	}
	if !agentFeatureFlags.Vitals || agentVitalsProvider == nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("生命体征暂未公开"))
		return
	}
	vitals, err := agentVitalsProvider.Snapshot(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(vitals))
}

type agentEmergencyRequest struct {
	Stopped *bool `json:"stopped"`
}

// SetAgentEmergency changes the cross-instance Agent emergency switch. The
// admin route is intentionally explicit so a normal frontend cannot toggle it
// by accident and so the current state remains auditable at the HTTP edge.
// @Summary      Agent 紧急开关
// @Description  暂停或恢复公开 Agent 能力及自主行为 worker
// @Success      200 {object} model.ResultVO
// @Router       /admin/agent/emergency [PUT]
func SetAgentEmergency(c *gin.Context) {
	if agentSafetySwitch == nil {
		c.JSON(http.StatusOK, model.ResultFromError(errors.Unavailable("agent.safety_switch", nil)))
		return
	}
	var request agentEmergencyRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Stopped == nil {
		c.JSON(http.StatusBadRequest, model.ResultFromError(errors.Invalid("agent.safety_switch.request", "stopped is required")))
		return
	}
	if err := agentSafetySwitch.SetStopped(c.Request.Context(), *request.Stopped); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(map[string]bool{"stopped": *request.Stopped}))
}
