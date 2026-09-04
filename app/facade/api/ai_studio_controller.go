package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TestAIProvider performs a bounded, configured-provider smoke test. The
// request can select only chat, vision, or embedding; it cannot provide a key
// or endpoint.
// @Summary      AI Provider 烟测
// @Description  测试已配置模型连接，不保存或返回 Provider 凭据
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/providers/test [POST]
func TestAIProvider(c *gin.Context) {
	c.JSON(http.StatusOK, aiProviderProbeService.Probe(applicationRequest(c)))
}

// GetAIOperationalMetrics returns bounded, detail-free runtime aggregates for
// the administration plane. It does not invoke a Provider or expose prompts,
// completions, credentials, request IDs, or query text.
// @Summary      AI/搜索运行时观测
// @Description 返回可供外部监控聚合的脱敏 Provider 与索引指标
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/observability [GET]
func GetAIOperationalMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, operationalObservability.Get(c.Request.Context()))
}

// PreviewWriting generates an unsaved writing preview and creates its pending
// review record for later audit actions.
// @Summary      AI Studio
// @Description  生成写作预览，不自动保存文章
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/writing/preview [POST]
func PreviewWriting(c *gin.Context) {
	c.JSON(http.StatusOK, aiStudioService.PreviewWriting(applicationRequest(c)))
}

// PreviewVision generates an image-understanding preview and records it as a
// pending review. It never publishes or modifies an article.
// @Summary      AI 视觉理解
// @Description  根据图片生成视觉理解预览，结果进入待审核队列
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/vision/preview [POST]
func PreviewVision(c *gin.Context) {
	c.JSON(http.StatusOK, aiVisionService.Preview(applicationRequest(c)))
}

// ListAIReviews lists generated previews awaiting or having completed review.
// @Summary      AI 审核
// @Description  查询 AI 生成物审核记录
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/reviews [GET]
func ListAIReviews(c *gin.Context) {
	c.JSON(http.StatusOK, aiStudioService.ListReviews(applicationRequest(c)))
}

// GetAIReview returns one review with its source, run and publication state.
// @Summary      AI 审核详情
// @Description  查询单条 AI 生成物审核记录
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/reviews/{id} [GET]
func GetAIReview(c *gin.Context) {
	c.JSON(http.StatusOK, aiStudioService.GetReview(applicationRequest(c)))
}

// AcceptAIReview records full acceptance of a generated preview.
// @Summary      AI 审核
// @Description  接受 AI 生成物
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/reviews/{id}/approve [POST]
func AcceptAIReview(c *gin.Context) {
	c.JSON(http.StatusOK, aiStudioService.AcceptReview(applicationRequest(c)))
}

// PartiallyAcceptAIReview records acceptance of an edited subset of a
// generated preview. It does not write the article.
// @Summary      AI 审核
// @Description  部分接受 AI 生成物
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/reviews/{id}/partial [POST]
func PartiallyAcceptAIReview(c *gin.Context) {
	c.JSON(http.StatusOK, aiStudioService.PartiallyAcceptReview(applicationRequest(c)))
}

// RejectAIReview records a rejection and its reason.
// @Summary      AI 审核
// @Description  拒绝 AI 生成物
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/reviews/{id}/reject [POST]
func RejectAIReview(c *gin.Context) {
	c.JSON(http.StatusOK, aiStudioService.RejectReview(applicationRequest(c)))
}

// RegenerateAIReview records that the operator requested another generation.
// The newly generated RunID can be supplied in the request body for audit.
// @Summary      AI 审核
// @Description  记录重新生成操作
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/reviews/{id}/regenerate [POST]
func RegenerateAIReview(c *gin.Context) {
	c.JSON(http.StatusOK, aiStudioService.RegenerateReview(applicationRequest(c)))
}

// ExpireAIReview manually expires a pending review and writes an audit action.
// @Summary      AI 审核过期
// @Description  将待审核记录标记为过期
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/reviews/{id}/expire [POST]
func ExpireAIReview(c *gin.Context) {
	c.JSON(http.StatusOK, aiStudioService.ExpireReview(applicationRequest(c)))
}
