package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type CollectionReactionService interface {
	ToggleCollectionReaction(context.Context, int, int, string, bool) (port.CollectionReactionResult, error)
	GetCollectionReactionState(context.Context, int, int) (port.CollectionReactionState, error)
	ListMyCollectionReactions(context.Context, int, int, int, string) ([]*port.CollectionSummary, int, error)
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

func (s *MyCollectionReactionService) ToggleCollectionReaction(ctx context.Context, userInfoID, collectionID int, reaction string, desired bool) (port.CollectionReactionResult, error) {
	if collectionID <= 0 {
		return port.CollectionReactionResult{}, apperrors.Invalid("collection_reaction.toggle", "invalid collection")
	}
	if strings.TrimSpace(reaction) == "" {
		reaction = port.ReactionLike
	}
	if !port.IsReactionKind(reaction) {
		return port.CollectionReactionResult{}, apperrors.Invalid("collection_reaction.toggle", "unsupported reaction")
	}
	if userInfoID <= 0 {
		return port.CollectionReactionResult{}, apperrors.New(apperrors.KindUnauthorized, "collection_reaction.user", nil)
	}
	allowed, err := allowRateLimit(ctx, s.limiter, "collection-reaction:", strconv.Itoa(userInfoID), 30, time.Minute)
	if err != nil {
		return port.CollectionReactionResult{}, apperrors.Unavailable("collection_reaction.rate_limit", err)
	}
	if !allowed {
		return port.CollectionReactionResult{}, apperrors.Invalid("collection_reaction.rate_limit", "too many reactions")
	}
	active, counts, err := s.repo.Set(ctx, collectionID, userInfoID, reaction, desired)
	if err != nil {
		return port.CollectionReactionResult{}, err
	}
	return port.CollectionReactionResult{Active: active, LikeCount: counts.LikeCount, FavoriteCount: counts.FavoriteCount}, nil
}

func (s *MyCollectionReactionService) GetCollectionReactionState(ctx context.Context, userInfoID, collectionID int) (port.CollectionReactionState, error) {
	if collectionID <= 0 {
		return port.CollectionReactionState{}, apperrors.Invalid("collection_reaction.state", "invalid collection")
	}
	if userInfoID <= 0 {
		return port.CollectionReactionState{}, apperrors.New(apperrors.KindUnauthorized, "collection_reaction.user", nil)
	}
	states, err := s.repo.States(ctx, userInfoID, []int{collectionID})
	if err != nil {
		return port.CollectionReactionState{}, err
	}
	return states[collectionID], nil
}

func (s *MyCollectionReactionService) ListMyCollectionReactions(ctx context.Context, userInfoID, current, size int, reaction string) ([]*port.CollectionSummary, int, error) {
	if strings.TrimSpace(reaction) != port.ReactionFavorite {
		return nil, 0, apperrors.Invalid("collection_reaction.list", "unsupported reaction")
	}
	if userInfoID <= 0 {
		return nil, 0, apperrors.New(apperrors.KindUnauthorized, "collection_reaction.user", nil)
	}
	ids, total, err := s.repo.ListFavoriteCollectionIDsByUser(ctx, userInfoID, current, size)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []*port.CollectionSummary{}, total, nil
	}
	records, err := s.collections.ListPublicByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	return orderCollectionSummaries(ids, records), total, nil
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
