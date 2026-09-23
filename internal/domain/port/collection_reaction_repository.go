package port

import "context"

type CollectionReactionCounts struct {
	LikeCount     int `json:"likeCount"`
	FavoriteCount int `json:"favoriteCount"`
}

type CollectionReactionState struct {
	Like     bool `json:"like"`
	Favorite bool `json:"favorite"`
}

// CollectionReactionRepository stores reader reactions to public or unlisted
// reading lists. The explicit desired state keeps writes idempotent and
// retry-safe.
type CollectionReactionRepository interface {
	Set(ctx context.Context, collectionID, userInfoID int, reaction string, active bool) (bool, CollectionReactionCounts, error)
	Counts(ctx context.Context, collectionIDs []int) (map[int]CollectionReactionCounts, error)
	States(ctx context.Context, userInfoID int, collectionIDs []int) (map[int]CollectionReactionState, error)
	ListFavoriteCollectionIDsByUser(ctx context.Context, userInfoID, current, size int) ([]int, int, error)
}
