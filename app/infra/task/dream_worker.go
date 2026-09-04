package task

import (
	agentapp "benetnasch/app/application/agent"
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultDreamPollInterval = time.Second
	defaultDreamScanInterval = 15 * time.Minute
	defaultDreamLease        = 5 * time.Minute
	defaultDreamRetryBase    = 2 * time.Second
)

type dreamTaskScheduler interface {
	Scan(context.Context, int) (int, int, error)
}

type DreamWorkerDeps struct {
	Jobs          port.AIJobRepository
	Scheduler     dreamTaskScheduler
	Generator     port.DreamCandidateGenerator
	Safety        port.AgentSafetySwitch
	WorkerID      string
	PollInterval  time.Duration
	ScanInterval  time.Duration
	LeaseDuration time.Duration
	RetryBase     time.Duration
}

// DreamWorker turns durable dream tasks into pending AI reviews. It has no
// publication dependency; approval remains an explicit AI Studio action.
type DreamWorker struct {
	jobs          port.AIJobRepository
	scheduler     dreamTaskScheduler
	generator     port.DreamCandidateGenerator
	safety        port.AgentSafetySwitch
	workerID      string
	pollInterval  time.Duration
	scanInterval  time.Duration
	leaseDuration time.Duration
	retryBase     time.Duration
	now           func() time.Time
}

var _ Worker = (*DreamWorker)(nil).Run

func NewDreamWorker(deps DreamWorkerDeps) (*DreamWorker, error) {
	if deps.Jobs == nil || deps.Scheduler == nil || deps.Generator == nil {
		return nil, errors.New("dream worker requires jobs, scheduler and generator")
	}
	if strings.TrimSpace(deps.WorkerID) == "" {
		deps.WorkerID = "agent-dream-" + uuid.NewString()
	}
	if deps.PollInterval <= 0 {
		deps.PollInterval = defaultDreamPollInterval
	}
	if deps.ScanInterval <= 0 {
		deps.ScanInterval = defaultDreamScanInterval
	}
	if deps.LeaseDuration <= 0 {
		deps.LeaseDuration = defaultDreamLease
	}
	if deps.RetryBase <= 0 {
		deps.RetryBase = defaultDreamRetryBase
	}
	return &DreamWorker{
		jobs:          deps.Jobs,
		scheduler:     deps.Scheduler,
		generator:     deps.Generator,
		safety:        deps.Safety,
		workerID:      strings.TrimSpace(deps.WorkerID),
		pollInterval:  deps.PollInterval,
		scanInterval:  deps.ScanInterval,
		leaseDuration: deps.LeaseDuration,
		retryBase:     deps.RetryBase,
		now:           func() time.Time { return time.Now().UTC() },
	}, nil
}

func (w *DreamWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("dream worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	lastScan := time.Time{}
	cursor := 0
	for {
		if w.safety != nil {
			stopped, err := w.safety.IsStopped(ctx)
			if err != nil {
				slog.Error("dream safety switch read failed", "worker", w.workerID, "error", safeWorkerError(err))
				if !w.wait(ctx) {
					return ctx.Err()
				}
				continue
			}
			if stopped {
				if !w.wait(ctx) {
					return ctx.Err()
				}
				continue
			}
		}
		if lastScan.IsZero() || w.currentTime().Sub(lastScan) >= w.scanInterval {
			next, count, err := w.scheduler.Scan(ctx, cursor)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				slog.Error("dream task scan failed", "worker", w.workerID, "error", safeWorkerError(err))
			} else {
				cursor = next
				lastScan = w.currentTime()
				if count > 0 {
					slog.Info("dream tasks enqueued", "worker", w.workerID, "count", count)
				}
			}
		}
		processed, err := w.processOne(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			slog.Error("dream worker iteration failed", "worker", w.workerID, "error", safeWorkerError(err))
		}
		if processed {
			continue
		}
		if !w.wait(ctx) {
			return ctx.Err()
		}
	}
}

// RunOnce consumes at most one already-enqueued dream generation job. It
// does not perform a new article scan.
func (w *DreamWorker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil {
		return false, errors.New("dream worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := checkOneShotSafety(ctx, w.safety); err != nil {
		return false, err
	}
	return w.processOne(ctx)
}

func (w *DreamWorker) wait(ctx context.Context) bool {
	timer := time.NewTimer(w.pollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *DreamWorker) processOne(ctx context.Context) (bool, error) {
	claimed, ok, err := port.ClaimAIJob(ctx, w.jobs, w.workerID, port.AIJobKindDreamCandidate, w.currentTime(), w.leaseDuration)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	payload, err := agentapp.DecodeDreamTask(claimed.Payload)
	if err != nil {
		return true, w.deadLetter(ctx, claimed, fmt.Errorf("decode dream task: %w", err))
	}
	outcome, err := w.generator.Generate(ctx, payload)
	if err != nil {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		if settleErr := w.failJob(ctx, claimed, err); settleErr != nil {
			return true, fmt.Errorf("generate dream candidate: %w; settle failure: %v", err, settleErr)
		}
		slog.Warn("dream job scheduled for retry or dead letter", "worker", w.workerID, "job_id", claimed.ID, "attempt", claimed.Attempts, "error", safeWorkerError(err))
		return true, nil
	}
	resultPayload, err := json.Marshal(struct {
		SchemaVersion int                       `json:"schemaVersion"`
		Status        string                    `json:"status"`
		Candidate     port.DreamCandidateResult `json:"candidate"`
	}{
		SchemaVersion: 1,
		Status:        dreamJobStatus(outcome),
		Candidate:     outcome,
	})
	if err != nil {
		return true, err
	}
	if err := w.jobs.Complete(ctx, claimed.ID, w.workerID, port.AIJobResult{RunID: outcome.RunID, Payload: resultPayload}); err != nil {
		return true, err
	}
	return true, nil
}

func dreamJobStatus(outcome port.DreamCandidateResult) string {
	if outcome.Created {
		return "review_created"
	}
	return "skipped"
}

func (w *DreamWorker) failJob(ctx context.Context, job port.AIJob, err error) error {
	if job.Attempts >= job.MaxAttempts {
		return w.deadLetter(ctx, job, err)
	}
	return w.jobs.Retry(ctx, job.ID, w.workerID, w.currentTime().Add(retryDelay(w.retryBase, job.Attempts)), safeWorkerError(err))
}

func (w *DreamWorker) deadLetter(ctx context.Context, job port.AIJob, err error) error {
	return w.jobs.DeadLetter(ctx, job.ID, w.workerID, safeWorkerError(err))
}

func (w *DreamWorker) currentTime() time.Time {
	if w == nil || w.now == nil {
		return time.Now().UTC()
	}
	return w.now().UTC()
}
