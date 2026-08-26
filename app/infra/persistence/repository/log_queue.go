package repository

import (
	"benetnasch/app/domain/entity"
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
)

const (
	logQueueCapacity = 1024
	logQueueWorkers  = 2
	logRetryAttempts = 3
)

// LogQueue is a bounded asynchronous sink for request and exception logs.
// Enqueue never blocks request handling and never starts an unbounded goroutine.
type LogQueue struct {
	ctx     context.Context
	optCh   chan entity.TOperationLog
	exCh    chan entity.TExceptionLog
	saveOpt func(context.Context, entity.TOperationLog) error
	saveEx  func(context.Context, entity.TExceptionLog) error
	mu      sync.RWMutex
	closed  bool
	once    sync.Once
	wg      sync.WaitGroup
}

var defaultLogQueue atomic.Pointer[LogQueue]

// NewLogQueue starts a bounded log queue. The context belongs to the
// background worker and may be canceled by the application during shutdown.
func NewLogQueue(ctx context.Context) *LogQueue {
	if ctx == nil {
		ctx = context.Background()
	}
	queue := newLogQueue(ctx, SaveOptLog, SaveExLog)
	queue.wg.Add(logQueueWorkers)
	for i := 0; i < logQueueWorkers; i++ {
		go queue.worker()
	}
	return queue
}

func newLogQueue(ctx context.Context, saveOpt func(context.Context, entity.TOperationLog) error, saveEx func(context.Context, entity.TExceptionLog) error) *LogQueue {
	if ctx == nil {
		ctx = context.Background()
	}
	queue := &LogQueue{
		ctx:     ctx,
		optCh:   make(chan entity.TOperationLog, logQueueCapacity),
		exCh:    make(chan entity.TExceptionLog, logQueueCapacity),
		saveOpt: saveOpt,
		saveEx:  saveEx,
	}
	return queue
}

// StartLogQueue installs q as the process-wide sink used by middleware.
func StartLogQueue(ctx context.Context) *LogQueue {
	queue := NewLogQueue(ctx)
	if previous := defaultLogQueue.Swap(queue); previous != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = previous.Stop(shutdownCtx)
		cancel()
	}
	return queue
}

// EnqueueOptLog queues an operation log or records a drop when the queue is full.
func EnqueueOptLog(log entity.TOperationLog) bool {
	queue := defaultLogQueue.Load()
	if queue == nil {
		slog.Warn("operation log dropped: queue is not started")
		return false
	}
	return queue.enqueueOpt(log)
}

// EnqueueExLog queues an exception log or records a drop when the queue is full.
func EnqueueExLog(log entity.TExceptionLog) bool {
	queue := defaultLogQueue.Load()
	if queue == nil {
		slog.Warn("exception log dropped: queue is not started")
		return false
	}
	return queue.enqueueEx(log)
}

func (q *LogQueue) enqueueOpt(log entity.TOperationLog) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return false
	}
	select {
	case q.optCh <- log:
		return true
	default:
		slog.Warn("operation log dropped: queue is full", "capacity", logQueueCapacity)
		return false
	}
}

func (q *LogQueue) enqueueEx(log entity.TExceptionLog) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return false
	}
	select {
	case q.exCh <- log:
		return true
	default:
		slog.Warn("exception log dropped: queue is full", "capacity", logQueueCapacity)
		return false
	}
}

func (q *LogQueue) worker() {
	defer q.wg.Done()
	optCh := q.optCh
	exCh := q.exCh
	for optCh != nil || exCh != nil {
		select {
		case opt, ok := <-optCh:
			if !ok {
				optCh = nil
				continue
			}
			q.persistOpt(opt)
		case ex, ok := <-exCh:
			if !ok {
				exCh = nil
				continue
			}
			q.persistEx(ex)
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *LogQueue) persistOpt(log entity.TOperationLog) {
	q.retry("operation", func() error { return q.saveOpt(q.ctx, log) })
}

func (q *LogQueue) persistEx(log entity.TExceptionLog) {
	q.retry("exception", func() error { return q.saveEx(q.ctx, log) })
}

func (q *LogQueue) retry(kind string, save func() error) {
	var err error
	for attempt := 1; attempt <= logRetryAttempts; attempt++ {
		if err = save(); err == nil {
			return
		}
		if !isRetryableLogError(err) {
			slog.Error("persist log failed", "kind", kind, "attempt", attempt, "retryable", false, "error", err)
			return
		}
		if attempt == logRetryAttempts {
			slog.Error("persist log failed after retries", "kind", kind, "attempts", attempt, "retryable", true, "error", err)
			return
		}
		delay := time.Duration(1<<(attempt-1)) * 100 * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-q.ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

func isRetryableLogError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		code := string(pqErr.Code)
		if len(code) < 2 {
			return true
		}
		switch code[:2] {
		case "08", "40", "53", "57", "58":
			return true
		default:
			return false
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	return true
}

// Stop closes the queue, drains pending entries, and waits up to ctx's deadline.
func (q *LogQueue) Stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	q.once.Do(func() {
		q.mu.Lock()
		q.closed = true
		close(q.optCh)
		close(q.exCh)
		q.mu.Unlock()
	})
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		defaultLogQueue.CompareAndSwap(q, nil)
		return nil
	case <-ctx.Done():
		return errors.Join(ctx.Err(), errors.New("log queue shutdown timed out"))
	}
}

// StopLogQueue stops the process-wide sink if one is installed.
func StopLogQueue(ctx context.Context) error {
	queue := defaultLogQueue.Load()
	if queue == nil {
		return nil
	}
	return queue.Stop(ctx)
}
