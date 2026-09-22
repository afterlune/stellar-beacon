package port

import (
	"context"
	"time"
)

const (
	RecommendationTargetArticle = "article"
	RecommendationTargetAuthor  = "author"
	RecommendationTargetTopic   = "topic"
)

const (
	RecommendationReasonSubscribedTopic = "subscribed_topic"
	RecommendationReasonFavorite        = "favorite_similar"
	RecommendationReasonLike            = "like_similar"
	RecommendationReasonReading         = "reading_similar"
	RecommendationReasonFollowedTopic   = "followed_author_topic"
	RecommendationReasonTrending        = "trending"
	RecommendationReasonLatest          = "latest"
)

func ValidRecommendationTarget(value string) bool {
	return value == RecommendationTargetArticle || value == RecommendationTargetAuthor || value == RecommendationTargetTopic
}

// RecommendationCursor is an opaque position in deterministic recommendation ordering.
type RecommendationCursor struct {
	Version     int       `json:"v"`
	Snapshot    time.Time `json:"snapshot"`
	Window      int       `json:"window"`
	Score       int       `json:"score"`
	PublishedAt time.Time `json:"publishedAt"`
	ArticleID   int       `json:"articleId"`
}

type RecommendationRequest struct {
	UserID         int
	SeedArticleIDs []int
	Size           int
	Snapshot       time.Time
	Cursor         *RecommendationCursor
}

type RecommendationReason struct {
	Type      string `json:"type"`
	Label     string `json:"label"`
	TopicType string `json:"topicType,omitempty"`
	TopicKey  string `json:"topicKey,omitempty"`
}

type RecommendationItem struct {
	ArticleCard
	Reason RecommendationReason `json:"reason"`
}

type RecommendationPage struct {
	Items        []RecommendationItem  `json:"items"`
	NextCursor   *RecommendationCursor `json:"nextCursor,omitempty"`
	HasMore      bool                  `json:"hasMore"`
	Personalized bool                  `json:"personalized"`
}

type RecommendationFeedback struct {
	ID         int       `json:"id"`
	TargetType string    `json:"targetType"`
	TargetKey  string    `json:"targetKey"`
	ArticleID  int       `json:"articleId,omitempty"`
	AuthorID   int       `json:"authorId,omitempty"`
	TopicType  string    `json:"topicType,omitempty"`
	TopicKey   string    `json:"topicKey,omitempty"`
	Label      string    `json:"label"`
	CreatedAt  time.Time `json:"createdAt"`
}

type RecommendationFeedbackInput struct {
	TargetType string
	ArticleID  int
	AuthorID   int
	TopicType  string
	TopicKey   string
}

type RecommendationRepository interface {
	ListRecommendations(ctx context.Context, request RecommendationRequest) (RecommendationPage, error)
	ListFeedback(ctx context.Context, userID, current, size int) ([]RecommendationFeedback, int, error)
	UpsertFeedback(ctx context.Context, userID int, input RecommendationFeedbackInput) (RecommendationFeedback, error)
	DeleteFeedback(ctx context.Context, userID, feedbackID int) error
}
