package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetAgentProfile
// @Summary      Agent 人设配置
// @Description  获取当前版本化人设配置
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/profile [GET]
func GetAgentProfile(c *gin.Context) {
	c.JSON(http.StatusOK, agentProfileService.Get(applicationRequest(c)))
}

// UpdateAgentProfile
// @Summary      Agent 人设配置
// @Description  局部更新人设配置；不接受任何 Provider 密钥
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/profile [PATCH]
func UpdateAgentProfile(c *gin.Context) {
	c.JSON(http.StatusOK, agentProfileService.Update(applicationRequest(c)))
}
