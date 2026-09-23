package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ToggleCommentReaction likes or unlikes a visible comment or reply.
// @Router /v1/auth/me/comment-reactions [PUT]
func ToggleCommentReaction(c *gin.Context) {
	c.JSON(http.StatusOK, commentReactionService.ToggleCommentReaction(c))
}
