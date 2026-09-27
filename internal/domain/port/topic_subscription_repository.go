package port

import (
	"context"
	"time"
)

// Topic subscription types. Only the two reader-facing taxonomy surfaces are
// subscribable: a series is a single author's work, so following the author
// already covers it.
const (
	TopicTypeCategory = "category"
	TopicTypeTag      = "tag"
)

// ValidTopicType reports whether value is a subscribable topic type.
func ValidTopicType(value string) bool {
	return value == TopicTypeCategory || value == TopicTypeTag
}

// TopicSubscription is one subscribed topic together with the live counters the
// management list renders. UnreadCount is zero for a muted subscription because
// the notification inbox suppresses muted topics as well.
type TopicSubscription struct {
	TopicType    string    `json:"topicType"`
	TopicKey     string    `json:"topicKey"`
	TopicName    string    `json:"topicName"`
	ArticleCount int       `json:"articleCount"`
	HotScore     int       `json:"hotScore"`
	Muted        bool      `json:"muted"`
	UnreadCount  int       `json:"unreadCount"`
	SubscribedAt time.Time `json:"subscribedAt"`
}

// TopicFeedItem is one published article that matched at least one of the
// reader's subscribed topics.
type TopicFeedItem struct {
	FollowFeedItem
	Topics []string `json:"topics"`
}

// TopicSubscriptionRepository owns the reader-facing topic subscription ledger
// and the derived topic feed. Subscriptions never fan out: notifications and the
// feed are resolved from the publish-event stream at read time.
type TopicSubscriptionRepository interface {
	Subscribe(ctx context.Context, userID int, topicType, topicKey string) error
	Unsubscribe(ctx context.Context, userID int, topicType, topicKey string) error
	SetMuted(ctx context.Context, userID int, topicType, topicKey string, muted bool) error
	ListSubscriptions(ctx context.Context, userID, current, size int) ([]TopicSubscription, int, error)
	ListTopicFeed(ctx context.Context, userID, current, size int) ([]TopicFeedItem, int, error)
}
