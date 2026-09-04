package task

import (
	"context"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

type fakeContentProjectionRepositoryForWorker struct {
	rows []port.ContentProjection
}

func (f *fakeContentProjectionRepositoryForWorker) Upsert(context.Context, port.ContentProjection) error {
	return nil
}

func (f *fakeContentProjectionRepositoryForWorker) MarkDeleted(context.Context, int, time.Time) error {
	return nil
}

func (f *fakeContentProjectionRepositoryForWorker) List(_ context.Context, filter port.ContentProjectionFilter) ([]port.ContentProjection, error) {
	if filter.Status != port.ContentProjectionPending {
		return nil, nil
	}
	return f.rows, nil
}

type fakeContentProjectionProcessorForWorker struct {
	called int
}

func (f *fakeContentProjectionProcessorForWorker) ProcessPending(_ context.Context, rows []port.ContentProjection) (int, error) {
	f.called = len(rows)
	return len(rows), nil
}

func TestContentProjectionWorkerProcessesPendingBatch(t *testing.T) {
	repository := &fakeContentProjectionRepositoryForWorker{rows: []port.ContentProjection{{ArticleID: 1, Status: port.ContentProjectionPending}}}
	processor := &fakeContentProjectionProcessorForWorker{}
	worker, err := NewContentProjectionWorker(ContentProjectionWorkerDeps{
		Projections: repository,
		Processor:   processor,
		BatchSize:   10,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.processOne(context.Background())
	if err != nil || !processed || processor.called != 1 {
		t.Fatalf("processOne() processed=%v error=%v called=%d", processed, err, processor.called)
	}
}

func TestContentProjectionWorkerRequiresDependencies(t *testing.T) {
	if _, err := NewContentProjectionWorker(ContentProjectionWorkerDeps{}); err == nil {
		t.Fatal("expected dependency validation error")
	}
}
