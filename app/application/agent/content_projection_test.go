package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

type fakeContentProjectionRepositoryForAgent struct {
	saved []port.ContentProjection
	err   error
}

func (f *fakeContentProjectionRepositoryForAgent) Upsert(_ context.Context, projection port.ContentProjection) error {
	if f.err != nil {
		return f.err
	}
	f.saved = append(f.saved, projection)
	return nil
}

func (f *fakeContentProjectionRepositoryForAgent) MarkDeleted(context.Context, int, time.Time) error {
	return nil
}

func (f *fakeContentProjectionRepositoryForAgent) List(context.Context, port.ContentProjectionFilter) ([]port.ContentProjection, error) {
	return nil, nil
}

func pendingContentProjectionForAgent(id int, values []float32) port.ContentProjection {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	return port.ContentProjection{
		ArticleID:          id,
		LifeStage:          port.ContentLifeStageSettled,
		EmbeddingModel:     "test-model",
		EmbeddingVersion:   "test-v1",
		EmbeddingDimension: len(values),
		PCAInput:           values,
		PCAInputVersion:    port.DefaultPCAInputVersion,
		Status:             port.ContentProjectionPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func TestContentProjectionBatchProcessorPersistsDeterministicCoordinates(t *testing.T) {
	repository := &fakeContentProjectionRepositoryForAgent{}
	processor, err := NewContentProjectionBatchProcessor(repository)
	if err != nil {
		t.Fatal(err)
	}
	processor.now = func() time.Time { return time.Date(2026, 8, 29, 13, 0, 0, 0, time.UTC) }
	count, err := processor.ProcessPending(context.Background(), []port.ContentProjection{
		pendingContentProjectionForAgent(3, []float32{3, 0, 0}),
		pendingContentProjectionForAgent(1, []float32{1, 0, 0}),
		pendingContentProjectionForAgent(2, []float32{2, 0, 0}),
	})
	if err != nil || count != 3 {
		t.Fatalf("ProcessPending() count=%d error=%v", count, err)
	}
	if len(repository.saved) != 3 {
		t.Fatalf("saved projections=%d, want 3", len(repository.saved))
	}
	for _, projection := range repository.saved {
		if projection.Status != port.ContentProjectionReady || projection.ProjectionX == nil || projection.ProjectionY == nil {
			t.Fatalf("projection was not made ready: %+v", projection)
		}
		if !projection.UpdatedAt.Equal(time.Date(2026, 8, 29, 13, 0, 0, 0, time.UTC)) {
			t.Fatalf("updated at=%s", projection.UpdatedAt)
		}
	}
}

func TestContentProjectionBatchProcessorRejectsInvalidInputAndHonorsCancellation(t *testing.T) {
	repository := &fakeContentProjectionRepositoryForAgent{}
	processor, err := NewContentProjectionBatchProcessor(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := processor.ProcessPending(context.Background(), []port.ContentProjection{
		pendingContentProjectionForAgent(1, []float32{1}),
		pendingContentProjectionForAgent(2, []float32{1, 2}),
	}); err == nil {
		t.Fatal("expected mismatched dimensions to fail")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = processor.ProcessPending(ctx, []port.ContentProjection{pendingContentProjectionForAgent(1, []float32{1, 2})})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled ProcessPending() error=%v", err)
	}
	if len(repository.saved) != 0 {
		t.Fatalf("canceled processor wrote %d projections", len(repository.saved))
	}
}
