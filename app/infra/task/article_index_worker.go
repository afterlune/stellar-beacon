package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/search"

	"github.com/google/uuid"
)

const (
	defaultArticleIndexPollInterval = time.Second
	defaultArticleIndexLease        = 5 * time.Minute
	defaultArticleIndexRetryBase    = 2 * time.Second
	maxArticleIndexRetryDelay       = 5 * time.Minute
)

// ArticleIndexWorkerDeps is the explicit composition contract for the
// article-index projection. The worker does not create an index; provisioning
// remains an explicit operator action, which keeps existing Meilisearch
// containers and indexes untouched at application startup.
type ArticleIndexWorkerDeps struct {
	Jobs          port.AIJobRepository
	Articles      port.ArticleRepository
	Router        port.ModelRouter
	Index         port.ArticleChunkIndex
	Spec          search.ArticleChunksIndexSpec
	Chunker       search.MarkdownChunker
	Projections   port.ContentProjectionRepository
	WorkerID      string
	PollInterval  time.Duration
	LeaseDuration time.Duration
	RetryBase     time.Duration
}

// ArticleIndexWorker consumes durable article.index_sync jobs one at a time.
// It owns no goroutines: Run is the goroutine body registered with Supervisor.
type ArticleIndexWorker struct {
	jobs          port.AIJobRepository
	articles      port.ArticleRepository
	projector     *ArticleIndexProjector
	spec          search.ArticleChunksIndexSpec
	workerID      string
	pollInterval  time.Duration
	leaseDuration time.Duration
	retryBase     time.Duration
	now           func() time.Time
}

var _ Worker = (*ArticleIndexWorker)(nil).Run

func NewArticleIndexWorker(deps ArticleIndexWorkerDeps) (*ArticleIndexWorker, error) {
	if deps.Jobs == nil {
		return nil, apperrors.Invalid("task.article_index_worker.dependencies", "AI job repository is required")
	}
	if deps.Articles == nil {
		return nil, apperrors.Invalid("task.article_index_worker.dependencies", "article repository is required")
	}
	if deps.Router == nil {
		return nil, apperrors.Invalid("task.article_index_worker.dependencies", "model router is required")
	}
	if deps.Index == nil {
		return nil, apperrors.Invalid("task.article_index_worker.dependencies", "article chunk index is required")
	}
	if err := deps.Spec.Validate(); err != nil {
		return nil, err
	}
	projector, err := NewArticleIndexProjector(ArticleIndexProjectorDeps{
		Router:      deps.Router,
		Index:       deps.Index,
		Spec:        deps.Spec,
		Chunker:     deps.Chunker,
		Projections: deps.Projections,
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(deps.WorkerID) == "" {
		deps.WorkerID = "article-index-" + uuid.NewString()
	}
	if deps.PollInterval <= 0 {
		deps.PollInterval = defaultArticleIndexPollInterval
	}
	if deps.LeaseDuration <= 0 {
		deps.LeaseDuration = defaultArticleIndexLease
	}
	if deps.RetryBase <= 0 {
		deps.RetryBase = defaultArticleIndexRetryBase
	}
	return &ArticleIndexWorker{
		jobs:          deps.Jobs,
		articles:      deps.Articles,
		projector:     projector,
		spec:          deps.Spec,
		workerID:      strings.TrimSpace(deps.WorkerID),
		pollInterval:  deps.PollInterval,
		leaseDuration: deps.LeaseDuration,
		retryBase:     deps.RetryBase,
		now:           func() time.Time { return time.Now().UTC() },
	}, nil
}

func (w *ArticleIndexWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("article index worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		processed, err := w.processOne(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			slog.Error("article index worker iteration failed", "worker", w.workerID, "error", safeWorkerError(err))
		}
		if processed {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// RunOnce processes at most one already-enqueued article index job. It does
// not scan, schedule, or start a background loop.
func (w *ArticleIndexWorker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil {
		return false, errors.New("article index worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return w.processOne(ctx)
}

func (w *ArticleIndexWorker) processOne(ctx context.Context) (bool, error) {
	claimed, ok, err := port.ClaimAIJob(ctx, w.jobs, w.workerID, port.AIJobKindArticleIndex, w.currentTime(), w.leaseDuration)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if claimed.Kind != port.AIJobKindArticleIndex {
		return true, w.deadLetter(ctx, claimed, fmt.Errorf("unsupported job kind %q", claimed.Kind))
	}
	if err := w.handleJob(ctx, claimed); err != nil {
		// Only the parent context controls worker shutdown. A provider request
		// can have its own timeout; that is a transient job error and must go
		// through retry/dead-letter handling instead of abandoning the lease.
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		if settleErr := w.failJob(ctx, claimed, err); settleErr != nil {
			return true, fmt.Errorf("handle article index job: %w; settle failure: %v", err, settleErr)
		}
		slog.Warn("article index job scheduled for retry or dead letter", "worker", w.workerID, "job_id", claimed.ID, "attempt", claimed.Attempts, "error", safeWorkerError(err))
		return true, nil
	}
	resultPayload, err := json.Marshal(map[string]any{
		"articleId":    payloadArticleID(claimed.Payload),
		"indexedAt":    w.currentTime(),
		"workerId":     w.workerID,
		"index":        w.spec.UID,
		"model":        w.spec.Model,
		"modelVersion": w.spec.ModelVersion,
	})
	if err != nil {
		return true, w.failJob(ctx, claimed, err)
	}
	if err := w.jobs.Complete(ctx, claimed.ID, w.workerID, port.AIJobResult{
		RunID:   uuid.NewString(),
		Payload: resultPayload,
	}); err != nil {
		return true, err
	}
	return true, nil
}

func (w *ArticleIndexWorker) handleJob(ctx context.Context, job port.AIJob) error {
	var payload port.ArticleIndexJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode article index job payload: %w", err)
	}
	if err := payload.Validate(); err != nil {
		return fmt.Errorf("validate article index job payload: %w", err)
	}
	switch payload.Action {
	case port.ArticleIndexDelete:
		return w.projector.DeleteArticle(ctx, payload.ArticleID)
	case port.ArticleIndexUpsert:
		return w.upsertArticle(ctx, payload.ArticleID)
	default:
		return fmt.Errorf("unsupported article index action %q", payload.Action)
	}
}

func (w *ArticleIndexWorker) upsertArticle(ctx context.Context, articleID int) error {
	articleRecord, category, tags, err := w.articles.GetAdminArticle(ctx, articleID)
	if err != nil {
		if apperrors.KindOf(err) == apperrors.KindNotFound {
			// An older upsert can race with a hard delete. Treat the current
			// absence as the desired deleted projection instead of retaining
			// stale chunks.
			return w.projector.DeleteArticle(ctx, articleID)
		}
		return fmt.Errorf("read article %d: %w", articleID, err)
	}
	if !port.IsPublicArticle(articleRecord.Status, articleRecord.IsDelete) {
		return w.projector.DeleteArticle(ctx, articleID)
	}
	_, err = w.projector.UpsertSource(ctx, port.ArticleIndexSource{
		Article:      articleRecord,
		CategoryName: category,
		Tags:         tags,
	})
	return err
}

func (w *ArticleIndexWorker) failJob(ctx context.Context, job port.AIJob, err error) error {
	if job.Attempts >= job.MaxAttempts {
		return w.deadLetter(ctx, job, err)
	}
	return w.jobs.Retry(ctx, job.ID, w.workerID, w.currentTime().Add(retryDelay(w.retryBase, job.Attempts)), safeWorkerError(err))
}

func (w *ArticleIndexWorker) deadLetter(ctx context.Context, job port.AIJob, err error) error {
	return w.jobs.DeadLetter(ctx, job.ID, w.workerID, safeWorkerError(err))
}

func retryDelay(base time.Duration, attempt int) time.Duration {
	if base <= 0 {
		base = defaultArticleIndexRetryBase
	}
	if attempt <= 1 {
		return base
	}
	delay := base
	for index := 1; index < attempt && delay < maxArticleIndexRetryDelay; index++ {
		if delay > maxArticleIndexRetryDelay/2 {
			return maxArticleIndexRetryDelay
		}
		delay *= 2
	}
	if delay > maxArticleIndexRetryDelay || delay < 0 {
		return maxArticleIndexRetryDelay
	}
	return delay
}

func (w *ArticleIndexWorker) currentTime() time.Time {
	if w == nil || w.now == nil {
		return time.Now().UTC()
	}
	return w.now().UTC()
}

func articleForIndex(article port.TArticle, category string) port.Article {
	return port.Article{
		Id:             article.Id,
		ArticleCover:   article.ArticleCover,
		ArticleTitle:   article.ArticleTitle,
		ArticleContent: article.ArticleContent,
		IsTop:          article.IsTop,
		IsFeatured:     article.IsFeatured,
		CategoryName:   category,
		Status:         article.Status,
		CreateTime:     article.CreateTime,
		UpdateTime:     article.UpdateTime,
		Type:           article.Type,
		OriginalUrl:    article.OriginalUrl,
		IsDelete:       article.IsDelete,
	}
}

func payloadArticleID(payload []byte) int {
	var decoded struct {
		ArticleID int `json:"articleId"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return 0
	}
	return decoded.ArticleID
}

func safeWorkerError(err error) string {
	return apperrors.SafeCode(err)
}
