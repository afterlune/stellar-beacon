package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAgentReviewPolicy
// @Summary      Agent 审核策略
// @Description  获取版本化审核安全策略；人工审核开关始终为必选
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/review-policy [GET]
func GetAgentReviewPolicy(c *gin.Context) {
	c.JSON(http.StatusOK, agentReviewPolicyService.Get(applicationRequest(c)))
}

// UpdateAgentReviewPolicy
// @Summary      Agent 审核策略
// @Description  局部更新审核安全策略，使用版本号防止覆盖并发修改
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/review-policy [PATCH]
func UpdateAgentReviewPolicy(c *gin.Context) {
	c.JSON(http.StatusOK, agentReviewPolicyService.Update(applicationRequest(c)))
}
