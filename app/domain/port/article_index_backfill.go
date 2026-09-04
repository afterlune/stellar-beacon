package port

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ArticleIndexBackfillStatus is the durable lifecycle of one explicit full
// index rebuild. A failed run keeps its cursor and may be resumed after the
// operator has inspected the error.
type ArticleIndexBackfillStatus string

const (
	ArticleIndexBackfillPending   ArticleIndexBackfillStatus = "pending"
	ArticleIndexBackfillRunning   ArticleIndexBackfillStatus = "running"
	ArticleIndexBackfillPaused    ArticleIndexBackfillStatus = "paused"
	ArticleIndexBackfillCompleted ArticleIndexBackfillStatus = "completed"
	ArticleIndexBackfillFailed    ArticleIndexBackfillStatus = "failed"
)

type ArticleIndexBackfillState struct {
	ID                 string
	IndexUID           string
	IndexVersion       string
	Provider           string
	Model              string
	ModelVersion       string
	Dimension          int
	EmbeddingBatchSize int
	PageSize           int
	Status             ArticleIndexBackfillStatus
	Cursor             int
	ProcessedArticles  int64
	IndexedChunks      int64
	PauseRequested     bool
	LeaseOwner         string
	LeaseUntil         time.Time
	LastError          string
	StartedAt          time.Time
	CompletedAt        time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (s ArticleIndexBackfillState) Validate() error {
	switch {
	case strings.TrimSpace(s.ID) == "":
		return errors.New("backfill id is required")
	case strings.TrimSpace(s.IndexUID) == "":
		return errors.New("backfill index UID is required")
	case strings.TrimSpace(s.IndexVersion) == "":
		return errors.New("backfill index version is required")
	case strings.TrimSpace(s.Provider) == "":
		return errors.New("backfill embedding provider is required")
	case strings.TrimSpace(s.Model) == "":
		return errors.New("backfill embedding model is required")
	case strings.TrimSpace(s.ModelVersion) == "":
		return errors.New("backfill embedding model version is required")
	case s.Dimension <= 0:
		return errors.New("backfill embedding dimension must be positive")
	case s.EmbeddingBatchSize <= 0:
		return errors.New("backfill embedding batch size must be positive")
	case s.PageSize <= 0:
		return errors.New("backfill page size must be positive")
	case s.Cursor < 0:
		return errors.New("backfill cursor cannot be negative")
	case s.ProcessedArticles < 0:
		return errors.New("backfill processed article count cannot be negative")
	case s.IndexedChunks < 0:
		return errors.New("backfill indexed chunk count cannot be negative")
	case !validArticleIndexBackfillStatus(s.Status):
		return errors.New("backfill status is invalid")
	default:
		return nil
	}
}

func validArticleIndexBackfillStatus(status ArticleIndexBackfillStatus) bool {
	switch status {
	case ArticleIndexBackfillPending,
		ArticleIndexBackfillRunning,
		ArticleIndexBackfillPaused,
		ArticleIndexBackfillCompleted,
		ArticleIndexBackfillFailed:
		return true
	default:
		return false
	}
}

// ArticleIndexBackfillRepository persists run state independently from the
// article index itself. Claim and checkpoint operations must enforce the
// current lease owner so two CLI processes cannot advance one cursor.
type ArticleIndexBackfillRepository interface {
	CreateOrGet(context.Context, ArticleIndexBackfillState) (ArticleIndexBackfillState, error)
	Get(context.Context, string) (ArticleIndexBackfillState, error)
	Claim(context.Context, string, string, time.Time, time.Duration) (ArticleIndexBackfillState, bool, error)
	RequestPause(context.Context, string) error
	Resume(context.Context, string) error
	Checkpoint(context.Context, string, string, int, int64, int64, time.Time, time.Duration) error
	Pause(context.Context, string, string, time.Time) error
	Complete(context.Context, string, string, time.Time) error
	Fail(context.Context, string, string, string, time.Time) error
}
