package port

import (
	"context"
	"time"
)

const (
	FollowContentArticle    = "article"
	FollowContentTalk       = "talk"
	FollowContentCollection = "collection"
)

type FollowUser struct {
	Id         int       `json:"id"`
	Handle     string    `json:"handle"`
	Nickname   string    `json:"nickname"`
	Avatar     string    `json:"avatar"`
	Intro      string    `json:"intro"`
	FollowedAt time.Time `json:"followedAt,omitempty"`
}

type FollowFeedItem struct {
	EventId     int64        `json:"eventId"`
	ContentType string       `json:"contentType"`
	ContentId   int          `json:"contentId"`
	Author      PublicAuthor `json:"author"`
	Title       string       `json:"title"`
	Excerpt     string       `json:"excerpt"`
	Cover       string       `json:"cover,omitempty"`
	Images      []string     `json:"images,omitempty"`
	PublishedAt time.Time    `json:"publishedAt"`
}

type FollowRepository interface {
	Follow(ctx context.Context, followerID, authorID int) error
	Unfollow(ctx context.Context, followerID, authorID int) error
	ListFollowing(ctx context.Context, userID, current, size int) ([]FollowUser, int, error)
	ListFollowers(ctx context.Context, userID, current, size int) ([]FollowUser, int, error)
	ListFollowFeed(ctx context.Context, userID int, contentType string, current, size int) ([]FollowFeedItem, int, error)
	ListNotifications(ctx context.Context, userID int, group string, current, size int) (NotificationPage, error)
	UnreadNotificationCount(ctx context.Context, userID int) (int, error)
	MarkNotificationsRead(ctx context.Context, userID int, cursor NotificationCursor) error
}
