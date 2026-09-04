package port

import (
	"encoding/json"
	"errors"
	"time"
)

const AIJobKindArticleIndex = "article.index_sync"

type ArticleIndexAction string

const (
	ArticleIndexUpsert ArticleIndexAction = "upsert"
	ArticleIndexDelete ArticleIndexAction = "delete"
)

type ArticleLifecycleEvent string

const (
	ArticlePublished ArticleLifecycleEvent = "article.published"
	ArticleUpdated   ArticleLifecycleEvent = "article.updated"
	ArticlePrivate   ArticleLifecycleEvent = "article.private"
	ArticleDeleted   ArticleLifecycleEvent = "article.deleted"
	ArticleRestored  ArticleLifecycleEvent = "article.restored"
)

// ArticleIndexJobPayload is intentionally metadata-only. The worker reads the
// current article by ID, so a retry cannot index stale content from an old
// request and a queue row never duplicates the article body.
type ArticleIndexJobPayload struct {
	SchemaVersion int                   `json:"schemaVersion"`
	ArticleID     int                   `json:"articleId"`
	Action        ArticleIndexAction    `json:"action"`
	Event         ArticleLifecycleEvent `json:"event"`
	Status        int                   `json:"status"`
	IsDelete      int                   `json:"isDelete"`
	OccurredAt    time.Time             `json:"occurredAt"`
}

func (p ArticleIndexJobPayload) Validate() error {
	switch {
	case p.SchemaVersion <= 0:
		return errors.New("article index job schema version must be positive")
	case p.ArticleID <= 0:
		return errors.New("article index job article id must be positive")
	case p.Action != ArticleIndexUpsert && p.Action != ArticleIndexDelete:
		return errors.New("article index job action is invalid")
	case !validArticleLifecycleEvent(p.Event):
		return errors.New("article index job event is invalid")
	case p.IsDelete != 0 && p.IsDelete != 1:
		return errors.New("article index job delete flag is invalid")
	case p.OccurredAt.IsZero():
		return errors.New("article index job occurrence time is required")
	default:
		return nil
	}
}

func validArticleLifecycleEvent(event ArticleLifecycleEvent) bool {
	switch event {
	case ArticlePublished, ArticleUpdated, ArticlePrivate, ArticleDeleted, ArticleRestored:
		return true
	default:
		return false
	}
}

func NewArticleIndexJobPayload(articleID int, action ArticleIndexAction, event ArticleLifecycleEvent, status, isDelete int, occurredAt time.Time) ([]byte, error) {
	payload := ArticleIndexJobPayload{
		SchemaVersion: 1,
		ArticleID:     articleID,
		Action:        action,
		Event:         event,
		Status:        status,
		IsDelete:      isDelete,
		OccurredAt:    occurredAt.UTC(),
	}
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(payload)
}
