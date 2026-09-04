package port

import (
	"math"
	"testing"
	"time"
)

func TestCalculateContentLifeStageUsesAgeViewsAndIdleTime(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		published time.Time
		updated   time.Time
		views     int64
		want      ContentLifeStage
	}{
		{name: "newborn", published: now.Add(-24 * time.Hour), updated: now.Add(-24 * time.Hour), want: ContentLifeStageNewborn},
		{name: "growing", published: now.Add(-14 * 24 * time.Hour), updated: now.Add(-14 * 24 * time.Hour), want: ContentLifeStageGrowing},
		{name: "popular old article", published: now.Add(-120 * 24 * time.Hour), updated: now.Add(-120 * 24 * time.Hour), views: 1_000, want: ContentLifeStageGrowing},
		{name: "settled", published: now.Add(-120 * 24 * time.Hour), updated: now.Add(-120 * 24 * time.Hour), views: 10, want: ContentLifeStageSettled},
		{name: "recently updated old article", published: now.Add(-365 * 24 * time.Hour), updated: now.Add(-10 * 24 * time.Hour), want: ContentLifeStageSettled},
		{name: "forgotten", published: now.Add(-365 * 24 * time.Hour), updated: now.Add(-120 * 24 * time.Hour), views: 10, want: ContentLifeStageForgotten},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CalculateContentLifeStage(ContentLifecycleSnapshot{
				PublishedAt: test.published,
				UpdatedAt:   test.updated,
				ViewCount:   test.views,
				Now:         now,
			})
			if err != nil || got != test.want {
				t.Fatalf("stage=%q error=%v want=%q", got, err, test.want)
			}
		})
	}
}

func TestCalculateContentLifeStageRejectsNondeterministicInputs(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	for index, input := range []ContentLifecycleSnapshot{
		{Now: now},
		{PublishedAt: now, ViewCount: -1, Now: now},
		{PublishedAt: now, ViewCount: 1},
	} {
		if _, err := CalculateContentLifeStage(input); err == nil {
			t.Fatalf("case %d: invalid lifecycle input was accepted", index)
		}
	}
}

func TestValidateContentProjectionRejectsVectorsOnDelete(t *testing.T) {
	deleted := ContentProjection{
		ArticleID: 1,
		LifeStage: ContentLifeStageForgotten,
		IsDeleted: true,
		Status:    ContentProjectionDeleted,
	}
	if err := ValidateContentProjection(deleted); err != nil {
		t.Fatal(err)
	}
	deleted.PCAInput = []float32{1}
	if err := ValidateContentProjection(deleted); err == nil {
		t.Fatal("deleted projection retained PCA input")
	}

	live := ContentProjection{
		ArticleID: 1, LifeStage: ContentLifeStageGrowing, EmbeddingModel: "model", EmbeddingVersion: "v1",
		EmbeddingDimension: 2, PCAInput: []float32{0.1, 0.2}, PCAInputVersion: DefaultPCAInputVersion,
		Status: ContentProjectionReady,
	}
	if err := ValidateContentProjection(live); err != nil {
		t.Fatal(err)
	}
	live.PCAInput[0] = float32(math.NaN())
	if err := ValidateContentProjection(live); err == nil {
		t.Fatal("non-finite PCA input was accepted")
	}
}
