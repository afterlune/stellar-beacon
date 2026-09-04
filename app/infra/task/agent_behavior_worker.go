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
	defaultAgentBehaviorPollInterval          = time.Second
	defaultAgentBehaviorScanInterval          = 15 * time.Minute
	defaultAgentBehaviorForgottenScanInterval = 24 * time.Hour
	defaultAgentBehaviorLease                 = 5 * time.Minute
	defaultAgentBehaviorRetryBase             = 2 * time.Second
)

type behaviorTaskScheduler interface {
	Scan(context.Context, int) (int, int, error)
}

type forgottenBehaviorTaskScheduler interface {
	ScanForgotten(context.Context, int) (int, int, error)
}

type AgentBehaviorWorkerDeps struct {
	Jobs          port.AIJobRepository
	Scheduler     behaviorTaskScheduler
	Generator     port.AgentBehaviorGenerator
	Safety        port.AgentSafetySwitch
	WorkerID      string
	PollInterval  time.Duration
	ScanInterval  time.Duration
	LeaseDuration time.Duration
	RetryBase     time.Duration
}

// AgentBehaviorWorker scans public article snapshots into reading jobs and
// turns claimed jobs into pending review records. It intentionally has no
// comment, talk, article, or publication write dependency.
type AgentBehaviorWorker struct {
	jobs                  port.AIJobRepository
	scheduler             behaviorTaskScheduler
	generator             port.AgentBehaviorGenerator
	safety                port.AgentSafetySwitch
	workerID              string
	pollInterval          time.Duration
	scanInterval          time.Duration
	leaseDuration         time.Duration
	retryBase             time.Duration
	forgottenScanInterval time.Duration
	now                   func() time.Time
}

var _ Worker = (*AgentBehaviorWorker)(nil).Run

func NewAgentBehaviorWorker(deps AgentBehaviorWorkerDeps) (*AgentBehaviorWorker, error) {
	if deps.Jobs == nil || deps.Scheduler == nil || deps.Generator == nil {
		return nil, fmt.Errorf("agent behavior worker requires jobs, scheduler and generator")
	}
	if strings.TrimSpace(deps.WorkerID) == "" {
		deps.WorkerID = "agent-behavior-" + uuid.NewString()
	}
	if deps.PollInterval <= 0 {
		deps.PollInterval = defaultAgentBehaviorPollInterval
	}
	if deps.ScanInterval <= 0 {
		deps.ScanInterval = defaultAgentBehaviorScanInterval
	}
	if deps.LeaseDuration <= 0 {
		deps.LeaseDuration = defaultAgentBehaviorLease
	}
	if deps.RetryBase <= 0 {
		deps.RetryBase = defaultAgentBehaviorRetryBase
	}
	return &AgentBehaviorWorker{
		jobs:                  deps.Jobs,
		scheduler:             deps.Scheduler,
		generator:             deps.Generator,
		safety:                deps.Safety,
		workerID:              strings.TrimSpace(deps.WorkerID),
		pollInterval:          deps.PollInterval,
		scanInterval:          deps.ScanInterval,
		leaseDuration:         deps.LeaseDuration,
		retryBase:             deps.RetryBase,
		forgottenScanInterval: defaultAgentBehaviorForgottenScanInterval,
		now:                   func() time.Time { return time.Now().UTC() },
	}, nil
}

func (w *AgentBehaviorWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("agent behavior worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	lastScan := time.Time{}
	cursor := 0
	lastForgottenScan := time.Time{}
	forgottenCursor := 0
	for {
		if w.safety != nil {
			stopped, safetyErr := w.safety.IsStopped(ctx)
			if safetyErr != nil {
				slog.Error("agent behavior safety switch read failed", "worker", w.workerID, "error", safeWorkerError(safetyErr))
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
				slog.Error("agent behavior scan failed", "worker", w.workerID, "error", safeWorkerError(err))
			} else {
				cursor = next
				lastScan = w.currentTime()
				if count > 0 {
					slog.Info("agent behavior reading tasks enqueued", "worker", w.workerID, "count", count)
				}
			}
		}
		if scheduler, ok := w.scheduler.(forgottenBehaviorTaskScheduler); ok && (lastForgottenScan.IsZero() || w.currentTime().Sub(lastForgottenScan) >= w.forgottenScanInterval) {
			next, count, err := scheduler.ScanForgotten(ctx, forgottenCursor)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				slog.Error("agent forgotten-content scan failed", "worker", w.workerID, "error", safeWorkerError(err))
			} else {
				forgottenCursor = next
				lastForgottenScan = w.currentTime()
				if count > 0 {
					slog.Info("agent forgotten-content tasks enqueued", "worker", w.workerID, "count", count)
				}
			}
		}
		processed, err := w.processOne(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			slog.Error("agent behavior worker iteration failed", "worker", w.workerID, "error", safeWorkerError(err))
		}
		if processed {
			continue
		}
		if !w.wait(ctx) {
			return ctx.Err()
		}
	}
}

// RunOnce consumes at most one already-enqueued behavior reading job. It
// deliberately skips the periodic scanner; manual execution cannot broaden
// the set of articles or actions selected by the configured policy.
func (w *AgentBehaviorWorker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil {
		return false, errors.New("agent behavior worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := checkOneShotSafety(ctx, w.safety); err != nil {
		return false, err
	}
	return w.processOne(ctx)
}

func (w *AgentBehaviorWorker) wait(ctx context.Context) bool {
	timer := time.NewTimer(w.pollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *AgentBehaviorWorker) processOne(ctx context.Context) (bool, error) {
	claimed, ok, err := port.ClaimAIJob(ctx, w.jobs, w.workerID, port.AIJobKindAgentArticleReading, w.currentTime(), w.leaseDuration)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	payload, err := agentapp.DecodeAgentReadingTask(claimed.Payload)
	if err != nil {
		return true, w.deadLetter(ctx, claimed, fmt.Errorf("decode agent reading task: %w", err))
	}
	outcome, err := w.generator.Generate(ctx, payload)
	if err != nil {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		if settleErr := w.failJob(ctx, claimed, err); settleErr != nil {
			return true, fmt.Errorf("generate behavior candidate: %w; settle failure: %v", err, settleErr)
		}
		slog.Warn("agent behavior job scheduled for retry or dead letter", "worker", w.workerID, "job_id", claimed.ID, "attempt", claimed.Attempts, "error", safeWorkerError(err))
		return true, nil
	}
	resultPayload, err := json.Marshal(struct {
		SchemaVersion int                               `json:"schemaVersion"`
		ArticleID     int                               `json:"articleId"`
		Action        port.AgentBehaviorAction          `json:"action"`
		Status        string                            `json:"status"`
		Candidate     port.AgentBehaviorCandidateResult `json:"candidate"`
	}{
		SchemaVersion: 1,
		ArticleID:     payload.ArticleID,
		Action:        payload.Action,
		Status:        behaviorJobStatus(outcome),
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

func behaviorJobStatus(outcome port.AgentBehaviorCandidateResult) string {
	if outcome.Created {
		return "review_created"
	}
	return "skipped"
}

func (w *AgentBehaviorWorker) failJob(ctx context.Context, job port.AIJob, err error) error {
	if job.Attempts >= job.MaxAttempts {
		return w.deadLetter(ctx, job, err)
	}
	return w.jobs.Retry(ctx, job.ID, w.workerID, w.currentTime().Add(retryDelay(w.retryBase, job.Attempts)), safeWorkerError(err))
}

func (w *AgentBehaviorWorker) deadLetter(ctx context.Context, job port.AIJob, err error) error {
	return w.jobs.DeadLetter(ctx, job.ID, w.workerID, safeWorkerError(err))
}

func (w *AgentBehaviorWorker) currentTime() time.Time {
	if w == nil || w.now == nil {
		return time.Now().UTC()
	}
	return w.now().UTC()
}
