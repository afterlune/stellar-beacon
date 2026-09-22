package service

import (
	"strconv"
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type CollectionReactionService interface {
	ToggleCollectionReaction(c *gin.Context) model.ResultVO
	GetCollectionReactionState(c *gin.Context) model.ResultVO
}

type MyCollectionReactionService struct {
	repo    port.CollectionReactionRepository
	limiter port.RateLimiter
}

func NewCollectionReactionService(deps CollectionReactionServiceDeps) (*MyCollectionReactionService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyCollectionReactionService{repo: deps.Repo, limiter: deps.Limiter}, nil
}

func (s *MyCollectionReactionService) ToggleCollectionReaction(c *gin.Context) model.ResultVO {
	var vo model.CollectionReactionToggleVO
	if err := c.ShouldBind(&vo); err != nil || vo.CollectionId <= 0 {
		return model.ResultFromError(apperrors.Invalid("collection_reaction.toggle", "invalid collection"))
	}
	userInfoID, ok := currentUserInfoID(c)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "collection_reaction.user", nil))
	}
	if allowed, err := allowRateLimit(c.Request.Context(), s.limiter, "collection-reaction:", strconv.Itoa(userInfoID), 30, time.Minute); err != nil {
		return model.ResultFromError(apperrors.Unavailable("collection_reaction.rate_limit", err))
	} else if !allowed {
		return model.ResultFromError(apperrors.Invalid("collection_reaction.rate_limit", "too many reactions"))
	}
	active, likeCount, err := s.repo.Set(c.Request.Context(), vo.CollectionId, userInfoID, vo.Active)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.CollectionReactionToggleDTO{Active: active, LikeCount: likeCount})
}

func (s *MyCollectionReactionService) GetCollectionReactionState(c *gin.Context) model.ResultVO {
	collectionID, err := strconv.Atoi(c.Query("collectionId"))
	if err != nil || collectionID <= 0 {
		return model.ResultFromError(apperrors.Invalid("collection_reaction.state", "invalid collection"))
	}
	userInfoID, ok := currentUserInfoID(c)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "collection_reaction.user", nil))
	}
	states, err := s.repo.States(c.Request.Context(), userInfoID, []int{collectionID})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.CollectionReactionStateDTO{CollectionId: collectionID, Like: states[collectionID]})
}

var _ CollectionReactionService = (*MyCollectionReactionService)(nil)
