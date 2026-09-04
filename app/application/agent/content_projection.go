package agent

import (
	"context"
	"errors"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

// ContentProjectionBatchProcessor calculates one deterministic PCA snapshot
// and persists only the coordinate fields. It is deliberately independent of
// a scheduler or database implementation; the infrastructure worker owns the
// polling lifecycle.
type ContentProjectionBatchProcessor struct {
	repository port.ContentProjectionRepository
	now        func() time.Time
}

func NewContentProjectionBatchProcessor(repository port.ContentProjectionRepository) (*ContentProjectionBatchProcessor, error) {
	if repository == nil {
		return nil, apperrors.Invalid("agent.content_projection.dependencies", "content projection repository is required")
	}
	return &ContentProjectionBatchProcessor{
		repository: repository,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

// ProcessPending computes coordinates for the supplied pending rows. The
// caller must provide a single batch from the repository; keeping the batch
// boundary explicit makes future queue/checkpoint implementations possible.
func (p *ContentProjectionBatchProcessor) ProcessPending(ctx context.Context, projections []port.ContentProjection) (int, error) {
	if p == nil || p.repository == nil {
		return 0, apperrors.Unavailable("agent.content_projection.process", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	pending := make([]port.ContentProjection, 0, len(projections))
	inputs := make([]port.PCAInputVector, 0, len(projections))
	for _, projection := range projections {
		if projection.Status != port.ContentProjectionPending || projection.IsDeleted {
			continue
		}
		if err := port.ValidateContentProjection(projection); err != nil {
			return 0, apperrors.Invalid("agent.content_projection.input", err.Error())
		}
		pending = append(pending, projection)
		inputs = append(inputs, port.PCAInputVector{
			ArticleID: projection.ArticleID,
			Values:    append([]float32(nil), projection.PCAInput...),
		})
	}
	if len(pending) == 0 {
		return 0, nil
	}
	points, err := port.CalculateContentPCA2D(inputs)
	if err != nil {
		return 0, apperrors.Invalid("agent.content_projection.pca", err.Error())
	}
	byArticleID := make(map[int]port.ContentProjectionPoint, len(points))
	for _, point := range points {
		byArticleID[point.ArticleID] = point
	}
	now := p.now()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	for _, projection := range pending {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		point, ok := byArticleID[projection.ArticleID]
		if !ok {
			return 0, errors.New("PCA result is missing an article")
		}
		x, y := point.X, point.Y
		projection.ProjectionX = &x
		projection.ProjectionY = &y
		projection.Status = port.ContentProjectionReady
		projection.UpdatedAt = now
		if err := p.repository.Upsert(ctx, projection); err != nil {
			return 0, err
		}
	}
	return len(pending), nil
}
