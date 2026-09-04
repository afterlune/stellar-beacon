package repository

import (
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestNormalizeContentProjectionKeepsEmbeddingContractAndJSONInput(t *testing.T) {
	x, y := 1.25, -0.5
	projection, encoded, err := normalizeContentProjection(port.ContentProjection{
		ArticleID:          42,
		LifeStage:          port.ContentLifeStageGrowing,
		EmbeddingModel:     " text-embedding-3-small ",
		EmbeddingVersion:   "2026-08-29",
		EmbeddingDimension: 3,
		PCAInput:           []float32{0.1, 0.2, 0.3},
		PCAInputVersion:    port.DefaultPCAInputVersion,
		ProjectionX:        &x,
		ProjectionY:        &y,
		Status:             port.ContentProjectionReady,
		UpdatedAt:          time.Date(2026, 8, 29, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if projection.EmbeddingModel != "text-embedding-3-small" || projection.CreatedAt.Location() != time.UTC || encoded != `[0.1,0.2,0.3]` {
		t.Fatalf("projection=%+v encoded=%s", projection, encoded)
	}
}

func TestNormalizeContentProjectionDeleteClearsVectorData(t *testing.T) {
	projection, encoded, err := normalizeContentProjection(port.ContentProjection{
		ArticleID: 42,
		IsDeleted: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !projection.IsDeleted || projection.LifeStage != port.ContentLifeStageForgotten || projection.Status != port.ContentProjectionDeleted || len(projection.PCAInput) != 0 || encoded != `[]` {
		t.Fatalf("deleted projection=%+v encoded=%s", projection, encoded)
	}
}

func TestNormalizeContentProjectionFilterDefaultsToReady(t *testing.T) {
	filter, err := normalizeContentProjectionFilter(port.ContentProjectionFilter{LifeStage: port.ContentLifeStageSettled, Size: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if filter.Status != port.ContentProjectionReady || filter.Current != 1 || filter.Size != 100 {
		t.Fatalf("filter=%+v", filter)
	}
	if _, err := normalizeContentProjectionFilter(port.ContentProjectionFilter{Status: "unknown"}); err == nil {
		t.Fatal("unknown projection status was accepted")
	}
}
