package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
	"xorm.io/xorm"
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
	cancel  context.CancelFunc
	optCh   chan port.TOperationLog
	exCh    chan port.TExceptionLog
	saveOpt func(context.Context, port.TOperationLog) error
	saveEx  func(context.Context, port.TExceptionLog) error
	mu      sync.RWMutex
	closed  bool
	once    sync.Once
	wg      sync.WaitGroup
}

var defaultLogQueue atomic.Pointer[LogQueue]

// NewLogQueue starts a bounded log queue. The context supplies values to the
// persistence adapter; the queue owns worker cancellation so Stop can drain
// entries even when the application context has already been canceled.
func NewLogQueue(ctx context.Context, engine *xorm.Engine) *LogQueue {
	if ctx == nil {
		ctx = context.Background()
	}
	queue := newLogQueue(
		ctx,
		func(ctx context.Context, log port.TOperationLog) error {
			return saveOptLog(engine, ctx, log)
		},
		func(ctx context.Context, log port.TExceptionLog) error {
			return saveExLog(engine, ctx, log)
		},
	)
	queue.wg.Add(logQueueWorkers)
	for i := 0; i < logQueueWorkers; i++ {
		go queue.worker()
	}
	return queue
}

func newLogQueue(ctx context.Context, saveOpt func(context.Context, port.TOperationLog) error, saveEx func(context.Context, port.TExceptionLog) error) *LogQueue {
	if ctx == nil {
		ctx = context.Background()
	}
	// A request/application cancellation must not race with the explicit
	// graceful drain in Stop. Keep context values for the persistence adapter,
	// but let this queue own worker cancellation so queued audit records are not
	// silently abandoned when the server receives SIGTERM.
	workerCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	queue := &LogQueue{
		ctx:     workerCtx,
		cancel:  cancel,
		optCh:   make(chan port.TOperationLog, logQueueCapacity),
		exCh:    make(chan port.TExceptionLog, logQueueCapacity),
		saveOpt: saveOpt,
		saveEx:  saveEx,
	}
	return queue
}

// StartLogQueue installs q as the process-wide sink used by middleware.
func StartLogQueue(ctx context.Context, engine *xorm.Engine) *LogQueue {
	queue := NewLogQueue(ctx, engine)
	if previous := defaultLogQueue.Swap(queue); previous != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = previous.Stop(shutdownCtx)
		cancel()
	}
	return queue
}

// EnqueueOptLog queues an operation log or records a drop when the queue is full.
func EnqueueOptLog(log port.TOperationLog) bool {
	queue := defaultLogQueue.Load()
	if queue == nil {
		slog.Warn("operation log dropped: queue is not started")
		return false
	}
	return queue.enqueueOpt(log)
}

// EnqueueExLog queues an exception log or records a drop when the queue is full.
func EnqueueExLog(log port.TExceptionLog) bool {
	queue := defaultLogQueue.Load()
	if queue == nil {
		slog.Warn("exception log dropped: queue is not started")
		return false
	}
	return queue.enqueueEx(log)
}

func (q *LogQueue) enqueueOpt(log port.TOperationLog) bool {
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

func (q *LogQueue) enqueueEx(log port.TExceptionLog) bool {
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

func (q *LogQueue) persistOpt(log port.TOperationLog) {
	q.retry("operation", func() error { return q.saveOpt(q.ctx, log) })
}

func (q *LogQueue) persistEx(log port.TExceptionLog) {
	q.retry("exception", func() error { return q.saveEx(q.ctx, log) })
}

func (q *LogQueue) retry(kind string, save func() error) {
	var err error
	for attempt := 1; attempt <= logRetryAttempts; attempt++ {
		if err = save(); err == nil {
			return
		}
		if !isRetryableLogError(err) {
			slog.Error("persist log failed", "kind", kind, "attempt", attempt, "retryable", false, "error_code", apperrors.SafeCode(err))
			return
		}
		if attempt == logRetryAttempts {
			slog.Error("persist log failed after retries", "kind", kind, "attempts", attempt, "retryable", true, "error_code", apperrors.SafeCode(err))
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
		q.cancel()
		defaultLogQueue.CompareAndSwap(q, nil)
		return nil
	case <-ctx.Done():
		// A timed-out shutdown must still release the worker context so retry
		// timers and in-flight database operations can observe cancellation.
		q.cancel()
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
