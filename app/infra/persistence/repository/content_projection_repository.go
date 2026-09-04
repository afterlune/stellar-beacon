package repository

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"xorm.io/xorm"
)

var _ port.ContentProjectionRepository = (*MyContentProjectionRepository)(nil)

type MyContentProjectionRepository struct {
	engine *xorm.Engine
}

func NewContentProjectionRepository(engine *xorm.Engine) *MyContentProjectionRepository {
	return &MyContentProjectionRepository{engine: engine}
}

type contentProjectionRow struct {
	ArticleID          int       `xorm:"article_id"`
	LifeStage          string    `xorm:"life_stage"`
	IsDeleted          bool      `xorm:"is_deleted"`
	EmbeddingModel     string    `xorm:"embedding_model"`
	EmbeddingVersion   string    `xorm:"embedding_version"`
	EmbeddingDimension int       `xorm:"embedding_dimension"`
	PCAInput           string    `xorm:"pca_input"`
	PCAInputVersion    string    `xorm:"pca_input_version"`
	ProjectionX        *float64  `xorm:"projection_x"`
	ProjectionY        *float64  `xorm:"projection_y"`
	Status             string    `xorm:"status"`
	CreatedAt          time.Time `xorm:"created_at"`
	UpdatedAt          time.Time `xorm:"updated_at"`
}

const contentProjectionColumns = `article_id, life_stage, is_deleted, embedding_model,
embedding_version, embedding_dimension, COALESCE(pca_input, '[]') AS pca_input,
COALESCE(pca_input_version, '') AS pca_input_version, projection_x, projection_y,
status, created_at, updated_at`

func (r contentProjectionRow) projection() (port.ContentProjection, error) {
	input := make([]float32, 0)
	if strings.TrimSpace(r.PCAInput) != "" {
		if err := json.Unmarshal([]byte(r.PCAInput), &input); err != nil {
			return port.ContentProjection{}, err
		}
	}
	return port.ContentProjection{
		ArticleID:          r.ArticleID,
		LifeStage:          port.ContentLifeStage(r.LifeStage),
		IsDeleted:          r.IsDeleted,
		EmbeddingModel:     r.EmbeddingModel,
		EmbeddingVersion:   r.EmbeddingVersion,
		EmbeddingDimension: r.EmbeddingDimension,
		PCAInput:           input,
		PCAInputVersion:    r.PCAInputVersion,
		ProjectionX:        r.ProjectionX,
		ProjectionY:        r.ProjectionY,
		Status:             port.ContentProjectionStatus(r.Status),
		CreatedAt:          r.CreatedAt.UTC(),
		UpdatedAt:          r.UpdatedAt.UTC(),
	}, nil
}

func (r *MyContentProjectionRepository) Upsert(ctx context.Context, input port.ContentProjection) error {
	projection, encoded, err := normalizeContentProjection(input)
	if err != nil {
		return apperrors.Invalid("content_projection.upsert", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("content_projection.upsert.database", nil)
	}
	return repoTx(r.engine, ctx, "content_projection.upsert", func(session *xorm.Session) error {
		var existing contentProjectionRow
		found, err := session.SQL("SELECT "+contentProjectionColumns+" FROM t_content_projection WHERE article_id = ? FOR UPDATE", projection.ArticleID).Get(&existing)
		if err != nil {
			return err
		}
		if found && existing.IsDeleted && !projection.IsDeleted {
			return apperrors.Conflict("content_projection.upsert", "deleted projection requires a new lifecycle event before reindexing")
		}
		_, err = session.Exec(`
INSERT INTO t_content_projection
    (article_id, life_stage, is_deleted, embedding_model, embedding_version, embedding_dimension, pca_input, pca_input_version, projection_x, projection_y, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (article_id) DO UPDATE SET
    life_stage = EXCLUDED.life_stage,
    is_deleted = EXCLUDED.is_deleted,
    embedding_model = EXCLUDED.embedding_model,
    embedding_version = EXCLUDED.embedding_version,
    embedding_dimension = EXCLUDED.embedding_dimension,
    pca_input = EXCLUDED.pca_input,
    pca_input_version = EXCLUDED.pca_input_version,
    projection_x = EXCLUDED.projection_x,
    projection_y = EXCLUDED.projection_y,
    status = EXCLUDED.status,
    updated_at = EXCLUDED.updated_at`,
			projection.ArticleID, string(projection.LifeStage), projection.IsDeleted, projection.EmbeddingModel,
			projection.EmbeddingVersion, projection.EmbeddingDimension, encoded, projection.PCAInputVersion,
			nullableProjectionFloat(projection.ProjectionX), nullableProjectionFloat(projection.ProjectionY),
			string(projection.Status), projection.CreatedAt, projection.UpdatedAt)
		return err
	})
}

func (r *MyContentProjectionRepository) MarkDeleted(ctx context.Context, articleID int, now time.Time) error {
	if articleID <= 0 {
		return apperrors.Invalid("content_projection.delete", "article id must be positive")
	}
	if r == nil {
		return apperrors.Unavailable("content_projection.delete.database", nil)
	}
	now = normalizeTime(now)
	return repoTx(r.engine, ctx, "content_projection.delete", func(session *xorm.Session) error {
		_, err := session.Exec(`
INSERT INTO t_content_projection
    (article_id, life_stage, is_deleted, embedding_model, embedding_version, embedding_dimension, pca_input, pca_input_version, projection_x, projection_y, status, created_at, updated_at)
VALUES (?, ?, TRUE, '', '', 0, '[]', '', NULL, NULL, ?, ?, ?)
ON CONFLICT (article_id) DO UPDATE SET
    life_stage = EXCLUDED.life_stage,
    is_deleted = TRUE,
    embedding_model = '',
    embedding_version = '',
    embedding_dimension = 0,
    pca_input = '[]',
    pca_input_version = '',
    projection_x = NULL,
    projection_y = NULL,
    status = ?,
    updated_at = EXCLUDED.updated_at`,
			articleID, string(port.ContentLifeStageForgotten), string(port.ContentProjectionDeleted), now, now, string(port.ContentProjectionDeleted))
		return err
	})
}

func (r *MyContentProjectionRepository) List(ctx context.Context, filter port.ContentProjectionFilter) ([]port.ContentProjection, error) {
	filter, err := normalizeContentProjectionFilter(filter)
	if err != nil {
		return nil, apperrors.Invalid("content_projection.list", err.Error())
	}
	if r == nil {
		return nil, apperrors.Unavailable("content_projection.list.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "content_projection.list")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	where := "status = ?"
	args := []any{string(filter.Status)}
	if filter.LifeStage != "" {
		where += " AND life_stage = ?"
		args = append(args, string(filter.LifeStage))
	}
	if filter.Status == port.ContentProjectionDeleted {
		where += " AND is_deleted = TRUE"
	} else {
		where += " AND is_deleted = FALSE"
	}
	if !filter.UpdatedAfter.IsZero() {
		where += " AND updated_at > ?"
		args = append(args, filter.UpdatedAfter.UTC())
	}
	args = append(args, filter.Size, (filter.Current-1)*filter.Size)
	var rows []contentProjectionRow
	if err := session.SQL("SELECT "+contentProjectionColumns+" FROM t_content_projection WHERE "+where+" ORDER BY updated_at DESC, article_id DESC LIMIT ? OFFSET ?", args...).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("content_projection.list", err)
	}
	result := make([]port.ContentProjection, 0, len(rows))
	for _, row := range rows {
		projection, err := row.projection()
		if err != nil {
			return nil, apperrors.Unavailable("content_projection.decode", err)
		}
		result = append(result, projection)
	}
	return result, nil
}

func normalizeContentProjection(input port.ContentProjection) (port.ContentProjection, string, error) {
	input.EmbeddingModel = strings.TrimSpace(input.EmbeddingModel)
	input.EmbeddingVersion = strings.TrimSpace(input.EmbeddingVersion)
	input.PCAInputVersion = strings.TrimSpace(input.PCAInputVersion)
	if input.IsDeleted {
		if input.LifeStage == "" {
			input.LifeStage = port.ContentLifeStageForgotten
		}
		if input.Status == "" {
			input.Status = port.ContentProjectionDeleted
		}
		input.EmbeddingModel = ""
		input.EmbeddingVersion = ""
		input.EmbeddingDimension = 0
		input.PCAInput = nil
		input.PCAInputVersion = ""
		input.ProjectionX = nil
		input.ProjectionY = nil
	} else if input.Status == "" {
		input.Status = port.ContentProjectionPending
	}
	if strings.ContainsAny(input.EmbeddingModel+input.EmbeddingVersion+input.PCAInputVersion, "\r\n\x00") || len(input.EmbeddingModel) > 255 || len(input.EmbeddingVersion) > 128 {
		return port.ContentProjection{}, "", stderrors.New("content projection embedding metadata is invalid")
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = time.Now().UTC()
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = input.UpdatedAt
	}
	input.CreatedAt = input.CreatedAt.UTC()
	input.UpdatedAt = input.UpdatedAt.UTC()
	if err := port.ValidateContentProjection(input); err != nil {
		return port.ContentProjection{}, "", err
	}
	encoded, err := json.Marshal(input.PCAInput)
	if err != nil {
		return port.ContentProjection{}, "", err
	}
	if input.IsDeleted {
		encoded = []byte("[]")
	}
	return input, string(encoded), nil
}

func normalizeContentProjectionFilter(filter port.ContentProjectionFilter) (port.ContentProjectionFilter, error) {
	filter.LifeStage = port.ContentLifeStage(strings.ToLower(strings.TrimSpace(string(filter.LifeStage))))
	if filter.LifeStage != "" {
		if _, err := port.NormalizeContentLifeStage(string(filter.LifeStage)); err != nil {
			return port.ContentProjectionFilter{}, err
		}
	}
	if filter.Status == "" {
		filter.Status = port.ContentProjectionReady
	}
	if _, err := port.NormalizeContentProjectionStatus(string(filter.Status)); err != nil {
		return port.ContentProjectionFilter{}, err
	}
	if filter.Current <= 0 {
		filter.Current = 1
	}
	if filter.Size <= 0 {
		filter.Size = 50
	}
	if filter.Size > 100 {
		filter.Size = 100
	}
	return filter, nil
}

func nullableProjectionFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
