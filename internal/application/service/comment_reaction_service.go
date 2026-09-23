package service

import (
	"strconv"
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type CommentReactionService interface {
	ToggleCommentReaction(c *gin.Context) model.ResultVO
}

type MyCommentReactionService struct {
	repo    port.CommentReactionRepository
	limiter port.RateLimiter
}

func NewCommentReactionService(deps CommentReactionServiceDeps) (*MyCommentReactionService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyCommentReactionService{repo: deps.Repo, limiter: deps.Limiter}, nil
}

func (s *MyCommentReactionService) ToggleCommentReaction(c *gin.Context) model.ResultVO {
	var vo model.CommentReactionToggleVO
	if err := c.ShouldBind(&vo); err != nil || vo.CommentId <= 0 {
		return model.ResultFromError(apperrors.Invalid("comment_reaction.toggle", "invalid comment"))
	}
	userInfoID, ok := currentUserInfoID(c)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "comment_reaction.user", nil))
	}
	if allowed, err := allowRateLimit(c.Request.Context(), s.limiter, "comment-reaction:", strconv.Itoa(userInfoID), 60, time.Minute); err != nil {
		return model.ResultFromError(apperrors.Unavailable("comment_reaction.rate_limit", err))
	} else if !allowed {
		return model.ResultFromError(apperrors.Invalid("comment_reaction.rate_limit", "too many reactions"))
	}
	active, likeCount, err := s.repo.Set(c.Request.Context(), vo.CommentId, userInfoID, vo.Active)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.CommentReactionToggleDTO{Active: active, LikeCount: likeCount})
}

var _ CommentReactionService = (*MyCommentReactionService)(nil)
