package task

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

var (
	ErrSupervisorStarted = errors.New("worker supervisor already started")
	ErrSupervisorStopped = errors.New("worker supervisor is stopped")
)

// Worker is a long-running background task. It must return when ctx is
// canceled and must not create its own untracked goroutines.
type Worker func(context.Context) error

type workerSpec struct {
	name string
	work Worker
}

// Supervisor gives background workers one lifecycle: registration before
// startup, shared cancellation, bounded goroutine ownership, and a waitable
// shutdown. A worker returning unexpectedly is logged but does not spawn an
// unbounded restart loop.
type Supervisor struct {
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	workers []workerSpec
	started bool
	done    chan struct{}
	wg      sync.WaitGroup
}

func NewSupervisor(parent context.Context) *Supervisor {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	return &Supervisor{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

func (s *Supervisor) Add(name string, worker Worker) error {
	if s == nil {
		return errors.New("worker supervisor is nil")
	}
	if worker == nil {
		return errors.New("worker is nil")
	}
	if name == "" {
		return errors.New("worker name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return ErrSupervisorStarted
	}
	s.workers = append(s.workers, workerSpec{name: name, work: worker})
	return nil
}

func (s *Supervisor) Start() error {
	if s == nil {
		return errors.New("worker supervisor is nil")
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return ErrSupervisorStarted
	}
	s.started = true
	workers := append([]workerSpec(nil), s.workers...)
	s.wg.Add(len(workers))
	s.mu.Unlock()

	for _, worker := range workers {
		go s.run(worker)
	}
	go func() {
		s.wg.Wait()
		close(s.done)
	}()
	return nil
}

func (s *Supervisor) run(worker workerSpec) {
	defer s.wg.Done()
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("background worker panicked", "worker", worker.name, "error_code", "worker_panic")
		}
	}()
	if err := worker.work(s.ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		slog.Error("background worker stopped with error", "worker", worker.name, "error_code", safeWorkerError(err))
	}
}

// Wait waits for all registered workers. It is mainly useful to make process
// shutdown and focused tests deterministic.
func (s *Supervisor) Wait(ctx context.Context) error {
	if s == nil {
		return errors.New("worker supervisor is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	started := s.started
	s.mu.Unlock()
	if !started {
		return ErrSupervisorStopped
	}
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop cancels all workers and waits until each worker returns or the caller's
// shutdown deadline expires. Calling Stop more than once is safe.
func (s *Supervisor) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	started := s.started
	s.mu.Unlock()
	if !started {
		return nil
	}
	s.cancel()
	return s.Wait(ctx)
}
