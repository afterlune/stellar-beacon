package port

import (
	"context"
	"time"
)

type CollectionSubscription struct {
	CollectionID int          `json:"collectionId"`
	Slug         string       `json:"slug"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Cover        string       `json:"cover,omitempty"`
	ArticleCount int          `json:"articleCount"`
	Owner        PublicAuthor `json:"owner"`
	Muted        bool         `json:"muted"`
	UnreadCount  int          `json:"unreadCount"`
	SubscribedAt time.Time    `json:"subscribedAt"`
}

type CollectionSubscriptionStatus struct {
	Subscribed bool `json:"subscribed"`
	Muted      bool `json:"muted"`
}

type CollectionFeedItem struct {
	EventId         int64        `json:"eventId"`
	CollectionID    int          `json:"collectionId"`
	Slug            string       `json:"slug"`
	CollectionTitle string       `json:"collectionTitle"`
	ArticleID       int          `json:"articleId"`
	ArticleTitle    string       `json:"articleTitle"`
	Excerpt         string       `json:"excerpt"`
	Cover           string       `json:"cover,omitempty"`
	Owner           PublicAuthor `json:"owner"`
	PublishedAt     time.Time    `json:"publishedAt"`
}

type CollectionSubscriptionRepository interface {
	Subscribe(ctx context.Context, userID, collectionID int) error
	Unsubscribe(ctx context.Context, userID, collectionID int) error
	SetMuted(ctx context.Context, userID, collectionID int, muted bool) error
	GetStatus(ctx context.Context, userID, collectionID int) (CollectionSubscriptionStatus, error)
	ListSubscriptions(ctx context.Context, userID, current, size int) ([]CollectionSubscription, int, error)
	ListFeed(ctx context.Context, userID, current, size int) ([]CollectionFeedItem, int, error)
}
