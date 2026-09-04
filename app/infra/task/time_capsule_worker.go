package task

import (
	"benetnasch/app/domain/port"
	"context"
	"errors"
	"log/slog"
	"time"
)

const (
	defaultTimeCapsulePollInterval = 30 * time.Second
	defaultTimeCapsuleBatchSize    = 50
)

type TimeCapsuleWorkerDeps struct {
	Capsules     port.TimeCapsuleRepository
	Safety       port.AgentSafetySwitch
	PollInterval time.Duration
	BatchSize    int
}

// TimeCapsuleWorker materializes the sealed -> due transition. Actual
// delivery is acknowledged by the owner's authenticated GET, so this worker
// never sends email, push notifications, or profile data to an external
// system.
type TimeCapsuleWorker struct {
	capsules     port.TimeCapsuleRepository
	safety       port.AgentSafetySwitch
	pollInterval time.Duration
	batchSize    int
	now          func() time.Time
}

var _ Worker = (*TimeCapsuleWorker)(nil).Run

func NewTimeCapsuleWorker(deps TimeCapsuleWorkerDeps) (*TimeCapsuleWorker, error) {
	if deps.Capsules == nil {
		return nil, errors.New("time capsule worker requires repository")
	}
	if deps.PollInterval <= 0 {
		deps.PollInterval = defaultTimeCapsulePollInterval
	}
	if deps.BatchSize <= 0 {
		deps.BatchSize = defaultTimeCapsuleBatchSize
	}
	if deps.BatchSize > port.MaxTimeCapsulePageSize {
		deps.BatchSize = port.MaxTimeCapsulePageSize
	}
	return &TimeCapsuleWorker{
		capsules:     deps.Capsules,
		safety:       deps.Safety,
		pollInterval: deps.PollInterval,
		batchSize:    deps.BatchSize,
		now:          func() time.Time { return time.Now().UTC() },
	}, nil
}

func (w *TimeCapsuleWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("time capsule worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		if w.safety != nil {
			stopped, err := w.safety.IsStopped(ctx)
			if err != nil {
				slog.Error("time capsule safety switch read failed", "error", safeWorkerError(err))
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
		count, err := w.advanceOnce(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			slog.Error("time capsule due transition failed", "error", safeWorkerError(err))
		} else if count > 0 {
			slog.Info("time capsules moved to due", "count", count)
		}
		if !w.wait(ctx) {
			return ctx.Err()
		}
	}
}

// RunOnce advances at most one bounded batch of due time capsules.
func (w *TimeCapsuleWorker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil {
		return false, errors.New("time capsule worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := checkOneShotSafety(ctx, w.safety); err != nil {
		return false, err
	}
	count, err := w.advanceOnce(ctx)
	return count > 0, err
}

func (w *TimeCapsuleWorker) advanceOnce(ctx context.Context) (int, error) {
	if w == nil || w.capsules == nil {
		return 0, errors.New("time capsule worker is not configured")
	}
	return w.capsules.AdvanceDue(ctx, w.currentTime(), w.batchSize)
}

func (w *TimeCapsuleWorker) wait(ctx context.Context) bool {
	timer := time.NewTimer(w.pollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *TimeCapsuleWorker) currentTime() time.Time {
	if w != nil && w.now != nil {
		if value := w.now(); !value.IsZero() {
			return value.UTC()
		}
	}
	return time.Now().UTC()
}
