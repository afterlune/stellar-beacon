package api

import (
	"net/http"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ToggleCommentReaction likes or unlikes a visible comment or reply.
// @Router /v1/auth/me/comment-reactions [PUT]
func ToggleCommentReaction(c *gin.Context) {
	var request model.CommentReactionToggleVO
	if err := c.ShouldBind(&request); err != nil || request.CommentId <= 0 {
		c.JSON(http.StatusOK, model.ResultFromError(apperrors.Invalid("comment_reaction.toggle", "invalid comment")))
		return
	}
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "comment_reaction.user", nil)))
		return
	}
	active, count, err := commentReactionService.ToggleCommentReaction(c.Request.Context(), request.CommentId, userID, request.Active)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.CommentReactionToggleDTO{Active: active, LikeCount: count}))
}
