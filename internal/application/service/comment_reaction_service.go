package service

import (
	"context"
	"strconv"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type CommentReactionService interface {
	ToggleCommentReaction(context.Context, int, int, bool) (bool, int, error)
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

func (s *MyCommentReactionService) ToggleCommentReaction(ctx context.Context, commentID, userInfoID int, active bool) (bool, int, error) {
	if commentID <= 0 {
		return false, 0, apperrors.Invalid("comment_reaction.toggle", "invalid comment")
	}
	if userInfoID <= 0 {
		return false, 0, apperrors.New(apperrors.KindUnauthorized, "comment_reaction.user", nil)
	}
	allowed, err := allowRateLimit(ctx, s.limiter, "comment-reaction:", strconv.Itoa(userInfoID), 60, time.Minute)
	if err != nil {
		return false, 0, apperrors.Unavailable("comment_reaction.rate_limit", err)
	}
	if !allowed {
		return false, 0, apperrors.Invalid("comment_reaction.rate_limit", "too many reactions")
	}
	return s.repo.Set(ctx, commentID, userInfoID, active)
}

var _ CommentReactionService = (*MyCommentReactionService)(nil)
