package task

import (
	"context"
	"errors"
	"log/slog"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const (
	defaultContentProjectionPollInterval = 5 * time.Second
	defaultContentProjectionBatchSize    = 100
)

type ContentProjectionBatchProcessor interface {
	ProcessPending(context.Context, []port.ContentProjection) (int, error)
}

type ContentProjectionWorkerDeps struct {
	Projections  port.ContentProjectionRepository
	Processor    ContentProjectionBatchProcessor
	PollInterval time.Duration
	BatchSize    int
}

type ContentProjectionWorker struct {
	projections port.ContentProjectionRepository
	processor   ContentProjectionBatchProcessor
	pollEvery   time.Duration
	batchSize   int
}

var _ Worker = (*ContentProjectionWorker)(nil).Run

func NewContentProjectionWorker(deps ContentProjectionWorkerDeps) (*ContentProjectionWorker, error) {
	if deps.Projections == nil {
		return nil, apperrors.Invalid("task.content_projection_worker.dependencies", "content projection repository is required")
	}
	if deps.Processor == nil {
		return nil, apperrors.Invalid("task.content_projection_worker.dependencies", "content projection processor is required")
	}
	if deps.PollInterval <= 0 {
		deps.PollInterval = defaultContentProjectionPollInterval
	}
	if deps.BatchSize <= 0 || deps.BatchSize > 100 {
		deps.BatchSize = defaultContentProjectionBatchSize
	}
	return &ContentProjectionWorker{
		projections: deps.Projections,
		processor:   deps.Processor,
		pollEvery:   deps.PollInterval,
		batchSize:   deps.BatchSize,
	}, nil
}

func (w *ContentProjectionWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("content projection worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(w.pollEvery)
	defer ticker.Stop()
	for {
		processed, err := w.processOne(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			slog.Error("content projection worker iteration failed", "error", safeWorkerError(err))
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

func (w *ContentProjectionWorker) processOne(ctx context.Context) (bool, error) {
	projections, err := w.projections.List(ctx, port.ContentProjectionFilter{
		Status:  port.ContentProjectionPending,
		Current: 1,
		Size:    w.batchSize,
	})
	if err != nil {
		return false, err
	}
	if len(projections) == 0 {
		return false, nil
	}
	_, err = w.processor.ProcessPending(ctx, projections)
	return true, err
}
