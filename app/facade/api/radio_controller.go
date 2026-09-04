package api

import (
	"benetnasch/app/facade/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetRadio returns one deterministic, text-only public episode. Browser TTS
// remains an explicit user action in the blog and is never started here.
// @Summary      Benetnasch 电台
// @Description  返回当前节律下的文本节目
// @Success      200 {object} model.ResultVO
// @Router       /radio [GET]
func GetRadio(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if agentSafetySwitch != nil {
		stopped, err := agentSafetySwitch.IsStopped(c.Request.Context())
		if err != nil || stopped {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("电台暂未公开"))
			return
		}
	}
	if !agentFeatureFlags.Radio {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("电台暂未公开"))
		return
	}
	if radioService == nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("电台暂未公开"))
		return
	}
	c.JSON(http.StatusOK, radioService.Current(c.Request.Context()))
}
