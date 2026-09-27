package repository

import (
	"context"
	"errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
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
