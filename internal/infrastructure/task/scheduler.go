package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/robfig/cron/v3"
)

const (
	scheduledRunTimeout = 5 * time.Minute
	manualRunTimeout    = 30 * time.Second
	jobLockTTL          = 6 * time.Minute
	jobLogWriteTimeout  = 5 * time.Second
)

// Handler is one built-in job target. Handlers must be idempotent because a
// failed process can leave a distributed lock unavailable until its TTL ends.
type Handler func(context.Context) (port.JobRunResult, error)

type registeredTarget struct {
	meta    port.JobTarget
	handler Handler
}

// Scheduler is the process-local Cron scheduler backed by persisted job
// configuration. Redis locks coordinate execution across backend replicas.
type Scheduler struct {
	repo  port.JobRepository
	logs  port.JobLogRepository
	cache port.Cache

	targets map[string]registeredTarget

	mu      sync.Mutex
	cron    *cron.Cron
	entries map[int]cron.EntryID
	ctx     context.Context
	cancel  context.CancelFunc
	started bool

	runningMu sync.Mutex
	running   map[int]struct{}
	runWG     sync.WaitGroup
	ready     atomic.Bool
}

func NewScheduler(repo port.JobRepository, logs port.JobLogRepository, cache port.Cache) *Scheduler {
	return &Scheduler{
		repo:    repo,
		logs:    logs,
		cache:   cache,
		targets: make(map[string]registeredTarget),
		entries: make(map[int]cron.EntryID),
		running: make(map[int]struct{}),
	}
}

func (s *Scheduler) Register(target port.JobTarget, handler Handler) error {
	target.Target = strings.TrimSpace(target.Target)
	if target.Target == "" {
		return apperrors.Invalid("job.target", "target is required")
	}
	if handler == nil {
		return apperrors.Invalid("job.target", "handler is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.targets[target.Target]; exists {
		return apperrors.Conflict("job.target", "target already registered")
	}
	s.targets[target.Target] = registeredTarget{meta: target, handler: handler}
	return nil
}

func (s *Scheduler) Targets() []port.JobTarget {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]port.JobTarget, 0, len(s.targets))
	for _, target := range s.targets {
		result = append(result, target.meta)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Target < result[j].Target })
	return result
}

func (s *Scheduler) SupportsTarget(target string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.targets[strings.TrimSpace(target)]
	return ok
}

func (s *Scheduler) NextRun(expression string, after time.Time) (time.Time, error) {
	schedule, err := parseStandardExpression(expression)
	if err != nil {
		return time.Time{}, err
	}
	if after.IsZero() {
		after = time.Now()
	}
	return schedule.Next(after), nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.cron = cron.New(cron.WithParser(cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)))
	s.started = true
	s.cron.Start()
	s.mu.Unlock()

	if err := s.Reload(context.Background()); err != nil {
		_ = s.Stop(context.Background())
		return err
	}
	s.ready.Store(true)
	return nil
}

func (s *Scheduler) Ready() bool { return s.ready.Load() }

func (s *Scheduler) RunningCount() int {
	if s == nil {
		return 0
	}
	s.runningMu.Lock()
	defer s.runningMu.Unlock()
	return len(s.running)
}

func (s *Scheduler) Reload(ctx context.Context) error {
	if s.repo == nil {
		return apperrors.Unavailable("job.reload", errors.New("job repository is not configured"))
	}
	targets := s.Targets()
	registered := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		registered[target.Target] = struct{}{}
	}
	jobs, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return err
	}
	active := make([]entity.TJob, 0, len(jobs))
	for _, job := range jobs {
		if _, ok := registered[job.InvokeTarget]; !ok {
			continue
		}
		if _, err := parseStandardExpression(job.CronExpression); err != nil {
			continue
		}
		active = append(active, job)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started || s.cron == nil {
		return nil
	}
	next := make(map[int]struct{}, len(active))
	for _, job := range active {
		next[job.Id] = struct{}{}
	}
	for id, entryID := range s.entries {
		if _, keep := next[id]; keep {
			continue
		}
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
	for _, job := range active {
		if old, ok := s.entries[job.Id]; ok {
			s.cron.Remove(old)
			delete(s.entries, job.Id)
		}
		entryID, err := s.cron.AddFunc(job.CronExpression, s.scheduledFunc(job.Id))
		if err != nil {
			return err
		}
		s.entries[job.Id] = entryID
	}
	return nil
}

func (s *Scheduler) scheduledFunc(jobID int) func() {
	return func() {
		ctx := s.Context()
		if ctx == nil {
			return
		}
		job, err := s.repo.Get(ctx, jobID)
		if err != nil {
			slog.Error("load scheduled job failed", "jobId", jobID, "error", err)
			return
		}
		if job.Status != 1 {
			return
		}
		if _, err := s.Run(ctx, job, "schedule"); err != nil {
			slog.Error("scheduled job failed", "jobId", jobID, "target", job.InvokeTarget, "error", err)
		}
	}
}

func (s *Scheduler) Context() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}

func (s *Scheduler) Run(ctx context.Context, job entity.TJob, trigger string) (port.JobRunResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	target, ok := s.targets[job.InvokeTarget]
	s.mu.Unlock()
	if !ok {
		return port.JobRunResult{}, apperrors.Invalid("job.run", "job target is not registered")
	}

	if job.Concurrent == 0 {
		if !s.acquireLocal(job.Id) {
			return port.JobRunResult{Processed: false, Message: "job is already running"}, nil
		}
		defer s.releaseLocal(job.Id)
	}

	lockKey := fmt.Sprintf("stellar-beacon:job:%d:lock", job.Id)
	if job.Concurrent == 0 && s.cache != nil {
		acquired, err := s.cache.SetNX(ctx, lockKey, "1", jobLockTTL)
		if err != nil {
			return port.JobRunResult{}, apperrors.Unavailable("job.lock", err)
		}
		if !acquired {
			return port.JobRunResult{Processed: false, Message: "job is already running on another instance"}, nil
		}
		defer func() {
			if err := s.cache.Delete(context.Background(), lockKey); err != nil {
				slog.Warn("release job lock failed", "jobId", job.Id, "error", err)
			}
		}()
	}

	timeout := scheduledRunTimeout
	if trigger == "manual" {
		timeout = manualRunTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	result := port.JobRunResult{}
	var runErr error
	s.runWG.Add(1)
	func() {
		defer s.runWG.Done()
		result, runErr = executeHandler(runCtx, target.handler)
	}()
	finishedAt := time.Now()
	if strings.TrimSpace(result.Message) == "" {
		if runErr != nil {
			result.Message = runErr.Error()
		} else if result.Processed {
			result.Message = "job completed with work"
		} else {
			result.Message = "job completed with no work"
		}
	}
	if err := s.writeLog(job, trigger, startedAt, finishedAt, result, runErr); err != nil {
		slog.Error("write job log failed", "jobId", job.Id, "error", err)
		if runErr == nil {
			runErr = err
		}
	}
	return result, runErr
}

func executeHandler(ctx context.Context, handler Handler) (result port.JobRunResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("job handler panic: %v", recovered)
			result = port.JobRunResult{}
		}
	}()
	return handler(ctx)
}

func (s *Scheduler) Stop(ctx context.Context) error {
	s.ready.Store(false)
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	c := s.cron
	cancel := s.cancel
	s.mu.Unlock()
	if c == nil {
		if cancel != nil {
			cancel()
		}
		return nil
	}
	stopped := c.Stop()
	select {
	case <-stopped.Done():
	case <-ctx.Done():
		if cancel != nil {
			cancel()
		}
		return errors.Join(ctx.Err(), errors.New("scheduler shutdown timed out"))
	}
	done := make(chan struct{})
	go func() {
		s.runWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		if cancel != nil {
			cancel()
		}
		return errors.Join(ctx.Err(), errors.New("scheduled job shutdown timed out"))
	}
	if cancel != nil {
		cancel()
	}
	s.mu.Lock()
	s.started = false
	s.cron = nil
	s.entries = make(map[int]cron.EntryID)
	s.mu.Unlock()
	return nil
}

func (s *Scheduler) acquireLocal(id int) bool {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()
	if _, exists := s.running[id]; exists {
		return false
	}
	s.running[id] = struct{}{}
	return true
}

func (s *Scheduler) releaseLocal(id int) {
	s.runningMu.Lock()
	delete(s.running, id)
	s.runningMu.Unlock()
}

func (s *Scheduler) writeLog(job entity.TJob, trigger string, startedAt, finishedAt time.Time, result port.JobRunResult, runErr error) error {
	if s.logs == nil {
		return nil
	}
	message := fmt.Sprintf("[%s] %s", trigger, result.Message)
	exception := ""
	status := 0
	if runErr != nil {
		status = 1
		exception = runErr.Error()
	}
	entry := entity.TJobLog{
		JobId: job.Id, JobName: job.JobName, JobGroup: job.JobGroup,
		InvokeTarget: job.InvokeTarget, JobMessage: truncate(message, 500),
		Status: status, ExceptionInfo: truncate(exception, 2000),
		CreateTime: startedAt, StartTime: startedAt, EndTime: finishedAt,
	}
	ctx, cancel := context.WithTimeout(context.Background(), jobLogWriteTimeout)
	defer cancel()
	return s.logs.Create(ctx, entry)
}

func parseStandardExpression(expression string) (cron.Schedule, error) {
	expression = strings.TrimSpace(expression)
	if len(strings.Fields(expression)) != 5 {
		return nil, apperrors.Invalid("job.cron", "cron expression must contain five fields")
	}
	schedule, err := cron.ParseStandard(expression)
	if err != nil {
		return nil, apperrors.Invalid("job.cron", "cron expression is invalid")
	}
	return schedule, nil
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
