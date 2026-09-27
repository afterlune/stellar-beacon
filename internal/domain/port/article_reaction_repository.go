package port

import "context"

// Reaction kinds supported by the reader-interaction feature. The service
// layer validates every incoming value against this list.
const (
	ReactionLike     = "like"
	ReactionFavorite = "favorite"
)

// ReactionKinds lists the accepted reaction names.
var ReactionKinds = []string{ReactionLike, ReactionFavorite}

// IsReactionKind reports whether value is a supported reaction name.
func IsReactionKind(value string) bool {
	for _, kind := range ReactionKinds {
		if kind == value {
			return true
		}
	}
	return false
}

// ReactionCounts is the aggregate published with article payloads. It carries
// no account information, so it can be cached and shared between readers.
type ReactionCounts struct {
	LikeCount     int `json:"likeCount"`
	FavoriteCount int `json:"favoriteCount"`
}

// ReactionToggleResult reports the requested state and current aggregate
// counts after an idempotent reaction update.
type ReactionToggleResult struct {
	Active        bool `json:"active"`
	LikeCount     int  `json:"likeCount"`
	FavoriteCount int  `json:"favoriteCount"`
}

// ArticleReactionState is one account's reaction state for an article.
type ArticleReactionState struct {
	ArticleId int  `json:"articleId"`
	Like      bool `json:"like"`
	Favorite  bool `json:"favorite"`
}

// ArticleReactionRepository stores reader reactions and answers the two read
// shapes the API needs: per-article totals and per-account state.
type ArticleReactionRepository interface {
	// Toggle adds the reaction when it is missing and removes it otherwise,
	// returning whether the reaction is active afterwards.
	Toggle(ctx context.Context, articleID, userInfoID int, reaction string) (bool, error)
	// Counts aggregates like/favorite totals for the given articles. Articles
	// without reactions are absent from the map.
	Counts(ctx context.Context, articleIDs []int) (map[int]ReactionCounts, error)
	// States reports the active reactions of one account for the given articles.
	States(ctx context.Context, userInfoID int, articleIDs []int) (map[int]map[string]bool, error)
	// ListArticleIDsByUser pages through one account's reactions, newest first.
	ListArticleIDsByUser(ctx context.Context, userInfoID int, reaction string, current, size int) ([]int, int64, error)
}
