package port

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

// ContentLifeStage is owned by the server and is derived from source
// timestamps and counters. Models and browsers must not choose it.
type ContentLifeStage string

const (
	ContentLifeStageNewborn   ContentLifeStage = "newborn"
	ContentLifeStageGrowing   ContentLifeStage = "growing"
	ContentLifeStageSettled   ContentLifeStage = "settled"
	ContentLifeStageForgotten ContentLifeStage = "forgotten"
)

const (
	contentNewbornAge    = 7 * 24 * time.Hour
	contentGrowingAge    = 30 * 24 * time.Hour
	contentSettledAge    = 180 * 24 * time.Hour
	contentForgottenIdle = 90 * 24 * time.Hour
	contentGrowingViews  = int64(1_000)
	contentSettledViews  = int64(100)
)

type ContentLifecycleSnapshot struct {
	PublishedAt time.Time
	UpdatedAt   time.Time
	ViewCount   int64
	Now         time.Time
}

// CalculateContentLifeStage uses fixed thresholds and clamps future source
// timestamps to the current instant. A recently updated old article remains
// settled; only old, low-traffic and inactive content becomes forgotten.
func CalculateContentLifeStage(snapshot ContentLifecycleSnapshot) (ContentLifeStage, error) {
	if snapshot.PublishedAt.IsZero() {
		return "", errors.New("content published time is required")
	}
	if snapshot.Now.IsZero() {
		return "", errors.New("content calculation time is required")
	}
	if snapshot.ViewCount < 0 {
		return "", errors.New("content view count cannot be negative")
	}
	now := snapshot.Now.UTC()
	publishedAt := snapshot.PublishedAt.UTC()
	updatedAt := snapshot.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = publishedAt
	}
	age := now.Sub(publishedAt)
	if age < 0 {
		age = 0
	}
	idle := now.Sub(updatedAt)
	if idle < 0 {
		idle = 0
	}
	switch {
	case age < contentNewbornAge:
		return ContentLifeStageNewborn, nil
	case age < contentGrowingAge || (snapshot.ViewCount >= contentGrowingViews && age < contentSettledAge):
		return ContentLifeStageGrowing, nil
	case age < contentSettledAge || snapshot.ViewCount >= contentSettledViews || idle < contentForgottenIdle:
		return ContentLifeStageSettled, nil
	default:
		return ContentLifeStageForgotten, nil
	}
}

func NormalizeContentLifeStage(value string) (ContentLifeStage, error) {
	stage := ContentLifeStage(strings.ToLower(strings.TrimSpace(value)))
	switch stage {
	case ContentLifeStageNewborn, ContentLifeStageGrowing, ContentLifeStageSettled, ContentLifeStageForgotten:
		return stage, nil
	default:
		return "", errors.New("content life stage is invalid")
	}
}

const DefaultPCAInputVersion = "pca-input-v1"

type ContentProjectionStatus string

const (
	ContentProjectionPending ContentProjectionStatus = "pending"
	ContentProjectionReady   ContentProjectionStatus = "ready"
	ContentProjectionDeleted ContentProjectionStatus = "deleted"
	ContentProjectionFailed  ContentProjectionStatus = "failed"
)

func NormalizeContentProjectionStatus(value string) (ContentProjectionStatus, error) {
	status := ContentProjectionStatus(strings.ToLower(strings.TrimSpace(value)))
	switch status {
	case ContentProjectionPending, ContentProjectionReady, ContentProjectionDeleted, ContentProjectionFailed:
		return status, nil
	default:
		return "", errors.New("content projection status is invalid")
	}
}

type ContentProjection struct {
	ArticleID          int
	LifeStage          ContentLifeStage
	IsDeleted          bool
	EmbeddingModel     string
	EmbeddingVersion   string
	EmbeddingDimension int
	PCAInput           []float32
	PCAInputVersion    string
	ProjectionX        *float64
	ProjectionY        *float64
	Status             ContentProjectionStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ContentProjectionFilter struct {
	LifeStage    ContentLifeStage
	Status       ContentProjectionStatus
	UpdatedAfter time.Time
	Current      int
	Size         int
}

// ValidateContentProjection is used before persistence and before any future
// projector writes. Deleted projections intentionally have no vector or PCA
// coordinates, preserving only a tombstone for event idempotency.
func ValidateContentProjection(input ContentProjection) error {
	if input.ArticleID <= 0 {
		return errors.New("content projection article id must be positive")
	}
	if _, err := NormalizeContentLifeStage(string(input.LifeStage)); err != nil {
		return err
	}
	if _, err := NormalizeContentProjectionStatus(string(input.Status)); err != nil {
		return err
	}
	if input.IsDeleted {
		if input.Status != ContentProjectionDeleted || input.LifeStage != ContentLifeStageForgotten || len(input.PCAInput) != 0 || input.EmbeddingModel != "" || input.EmbeddingVersion != "" || input.EmbeddingDimension != 0 || input.PCAInputVersion != "" || input.ProjectionX != nil || input.ProjectionY != nil {
			return errors.New("deleted content projection must not retain vector data")
		}
		return nil
	}
	if input.Status == ContentProjectionDeleted {
		return errors.New("live content projection cannot use deleted status")
	}
	if strings.TrimSpace(input.EmbeddingModel) == "" || strings.TrimSpace(input.EmbeddingVersion) == "" || input.EmbeddingDimension <= 0 || len(input.PCAInput) != input.EmbeddingDimension {
		return errors.New("live content projection embedding contract is incomplete")
	}
	if strings.TrimSpace(input.PCAInputVersion) == "" || len(input.PCAInputVersion) > 64 {
		return errors.New("PCA input version is invalid")
	}
	for index, value := range input.PCAInput {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return errors.New("PCA input contains a non-finite value at position " + strconv.Itoa(index))
		}
	}
	if input.ProjectionX != nil && (math.IsNaN(*input.ProjectionX) || math.IsInf(*input.ProjectionX, 0)) {
		return errors.New("PCA X coordinate is invalid")
	}
	if input.ProjectionY != nil && (math.IsNaN(*input.ProjectionY) || math.IsInf(*input.ProjectionY, 0)) {
		return errors.New("PCA Y coordinate is invalid")
	}
	return nil
}

type ContentProjectionRepository interface {
	Upsert(context.Context, ContentProjection) error
	MarkDeleted(context.Context, int, time.Time) error
	List(context.Context, ContentProjectionFilter) ([]ContentProjection, error)
}
