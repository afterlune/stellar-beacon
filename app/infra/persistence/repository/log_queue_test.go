package repository

import (
	"benetnasch/app/domain/entity"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestLogQueueRetriesAndDrainsOnStop(t *testing.T) {
	attempts := 0
	queue := newLogQueue(
		context.Background(),
		func(context.Context, entity.TOperationLog) error {
			attempts++
			if attempts < logRetryAttempts {
				return errors.New("temporary database failure")
			}
			return nil
		},
		func(context.Context, entity.TExceptionLog) error { return nil },
	)
	queue.wg.Add(logQueueWorkers)
	for i := 0; i < logQueueWorkers; i++ {
		go queue.worker()
	}

	if !queue.enqueueOpt(entity.TOperationLog{OptUri: "/test"}) {
		t.Fatal("enqueueOpt() unexpectedly dropped the event")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := queue.Stop(shutdownCtx); err != nil {
		t.Fatal(err)
	}
	if attempts != logRetryAttempts {
		t.Fatalf("persist attempts = %d, want %d", attempts, logRetryAttempts)
	}
}

func TestLogQueueDrainsAfterParentContextCancellation(t *testing.T) {
	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	persisted := make(chan struct{}, 1)
	queue := newLogQueue(
		parentCtx,
		func(ctx context.Context, _ entity.TOperationLog) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			persisted <- struct{}{}
			return nil
		},
		func(context.Context, entity.TExceptionLog) error { return nil },
	)
	queue.wg.Add(logQueueWorkers)
	for i := 0; i < logQueueWorkers; i++ {
		go queue.worker()
	}

	cancelParent()
	if !queue.enqueueOpt(entity.TOperationLog{OptUri: "/after-cancel"}) {
		t.Fatal("enqueueOpt() unexpectedly dropped the event")
	}
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := queue.Stop(shutdownCtx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-persisted:
	default:
		t.Fatal("queued operation log was not drained after parent cancellation")
	}
}

func TestLogQueueDoesNotRetryPermanentDatabaseErrors(t *testing.T) {
	attempts := 0
	queue := newLogQueue(
		context.Background(),
		func(context.Context, entity.TOperationLog) error {
			attempts++
			return &pq.Error{Code: "23505", Message: "duplicate key"}
		},
		func(context.Context, entity.TExceptionLog) error { return nil },
	)

	queue.retry("operation", func() error { return queue.saveOpt(queue.ctx, entity.TOperationLog{}) })
	if attempts != 1 {
		t.Fatalf("permanent database error attempts = %d, want 1", attempts)
	}
}
