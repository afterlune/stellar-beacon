package task

import (
	"benetnasch/app/domain/port"
	"context"
	"testing"
	"time"
)

type timeCapsuleWorkerRepositoryFake struct {
	advanced int
	err      error
}

func (f *timeCapsuleWorkerRepositoryFake) Create(context.Context, port.TimeCapsule) error { return nil }
func (f *timeCapsuleWorkerRepositoryFake) GetOwned(context.Context, string, int, time.Time) (port.TimeCapsule, error) {
	return port.TimeCapsule{}, nil
}
func (f *timeCapsuleWorkerRepositoryFake) Seal(context.Context, string, int, time.Time) error {
	return nil
}
func (f *timeCapsuleWorkerRepositoryFake) AdvanceDue(context.Context, time.Time, int) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.advanced++
	return 2, nil
}
func (f *timeCapsuleWorkerRepositoryFake) Deliver(context.Context, string, int, time.Time) (port.TimeCapsule, error) {
	return port.TimeCapsule{}, nil
}
func (f *timeCapsuleWorkerRepositoryFake) MarkDeliveryFailed(context.Context, string, int, string, time.Time) error {
	return nil
}
func (f *timeCapsuleWorkerRepositoryFake) RetryDelivery(context.Context, string, int, time.Time) error {
	return nil
}

func TestTimeCapsuleWorkerMaterializesDueState(t *testing.T) {
	repository := &timeCapsuleWorkerRepositoryFake{}
	worker, err := NewTimeCapsuleWorker(TimeCapsuleWorkerDeps{Capsules: repository, BatchSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	count, err := worker.advanceOnce(context.Background())
	if err != nil || count != 2 || repository.advanced != 1 {
		t.Fatalf("count=%d err=%v advanced=%d", count, err, repository.advanced)
	}
	if worker.batchSize != 10 {
		t.Fatalf("batch size = %d, want 10", worker.batchSize)
	}
}

func TestTimeCapsuleWorkerBoundsBatchSize(t *testing.T) {
	worker, err := NewTimeCapsuleWorker(TimeCapsuleWorkerDeps{
		Capsules:  &timeCapsuleWorkerRepositoryFake{},
		BatchSize: port.MaxTimeCapsulePageSize + 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if worker.batchSize != port.MaxTimeCapsulePageSize {
		t.Fatalf("batch size = %d, want %d", worker.batchSize, port.MaxTimeCapsulePageSize)
	}
}
