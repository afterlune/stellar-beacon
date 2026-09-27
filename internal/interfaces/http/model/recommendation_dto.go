package model

import "github.com/eternallyzzz/stellar-beacon/internal/domain/port"

type RecommendationQueryVO struct {
	Cursor         string `json:"cursor" form:"cursor"`
	Size           int    `json:"size" form:"size"`
	SeedArticleIDs []int  `json:"seedArticleIds" form:"seedArticleIds"`
}

type RecommendationFeedbackVO struct {
	TargetType string `json:"targetType" form:"targetType"`
	ArticleID  int    `json:"articleId" form:"articleId"`
	AuthorID   int    `json:"authorId" form:"authorId"`
	TopicType  string `json:"topicType" form:"topicType"`
	TopicKey   string `json:"topicKey" form:"topicKey"`
}

type RecommendationFeedDTO struct {
	Items        []port.RecommendationItem `json:"items"`
	NextCursor   string                    `json:"nextCursor,omitempty"`
	HasMore      bool                      `json:"hasMore"`
	Personalized bool                      `json:"personalized"`
}
