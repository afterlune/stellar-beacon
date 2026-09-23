package service

import (
	"strconv"
	"strings"
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type CollectionReactionService interface {
	ToggleCollectionReaction(c *gin.Context) model.ResultVO
	GetCollectionReactionState(c *gin.Context) model.ResultVO
	ListMyCollectionReactions(c *gin.Context) model.ResultVO
}

type MyCollectionReactionService struct {
	repo        port.CollectionReactionRepository
	collections port.CollectionPublicBatchReader
	limiter     port.RateLimiter
}

func NewCollectionReactionService(deps CollectionReactionServiceDeps) (*MyCollectionReactionService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyCollectionReactionService{repo: deps.Repo, collections: deps.Collections, limiter: deps.Limiter}, nil
}

func (s *MyCollectionReactionService) ToggleCollectionReaction(c *gin.Context) model.ResultVO {
	var vo model.CollectionReactionToggleVO
	if err := c.ShouldBind(&vo); err != nil || vo.CollectionId <= 0 {
		return model.ResultFromError(apperrors.Invalid("collection_reaction.toggle", "invalid collection"))
	}
	if strings.TrimSpace(vo.Reaction) == "" {
		vo.Reaction = port.ReactionLike
	}
	if !port.IsReactionKind(vo.Reaction) {
		return model.ResultFromError(apperrors.Invalid("collection_reaction.toggle", "unsupported reaction"))
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
	active, counts, err := s.repo.Set(c.Request.Context(), vo.CollectionId, userInfoID, vo.Reaction, vo.Active)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.CollectionReactionToggleDTO{
		Active: active, LikeCount: counts.LikeCount, FavoriteCount: counts.FavoriteCount,
	})
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
	state := states[collectionID]
	return model.ResultOkWithData(model.CollectionReactionStateDTO{
		CollectionId: collectionID, Like: state.Like, Favorite: state.Favorite,
	})
}

func (s *MyCollectionReactionService) ListMyCollectionReactions(c *gin.Context) model.ResultVO {
	if strings.TrimSpace(c.Query("reaction")) != port.ReactionFavorite {
		return model.ResultFromError(apperrors.Invalid("collection_reaction.list", "unsupported reaction"))
	}
	userInfoID, ok := currentUserInfoID(c)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "collection_reaction.user", nil))
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	ids, total, err := s.repo.ListFavoriteCollectionIDsByUser(c.Request.Context(), userInfoID, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	if len(ids) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: []*port.CollectionSummary{}, Count: total, Page: current, PageSize: size})
	}
	records, err := s.collections.ListPublicByIDs(c.Request.Context(), ids)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: orderCollectionSummaries(ids, records), Count: total, Page: current, PageSize: size})
}

func orderCollectionSummaries(ids []int, records []*port.CollectionSummary) []*port.CollectionSummary {
	byID := make(map[int]*port.CollectionSummary, len(records))
	for _, record := range records {
		byID[record.ID] = record
	}
	ordered := make([]*port.CollectionSummary, 0, len(records))
	for _, id := range ids {
		if record := byID[id]; record != nil {
			ordered = append(ordered, record)
		}
	}
	return ordered
}

var _ CollectionReactionService = (*MyCollectionReactionService)(nil)
