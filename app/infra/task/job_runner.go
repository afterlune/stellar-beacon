package task

import (
	"context"
	"errors"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const defaultManualJobTimeout = 5 * time.Minute

// OneShotHandler processes at most one durable item and returns whether an
// item was found. It is intentionally narrower than Worker: it cannot start
// a loop or create an untracked goroutine.
type OneShotHandler func(context.Context) (processed bool, err error)

func checkOneShotSafety(ctx context.Context, safety port.AgentSafetySwitch) error {
	if safety == nil {
		return nil
	}
	stopped, err := safety.IsStopped(ctx)
	if err != nil {
		return apperrors.Unavailable("agent.safety_switch", err)
	}
	if stopped {
		return apperrors.New(apperrors.KindForbidden, "agent.safety_switch", nil)
	}
	return nil
}

// AllowlistedJobRunner is assembled only in bootstrap. The constructor
// rejects every target outside the domain allowlist, and Run performs the
// same check again at the request boundary.
type AllowlistedJobRunner struct {
	handlers map[string]OneShotHandler
	safety   port.AgentSafetySwitch
	timeout  time.Duration
}

var _ port.JobRunner = (*AllowlistedJobRunner)(nil)

func NewAllowlistedJobRunner(handlers map[string]OneShotHandler) (*AllowlistedJobRunner, error) {
	return newAllowlistedJobRunner(handlers, defaultManualJobTimeout, nil)
}

func NewAllowlistedJobRunnerWithTimeout(handlers map[string]OneShotHandler, timeout time.Duration) (*AllowlistedJobRunner, error) {
	return newAllowlistedJobRunner(handlers, timeout, nil)
}

func NewAllowlistedJobRunnerWithSafety(handlers map[string]OneShotHandler, safety port.AgentSafetySwitch) (*AllowlistedJobRunner, error) {
	return newAllowlistedJobRunner(handlers, defaultManualJobTimeout, safety)
}

func newAllowlistedJobRunner(handlers map[string]OneShotHandler, timeout time.Duration, safety port.AgentSafetySwitch) (*AllowlistedJobRunner, error) {
	if timeout <= 0 || timeout > 30*time.Minute {
		return nil, apperrors.Invalid("job.runner.timeout", "manual job timeout is invalid")
	}
	copyHandlers := make(map[string]OneShotHandler, len(handlers))
	for rawTarget, handler := range handlers {
		target := strings.TrimSpace(rawTarget)
		if !port.IsManualJobTarget(target) {
			return nil, apperrors.Invalid("job.runner.target", "manual job target is not allowlisted")
		}
		if handler == nil {
			return nil, apperrors.Invalid("job.runner.handler", "manual job handler is nil")
		}
		copyHandlers[target] = handler
	}
	return &AllowlistedJobRunner{handlers: copyHandlers, safety: safety, timeout: timeout}, nil
}

func (r *AllowlistedJobRunner) CanRun(target string) bool {
	if r == nil {
		return false
	}
	_, ok := r.handlers[strings.TrimSpace(target)]
	return ok
}

func (r *AllowlistedJobRunner) Run(ctx context.Context, request port.JobRunRequest) (port.JobRunOutcome, error) {
	if r == nil {
		return port.JobRunOutcome{}, apperrors.Unavailable("job.runner", nil)
	}
	if err := request.Validate(); err != nil {
		return port.JobRunOutcome{}, apperrors.Invalid("job.runner.request", err.Error())
	}
	target := strings.TrimSpace(request.InvokeTarget)
	handler, ok := r.handlers[target]
	if !ok {
		return port.JobRunOutcome{}, apperrors.Unavailable("job.runner.disabled", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	if err := checkOneShotSafety(runCtx, r.safety); err != nil {
		return port.JobRunOutcome{}, err
	}
	processed, err := handler(runCtx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return port.JobRunOutcome{}, err
		}
		switch apperrors.KindOf(err) {
		case apperrors.KindValidation, apperrors.KindForbidden, apperrors.KindConflict:
			return port.JobRunOutcome{}, err
		}
		return port.JobRunOutcome{}, apperrors.Unavailable("job.runner.run", err)
	}
	return port.JobRunOutcome{JobID: request.ID, Target: target, Processed: processed}, nil
}
