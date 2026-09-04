package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListAgentMemoryAssertions lists current durable memory assertions for
// authenticated operators. Public chat never calls this endpoint.
// @Summary      Agent 记忆断言
// @Description  查询带来源、版本、有效期和状态的长期记忆断言
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/memory/assertions [GET]
func ListAgentMemoryAssertions(c *gin.Context) {
	c.JSON(http.StatusOK, agentMemoryService.ListAssertions(applicationRequest(c)))
}

// ListAgentMemoryHistory returns immutable revisions for one assertion.
// @Summary      Agent 记忆历史
// @Description  查询单条长期记忆断言的不可变修订历史
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/memory/assertions/{id}/history [GET]
func ListAgentMemoryHistory(c *gin.Context) {
	c.JSON(http.StatusOK, agentMemoryService.ListHistory(applicationRequest(c)))
}

// RevokeAgentMemoryAssertion explicitly retracts one assertion. It does not
// delete its history and cannot be used by public Agent tools.
// @Summary      撤回 Agent 记忆
// @Description  撤回一条长期记忆断言并保留审计历史
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/memory/assertions/{id} [DELETE]
func RevokeAgentMemoryAssertion(c *gin.Context) {
	c.JSON(http.StatusOK, agentMemoryService.RevokeAssertion(applicationRequest(c)))
}

// ListAgentMemoryConflicts lists open or completed memory conflicts.
// @Summary      Agent 记忆冲突
// @Description  查询需要人工处理的长期记忆冲突及其成员
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/memory/conflicts [GET]
func ListAgentMemoryConflicts(c *gin.Context) {
	c.JSON(http.StatusOK, agentMemoryService.ListConflicts(applicationRequest(c)))
}

// ResolveAgentMemoryConflict explicitly selects a persisted conflict member.
// @Summary      解决 Agent 记忆冲突
// @Description  显式选择冲突赢家，其他成员转为 stale
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/memory/conflicts/{id}/resolve [POST]
func ResolveAgentMemoryConflict(c *gin.Context) {
	c.JSON(http.StatusOK, agentMemoryService.ResolveConflict(applicationRequest(c)))
}

// RejectAgentMemoryConflict rejects all members while retaining their history.
// @Summary      驳回 Agent 记忆冲突
// @Description  驳回冲突集合中的全部断言并保留审计历史
// @Success      200 {object} model.ResultVO
// @Router       /admin/ai/memory/conflicts/{id}/reject [POST]
func RejectAgentMemoryConflict(c *gin.Context) {
	c.JSON(http.StatusOK, agentMemoryService.RejectConflict(applicationRequest(c)))
}
