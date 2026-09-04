package service

import (
	"context"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
)

type fakeContentProjectionRepositoryForGalaxy struct {
	rows   []port.ContentProjection
	filter port.ContentProjectionFilter
	err    error
}

func (f *fakeContentProjectionRepositoryForGalaxy) Upsert(context.Context, port.ContentProjection) error {
	return nil
}

func (f *fakeContentProjectionRepositoryForGalaxy) MarkDeleted(context.Context, int, time.Time) error {
	return nil
}

func (f *fakeContentProjectionRepositoryForGalaxy) List(_ context.Context, filter port.ContentProjectionFilter) ([]port.ContentProjection, error) {
	f.filter = filter
	return f.rows, f.err
}

func TestContentGalaxyServiceReturnsOnlyPublicCoordinateFields(t *testing.T) {
	x, y := 1.25, -2.5
	updated := time.Date(2026, 8, 29, 13, 0, 0, 0, time.UTC)
	repository := &fakeContentProjectionRepositoryForGalaxy{rows: []port.ContentProjection{
		{ArticleID: 7, LifeStage: port.ContentLifeStageGrowing, Status: port.ContentProjectionReady, ProjectionX: &x, ProjectionY: &y, UpdatedAt: updated, EmbeddingModel: "secret-model", PCAInput: []float32{1, 2}},
		{ArticleID: 8, LifeStage: port.ContentLifeStageSettled, Status: port.ContentProjectionPending, UpdatedAt: updated},
	}}
	service, err := NewContentGalaxyService(repository, true)
	if err != nil {
		t.Fatal(err)
	}
	result := service.List(context.Background(), ContentGalaxyQuery{Current: "2", Size: "12", LifeStage: "GROWING", UpdatedAfter: "2026-08-28T00:00:00Z"})
	if !result.Flag {
		t.Fatalf("List() failed: %+v", result)
	}
	page, ok := result.Data.(model.ContentGalaxyPageDTO)
	if !ok || len(page.Records) != 1 {
		t.Fatalf("page=%#v", result.Data)
	}
	if page.Records[0].ArticleID != 7 || page.Records[0].X != x || page.Records[0].Y != y || page.Records[0].LifeStage != "growing" {
		t.Fatalf("record=%+v", page.Records[0])
	}
	if repository.filter.Current != 2 || repository.filter.Size != 12 || repository.filter.Status != port.ContentProjectionReady || repository.filter.LifeStage != port.ContentLifeStageGrowing {
		t.Fatalf("repository filter=%+v", repository.filter)
	}
}

func TestContentGalaxyServiceFailsClosedWhenDisabledOrQueryIsInvalid(t *testing.T) {
	service := NewDisabledContentGalaxyService()
	if result := service.List(context.Background(), ContentGalaxyQuery{}); result.Flag || result.Message != "星河暂未公开" {
		t.Fatalf("disabled result=%+v", result)
	}
	active, err := NewContentGalaxyService(&fakeContentProjectionRepositoryForGalaxy{}, true)
	if err != nil {
		t.Fatal(err)
	}
	result := active.List(context.Background(), ContentGalaxyQuery{Size: "not-a-number"})
	if result.Flag || result.Code != 51000 {
		t.Fatalf("invalid query result=%+v", result)
	}
	if _, err := NewContentGalaxyService(nil, true); err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatal("expected missing repository error")
	}
}
