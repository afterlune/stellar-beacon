package port

import (
	"context"
	"time"
)

const (
	CollectionVisibilityPrivate  = "private"
	CollectionVisibilityUnlisted = "unlisted"
	CollectionVisibilityPublic   = "public"
)

const (
	CollectionSortLatest = "latest"
	CollectionSortHot    = "hot"
)

func ValidCollectionVisibility(value string) bool {
	return value == CollectionVisibilityPrivate || value == CollectionVisibilityUnlisted || value == CollectionVisibilityPublic
}

type CollectionSummary struct {
	ID               int           `json:"id,omitempty"`
	Slug             string        `json:"slug"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	Visibility       string        `json:"visibility"`
	Owner            *PublicAuthor `json:"owner,omitempty"`
	Cover            string        `json:"cover,omitempty"`
	ArticleCount     int           `json:"articleCount"`
	LikeCount        int           `json:"likeCount"`
	CommentCount     int           `json:"commentCount"`
	HotScore         int           `json:"hotScore,omitempty"`
	ModerationStatus string        `json:"moderationStatus,omitempty"`
	ModerationReason string        `json:"moderationReason,omitempty"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
}

type CollectionItemRecord struct {
	ArticleID int    `json:"articleId"`
	Note      string `json:"note"`
	Position  int    `json:"position"`
	Available bool   `json:"available"`
}

type CollectionItemView struct {
	ArticleID int          `json:"articleId"`
	Note      string       `json:"note"`
	Position  int          `json:"position"`
	Available bool         `json:"available"`
	Article   *ArticleCard `json:"article,omitempty"`
}

type CollectionDetail struct {
	Collection CollectionSummary    `json:"collection"`
	Items      []CollectionItemView `json:"items"`
}

type CollectionRecord struct {
	Collection CollectionSummary
	Items      []CollectionItemRecord
}

// CollectionPublicReader is the narrow read surface used by comments and
// moderation-aware flows that only need one visible collection.
type CollectionPublicReader interface {
	GetPublicByID(ctx context.Context, collectionID int) (CollectionSummary, error)
}

type CollectionSaveInput struct {
	Title       string
	Description string
	Visibility  string
}

type CollectionRepository interface {
	ListPublic(ctx context.Context, sort string, current, size int) ([]*CollectionSummary, int, error)
	ListPublicByOwner(ctx context.Context, userID, current, size int) ([]*CollectionSummary, int, error)
	GetPublicBySlug(ctx context.Context, slug string) (CollectionRecord, error)
	ListOwned(ctx context.Context, userID, current, size int) ([]*CollectionSummary, int, error)
	GetOwned(ctx context.Context, userID, collectionID int) (CollectionRecord, error)
	CreateOwned(ctx context.Context, userID int, slug string, input CollectionSaveInput) (CollectionSummary, error)
	UpdateOwned(ctx context.Context, userID, collectionID int, input CollectionSaveInput) (CollectionSummary, error)
	DeleteOwned(ctx context.Context, userID, collectionID int) error
	AddItem(ctx context.Context, userID, collectionID, articleID int, note string) error
	RemoveItem(ctx context.Context, userID, collectionID, articleID int) error
	Reorder(ctx context.Context, userID, collectionID int, articleIDs []int) error
	ListAdmin(ctx context.Context, current, size int, moderation, keywords string) ([]*CollectionSummary, int, error)
}
