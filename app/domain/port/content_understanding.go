package port

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const AIJobKindContentUnderstanding = "article.content_understanding"

type ContentUnderstandingJobPayload struct {
	SchemaVersion int       `json:"schemaVersion"`
	ArticleID     int       `json:"articleId"`
	OccurredAt    time.Time `json:"occurredAt"`
}

func (p ContentUnderstandingJobPayload) Validate() error {
	if p.SchemaVersion <= 0 {
		return errors.New("content understanding job schema version must be positive")
	}
	if p.ArticleID <= 0 {
		return errors.New("content understanding job article id must be positive")
	}
	if p.OccurredAt.IsZero() {
		return errors.New("content understanding job occurrence time is required")
	}
	return nil
}

func NewContentUnderstandingJobPayload(articleID int, occurredAt time.Time) ([]byte, error) {
	payload := ContentUnderstandingJobPayload{
		SchemaVersion: 1,
		ArticleID:     articleID,
		OccurredAt:    occurredAt.UTC(),
	}
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(payload)
}

type ContentUnderstandingRequest struct {
	ArticleID int
	Title     string
	Content   string
}

type ContentUnderstanding struct {
	ArticleID int
	Summary   string
	Category  string
	Tags      []string
	RunID     string
}

type ContentUnderstandingGateway interface {
	Analyze(context.Context, ContentUnderstandingRequest) (ContentUnderstanding, error)
}
