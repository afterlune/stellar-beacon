package port

import (
	"context"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
)

// JobTarget describes one built-in job handler. Targets are intentionally
// closed: persisted invoke_target values can only run code registered by the
// composition root.
type JobTarget struct {
	Target      string `json:"target"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CronExample string `json:"cronExample"`
}

// JobRunResult is the outcome returned by a job target.
type JobRunResult struct {
	Processed bool   `json:"processed"`
	Message   string `json:"message"`
}

// JobScheduler owns Cron validation, scheduling and manual execution.
type JobScheduler interface {
	Targets() []JobTarget
	SupportsTarget(target string) bool
	NextRun(expression string, after time.Time) (time.Time, error)
	Reload(ctx context.Context) error
	Run(ctx context.Context, job entity.TJob, trigger string) (JobRunResult, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Ready() bool
}

// UserAreaRefresher recomputes the cached user-area distribution.
type UserAreaRefresher interface {
	RefreshUserAreas(ctx context.Context) (bool, error)
}
