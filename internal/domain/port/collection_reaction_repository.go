package port

import "context"

// CollectionReactionRepository stores reader reactions to public or unlisted
// reading lists. v3 exposes likes only; the explicit active state keeps writes
// idempotent and retry-safe.
type CollectionReactionRepository interface {
	Set(ctx context.Context, collectionID, userInfoID int, active bool) (bool, int, error)
	Counts(ctx context.Context, collectionIDs []int) (map[int]int, error)
	States(ctx context.Context, userInfoID int, collectionIDs []int) (map[int]bool, error)
}
