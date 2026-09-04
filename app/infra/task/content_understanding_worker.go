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

	"github.com/google/uuid"
)

const (
	defaultContentUnderstandingPollInterval = time.Second
	defaultContentUnderstandingLease        = 5 * time.Minute
	defaultContentUnderstandingRetryBase    = 2 * time.Second
)

type ContentUnderstandingWorkerDeps struct {
	Jobs          port.AIJobRepository
	Articles      port.ArticleRepository
	Analyzer      port.ContentUnderstandingGateway
	WorkerID      string
	PollInterval  time.Duration
	LeaseDuration time.Duration
	RetryBase     time.Duration
}

type ContentUnderstandingWorker struct {
	jobs          port.AIJobRepository
	articles      port.ArticleRepository
	analyzer      port.ContentUnderstandingGateway
	workerID      string
	pollInterval  time.Duration
	leaseDuration time.Duration
	retryBase     time.Duration
	now           func() time.Time
}

var _ Worker = (*ContentUnderstandingWorker)(nil).Run

func NewContentUnderstandingWorker(deps ContentUnderstandingWorkerDeps) (*ContentUnderstandingWorker, error) {
	if deps.Jobs == nil {
		return nil, apperrors.Invalid("task.content_understanding_worker.dependencies", "AI job repository is required")
	}
	if deps.Articles == nil {
		return nil, apperrors.Invalid("task.content_understanding_worker.dependencies", "article repository is required")
	}
	if deps.Analyzer == nil {
		return nil, apperrors.Invalid("task.content_understanding_worker.dependencies", "content analyzer is required")
	}
	if strings.TrimSpace(deps.WorkerID) == "" {
		deps.WorkerID = "content-understanding-" + uuid.NewString()
	}
	if deps.PollInterval <= 0 {
		deps.PollInterval = defaultContentUnderstandingPollInterval
	}
	if deps.LeaseDuration <= 0 {
		deps.LeaseDuration = defaultContentUnderstandingLease
	}
	if deps.RetryBase <= 0 {
		deps.RetryBase = defaultContentUnderstandingRetryBase
	}
	return &ContentUnderstandingWorker{
		jobs:          deps.Jobs,
		articles:      deps.Articles,
		analyzer:      deps.Analyzer,
		workerID:      strings.TrimSpace(deps.WorkerID),
		pollInterval:  deps.PollInterval,
		leaseDuration: deps.LeaseDuration,
		retryBase:     deps.RetryBase,
		now:           func() time.Time { return time.Now().UTC() },
	}, nil
}

func (w *ContentUnderstandingWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("content understanding worker is nil")
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
			slog.Error("content understanding worker iteration failed", "worker", w.workerID, "error", safeWorkerError(err))
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

// RunOnce processes at most one already-enqueued content understanding job.
func (w *ContentUnderstandingWorker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil {
		return false, errors.New("content understanding worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return w.processOne(ctx)
}

func (w *ContentUnderstandingWorker) processOne(ctx context.Context) (bool, error) {
	claimed, ok, err := port.ClaimAIJob(ctx, w.jobs, w.workerID, port.AIJobKindContentUnderstanding, w.currentTime(), w.leaseDuration)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if claimed.Kind != port.AIJobKindContentUnderstanding {
		return true, w.deadLetter(ctx, claimed, fmt.Errorf("unsupported job kind %q", claimed.Kind))
	}
	resultPayload, runID, err := w.handleJob(ctx, claimed)
	if err != nil {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		if settleErr := w.failJob(ctx, claimed, err); settleErr != nil {
			return true, fmt.Errorf("handle content understanding job: %w; settle failure: %v", err, settleErr)
		}
		slog.Warn("content understanding job scheduled for retry or dead letter", "worker", w.workerID, "job_id", claimed.ID, "attempt", claimed.Attempts, "error", safeWorkerError(err))
		return true, nil
	}
	if err := w.jobs.Complete(ctx, claimed.ID, w.workerID, port.AIJobResult{RunID: runID, Payload: resultPayload}); err != nil {
		return true, err
	}
	return true, nil
}

type contentUnderstandingJobResult struct {
	SchemaVersion int      `json:"schemaVersion"`
	ArticleID     int      `json:"articleId"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary,omitempty"`
	Category      string   `json:"category,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

func (w *ContentUnderstandingWorker) handleJob(ctx context.Context, job port.AIJob) ([]byte, string, error) {
	var payload port.ContentUnderstandingJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, "", fmt.Errorf("decode content understanding job payload: %w", err)
	}
	if err := payload.Validate(); err != nil {
		return nil, "", fmt.Errorf("validate content understanding job payload: %w", err)
	}
	article, _, _, err := w.articles.GetAdminArticle(ctx, payload.ArticleID)
	if err != nil {
		if apperrors.KindOf(err) == apperrors.KindNotFound {
			return marshalContentUnderstandingResult(contentUnderstandingJobResult{
				SchemaVersion: 1,
				ArticleID:     payload.ArticleID,
				Status:        "skipped_article_not_found",
			})
		}
		return nil, "", fmt.Errorf("read article %d: %w", payload.ArticleID, err)
	}
	if !port.IsPublicArticle(article.Status, article.IsDelete) {
		return marshalContentUnderstandingResult(contentUnderstandingJobResult{
			SchemaVersion: 1,
			ArticleID:     payload.ArticleID,
			Status:        "skipped_not_public",
		})
	}
	result, err := w.analyzer.Analyze(ctx, port.ContentUnderstandingRequest{
		ArticleID: payload.ArticleID,
		Title:     article.ArticleTitle,
		Content:   article.ArticleContent,
	})
	if err != nil {
		return nil, "", fmt.Errorf("analyze article %d: %w", payload.ArticleID, err)
	}
	if result.ArticleID != 0 && result.ArticleID != payload.ArticleID {
		return nil, "", errors.New("content analyzer returned a mismatched article ID")
	}
	result.ArticleID = payload.ArticleID
	result.Summary = strings.TrimSpace(result.Summary)
	result.Category = strings.TrimSpace(result.Category)
	if result.Summary == "" || result.Category == "" {
		return nil, "", errors.New("content analyzer returned incomplete result")
	}
	resultPayload, runID, err := marshalContentUnderstandingResult(contentUnderstandingJobResult{
		SchemaVersion: 1,
		ArticleID:     result.ArticleID,
		Status:        "ready",
		Summary:       result.Summary,
		Category:      result.Category,
		Tags:          append([]string(nil), result.Tags...),
	})
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(result.RunID) != "" {
		runID = strings.TrimSpace(result.RunID)
	}
	return resultPayload, runID, nil
}

func marshalContentUnderstandingResult(result contentUnderstandingJobResult) ([]byte, string, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, "", fmt.Errorf("encode content understanding result: %w", err)
	}
	return payload, uuid.NewString(), nil
}

func (w *ContentUnderstandingWorker) failJob(ctx context.Context, job port.AIJob, err error) error {
	if job.Attempts >= job.MaxAttempts {
		return w.deadLetter(ctx, job, err)
	}
	return w.jobs.Retry(ctx, job.ID, w.workerID, w.currentTime().Add(retryDelay(w.retryBase, job.Attempts)), safeWorkerError(err))
}

func (w *ContentUnderstandingWorker) deadLetter(ctx context.Context, job port.AIJob, err error) error {
	return w.jobs.DeadLetter(ctx, job.ID, w.workerID, safeWorkerError(err))
}

func (w *ContentUnderstandingWorker) currentTime() time.Time {
	if w == nil || w.now == nil {
		return time.Now().UTC()
	}
	return w.now().UTC()
}
