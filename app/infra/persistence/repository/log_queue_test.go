package repository

import (
	"benetnasch/app/domain/entity"
	"context"
	"errors"
	"testing"
	"time"
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
