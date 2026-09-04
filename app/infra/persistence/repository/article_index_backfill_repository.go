package repository

import (
	"context"
	"database/sql"
	stderrors "errors"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"xorm.io/xorm"
)

var _ port.ArticleIndexBackfillRepository = (*MyArticleIndexBackfillRepository)(nil)

type MyArticleIndexBackfillRepository struct {
	engine *xorm.Engine
}

func NewArticleIndexBackfillRepository(engine *xorm.Engine) *MyArticleIndexBackfillRepository {
	return &MyArticleIndexBackfillRepository{engine: engine}
}

type articleIndexBackfillRow struct {
	ID                 string       `xorm:"id"`
	IndexUID           string       `xorm:"index_uid"`
	IndexVersion       string       `xorm:"index_version"`
	Provider           string       `xorm:"provider"`
	Model              string       `xorm:"model"`
	ModelVersion       string       `xorm:"model_version"`
	Dimension          int          `xorm:"dimension"`
	EmbeddingBatchSize int          `xorm:"embedding_batch_size"`
	PageSize           int          `xorm:"page_size"`
	Status             string       `xorm:"status"`
	Cursor             int          `xorm:"cursor"`
	ProcessedArticles  int64        `xorm:"processed_articles"`
	IndexedChunks      int64        `xorm:"indexed_chunks"`
	PauseRequested     bool         `xorm:"pause_requested"`
	LeaseOwner         string       `xorm:"lease_owner"`
	LeaseUntil         sql.NullTime `xorm:"lease_until"`
	LastError          string       `xorm:"last_error"`
	StartedAt          sql.NullTime `xorm:"started_at"`
	CompletedAt        sql.NullTime `xorm:"completed_at"`
	CreatedAt          time.Time    `xorm:"created_at"`
	UpdatedAt          time.Time    `xorm:"updated_at"`
}

const articleIndexBackfillColumns = `id, index_uid, index_version, provider, model, model_version,
    dimension, embedding_batch_size, page_size, status, cursor, processed_articles,
    indexed_chunks, pause_requested, COALESCE(lease_owner, '') AS lease_owner,
    lease_until, COALESCE(last_error, '') AS last_error, started_at, completed_at,
    created_at, updated_at`

func (r *MyArticleIndexBackfillRepository) CreateOrGet(ctx context.Context, input port.ArticleIndexBackfillState) (port.ArticleIndexBackfillState, error) {
	state, err := normalizeArticleIndexBackfillState(input)
	if err != nil {
		return port.ArticleIndexBackfillState{}, apperrors.Invalid("ai_index_backfill.create", err.Error())
	}
	if state.Status != port.ArticleIndexBackfillPending {
		return port.ArticleIndexBackfillState{}, apperrors.Invalid("ai_index_backfill.create", "new backfill must be pending")
	}
	session, err := r.session(ctx, "ai_index_backfill.create")
	if err != nil {
		return port.ArticleIndexBackfillState{}, err
	}
	defer session.Close()
	_, err = session.Exec(`
INSERT INTO t_ai_index_backfill
    (id, index_uid, index_version, provider, model, model_version, dimension,
     embedding_batch_size, page_size, status, cursor, processed_articles,
     indexed_chunks, pause_requested, last_error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO NOTHING`,
		state.ID,
		state.IndexUID,
		state.IndexVersion,
		state.Provider,
		state.Model,
		state.ModelVersion,
		state.Dimension,
		state.EmbeddingBatchSize,
		state.PageSize,
		string(state.Status),
		state.Cursor,
		state.ProcessedArticles,
		state.IndexedChunks,
		state.PauseRequested,
		state.LastError,
		state.CreatedAt,
		state.UpdatedAt,
	)
	if err != nil {
		return port.ArticleIndexBackfillState{}, apperrors.Unavailable("ai_index_backfill.create", err)
	}
	stored, found, err := getArticleIndexBackfill(ctx, session, state.ID)
	if err != nil {
		return port.ArticleIndexBackfillState{}, err
	}
	if !found {
		return port.ArticleIndexBackfillState{}, apperrors.Unavailable("ai_index_backfill.create", stderrors.New("backfill row disappeared after insert"))
	}
	if !sameArticleIndexBackfillDefinition(stored, state) {
		return port.ArticleIndexBackfillState{}, apperrors.Conflict("ai_index_backfill.create", "backfill ID already belongs to a different index contract")
	}
	return stored, nil
}

func (r *MyArticleIndexBackfillRepository) Get(ctx context.Context, id string) (port.ArticleIndexBackfillState, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return port.ArticleIndexBackfillState{}, apperrors.Invalid("ai_index_backfill.get", "backfill ID is required")
	}
	session, err := r.session(ctx, "ai_index_backfill.get")
	if err != nil {
		return port.ArticleIndexBackfillState{}, err
	}
	defer session.Close()
	state, found, err := getArticleIndexBackfill(ctx, session, id)
	if err != nil {
		return port.ArticleIndexBackfillState{}, err
	}
	if !found {
		return port.ArticleIndexBackfillState{}, apperrors.NotFound("ai_index_backfill.get")
	}
	return state, nil
}

func (r *MyArticleIndexBackfillRepository) Claim(ctx context.Context, id, owner string, now time.Time, lease time.Duration) (port.ArticleIndexBackfillState, bool, error) {
	id = strings.TrimSpace(id)
	owner = strings.TrimSpace(owner)
	if id == "" || owner == "" {
		return port.ArticleIndexBackfillState{}, false, apperrors.Invalid("ai_index_backfill.claim", "backfill ID and lease owner are required")
	}
	if lease <= 0 {
		return port.ArticleIndexBackfillState{}, false, apperrors.Invalid("ai_index_backfill.claim", "lease must be positive")
	}
	now = normalizeBackfillTime(now)
	session, err := r.session(ctx, "ai_index_backfill.claim")
	if err != nil {
		return port.ArticleIndexBackfillState{}, false, err
	}
	defer session.Close()
	if err := session.Begin(); err != nil {
		return port.ArticleIndexBackfillState{}, false, apperrors.Unavailable("ai_index_backfill.claim.begin", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = session.Rollback()
		}
	}()

	var row articleIndexBackfillRow
	found, err := session.SQL(`
UPDATE t_ai_index_backfill
SET status = ?, lease_owner = ?, lease_until = ?,
    started_at = COALESCE(started_at, ?), updated_at = ?
WHERE id = ? AND pause_requested = FALSE AND (
    status = ? OR (status = ? AND lease_until IS NOT NULL AND lease_until <= ?)
)
RETURNING `+articleIndexBackfillColumns,
		string(port.ArticleIndexBackfillRunning),
		owner,
		now.Add(lease),
		now,
		now,
		id,
		string(port.ArticleIndexBackfillPending),
		string(port.ArticleIndexBackfillRunning),
		now,
	).Get(&row)
	if err != nil {
		return port.ArticleIndexBackfillState{}, false, apperrors.Unavailable("ai_index_backfill.claim", err)
	}
	if err := session.Commit(); err != nil {
		return port.ArticleIndexBackfillState{}, false, apperrors.Unavailable("ai_index_backfill.claim.commit", err)
	}
	committed = true
	if !found {
		return port.ArticleIndexBackfillState{}, false, nil
	}
	return row.articleIndexBackfillState(), true, nil
}

func (r *MyArticleIndexBackfillRepository) RequestPause(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return apperrors.Invalid("ai_index_backfill.request_pause", "backfill ID is required")
	}
	session, err := r.session(ctx, "ai_index_backfill.request_pause")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_ai_index_backfill
SET pause_requested = TRUE, updated_at = ?
WHERE id = ? AND status IN (?, ?, ?)`,
		time.Now().UTC(),
		id,
		string(port.ArticleIndexBackfillPending),
		string(port.ArticleIndexBackfillRunning),
		string(port.ArticleIndexBackfillPaused),
	)
	if err != nil {
		return apperrors.Unavailable("ai_index_backfill.request_pause", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("ai_index_backfill.request_pause", err)
	}
	if affected == 1 {
		return nil
	}
	return r.ensureActionableState(ctx, id, "ai_index_backfill.request_pause")
}

func (r *MyArticleIndexBackfillRepository) Resume(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return apperrors.Invalid("ai_index_backfill.resume", "backfill ID is required")
	}
	session, err := r.session(ctx, "ai_index_backfill.resume")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_ai_index_backfill
SET status = ?, pause_requested = FALSE, lease_owner = NULL, lease_until = NULL,
    last_error = '', completed_at = NULL, updated_at = ?
WHERE id = ? AND status IN (?, ?, ?)`,
		string(port.ArticleIndexBackfillPending),
		time.Now().UTC(),
		id,
		string(port.ArticleIndexBackfillPending),
		string(port.ArticleIndexBackfillPaused),
		string(port.ArticleIndexBackfillFailed),
	)
	if err != nil {
		return apperrors.Unavailable("ai_index_backfill.resume", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("ai_index_backfill.resume", err)
	}
	if affected == 1 {
		return nil
	}
	return r.ensureActionableState(ctx, id, "ai_index_backfill.resume")
}

func (r *MyArticleIndexBackfillRepository) Checkpoint(ctx context.Context, id, owner string, cursor int, processedArticles, indexedChunks int64, now time.Time, lease time.Duration) error {
	id = strings.TrimSpace(id)
	owner = strings.TrimSpace(owner)
	if id == "" || owner == "" {
		return apperrors.Invalid("ai_index_backfill.checkpoint", "backfill ID and lease owner are required")
	}
	if cursor < 0 || processedArticles < 0 || indexedChunks < 0 {
		return apperrors.Invalid("ai_index_backfill.checkpoint", "backfill progress cannot be negative")
	}
	if lease <= 0 {
		return apperrors.Invalid("ai_index_backfill.checkpoint", "lease must be positive")
	}
	now = normalizeBackfillTime(now)
	session, err := r.session(ctx, "ai_index_backfill.checkpoint")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_ai_index_backfill
SET cursor = ?, processed_articles = ?, indexed_chunks = ?,
    lease_until = ?, updated_at = ?
WHERE id = ? AND status = ? AND lease_owner = ? AND lease_until > ?
  AND cursor <= ? AND processed_articles <= ? AND indexed_chunks <= ?`,
		cursor,
		processedArticles,
		indexedChunks,
		now.Add(lease),
		now,
		id,
		string(port.ArticleIndexBackfillRunning),
		owner,
		now,
		cursor,
		processedArticles,
		indexedChunks,
	)
	if err != nil {
		return apperrors.Unavailable("ai_index_backfill.checkpoint", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("ai_index_backfill.checkpoint", err)
	}
	if affected != 1 {
		return apperrors.Conflict("ai_index_backfill.checkpoint", "backfill lease is no longer owned or progress regressed")
	}
	return nil
}

func (r *MyArticleIndexBackfillRepository) Pause(ctx context.Context, id, owner string, now time.Time) error {
	return r.transition(ctx, "ai_index_backfill.pause", id, owner, now, `
UPDATE t_ai_index_backfill
SET status = ?, pause_requested = FALSE, lease_owner = NULL, lease_until = NULL, updated_at = ?
WHERE id = ? AND status = ? AND lease_owner = ? AND lease_until > ?`,
		string(port.ArticleIndexBackfillPaused),
	)
}

func (r *MyArticleIndexBackfillRepository) Complete(ctx context.Context, id, owner string, now time.Time) error {
	now = normalizeBackfillTime(now)
	return r.transition(ctx, "ai_index_backfill.complete", id, owner, now, `
UPDATE t_ai_index_backfill
SET status = ?, pause_requested = FALSE, lease_owner = NULL, lease_until = NULL,
    last_error = '', completed_at = ?, updated_at = ?
WHERE id = ? AND status = ? AND pause_requested = FALSE AND lease_owner = ? AND lease_until > ?`,
		string(port.ArticleIndexBackfillCompleted),
		now,
	)
}

func (r *MyArticleIndexBackfillRepository) Fail(ctx context.Context, id, owner, lastError string, now time.Time) error {
	lastError = strings.TrimSpace(lastError)
	if lastError == "" {
		lastError = "backfill failed"
	}
	return r.transition(ctx, "ai_index_backfill.fail", id, owner, now, `
UPDATE t_ai_index_backfill
SET status = ?, pause_requested = FALSE, lease_owner = NULL, lease_until = NULL,
    last_error = ?, updated_at = ?
WHERE id = ? AND status = ? AND lease_owner = ? AND lease_until > ?`,
		string(port.ArticleIndexBackfillFailed),
		lastError,
	)
}

func (r *MyArticleIndexBackfillRepository) transition(ctx context.Context, operation, id, owner string, now time.Time, query string, status string, extraArgs ...interface{}) error {
	id = strings.TrimSpace(id)
	owner = strings.TrimSpace(owner)
	if id == "" || owner == "" {
		return apperrors.Invalid(operation, "backfill ID and lease owner are required")
	}
	now = normalizeBackfillTime(now)
	session, err := r.session(ctx, operation)
	if err != nil {
		return err
	}
	defer session.Close()
	args := make([]interface{}, 0, len(extraArgs)+6)
	args = append(args, status)
	args = append(args, extraArgs...)
	args = append(args, now, id, string(port.ArticleIndexBackfillRunning), owner, now)
	execArgs := make([]interface{}, 0, len(args)+1)
	execArgs = append(execArgs, query)
	execArgs = append(execArgs, args...)
	result, err := session.Exec(execArgs...)
	if err != nil {
		return apperrors.Unavailable(operation, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable(operation, err)
	}
	if affected != 1 {
		return apperrors.Conflict(operation, "backfill lease is no longer owned or has expired")
	}
	return nil
}

func (r *MyArticleIndexBackfillRepository) ensureActionableState(ctx context.Context, id, operation string) error {
	state, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	switch state.Status {
	case port.ArticleIndexBackfillCompleted:
		return apperrors.Conflict(operation, "backfill is already completed")
	case port.ArticleIndexBackfillRunning:
		return apperrors.Conflict(operation, "backfill is currently running")
	case port.ArticleIndexBackfillFailed:
		return apperrors.Conflict(operation, "backfill has failed; resume it after inspection")
	default:
		return apperrors.Conflict(operation, "backfill state could not be changed")
	}
}

func (r *MyArticleIndexBackfillRepository) session(ctx context.Context, operation string) (*xorm.Session, error) {
	if r == nil {
		return nil, apperrors.Unavailable(operation+".database", nil)
	}
	return repoSession(r.engine, ctx, operation)
}

func getArticleIndexBackfill(ctx context.Context, session *xorm.Session, id string) (port.ArticleIndexBackfillState, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var row articleIndexBackfillRow
	found, err := session.Context(ctx).SQL("SELECT "+articleIndexBackfillColumns+" FROM t_ai_index_backfill WHERE id = ?", id).Get(&row)
	if err != nil {
		return port.ArticleIndexBackfillState{}, false, apperrors.Unavailable("ai_index_backfill.get", err)
	}
	if !found {
		return port.ArticleIndexBackfillState{}, false, nil
	}
	return row.articleIndexBackfillState(), true, nil
}

func normalizeArticleIndexBackfillState(state port.ArticleIndexBackfillState) (port.ArticleIndexBackfillState, error) {
	state.ID = strings.TrimSpace(state.ID)
	state.IndexUID = strings.TrimSpace(state.IndexUID)
	state.IndexVersion = strings.TrimSpace(state.IndexVersion)
	state.Provider = strings.TrimSpace(state.Provider)
	state.Model = strings.TrimSpace(state.Model)
	state.ModelVersion = strings.TrimSpace(state.ModelVersion)
	state.LeaseOwner = strings.TrimSpace(state.LeaseOwner)
	state.LastError = strings.TrimSpace(state.LastError)
	if state.Status == "" {
		state.Status = port.ArticleIndexBackfillPending
	}
	now := time.Now().UTC()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = now
	}
	state.CreatedAt = state.CreatedAt.UTC()
	state.UpdatedAt = state.UpdatedAt.UTC()
	return state, state.Validate()
}

func sameArticleIndexBackfillDefinition(left, right port.ArticleIndexBackfillState) bool {
	return left.ID == right.ID &&
		left.IndexUID == right.IndexUID &&
		left.IndexVersion == right.IndexVersion &&
		left.Provider == right.Provider &&
		left.Model == right.Model &&
		left.ModelVersion == right.ModelVersion &&
		left.Dimension == right.Dimension &&
		left.EmbeddingBatchSize == right.EmbeddingBatchSize &&
		left.PageSize == right.PageSize
}

func normalizeBackfillTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func (r articleIndexBackfillRow) articleIndexBackfillState() port.ArticleIndexBackfillState {
	return port.ArticleIndexBackfillState{
		ID:                 r.ID,
		IndexUID:           r.IndexUID,
		IndexVersion:       r.IndexVersion,
		Provider:           r.Provider,
		Model:              r.Model,
		ModelVersion:       r.ModelVersion,
		Dimension:          r.Dimension,
		EmbeddingBatchSize: r.EmbeddingBatchSize,
		PageSize:           r.PageSize,
		Status:             port.ArticleIndexBackfillStatus(r.Status),
		Cursor:             r.Cursor,
		ProcessedArticles:  r.ProcessedArticles,
		IndexedChunks:      r.IndexedChunks,
		PauseRequested:     r.PauseRequested,
		LeaseOwner:         r.LeaseOwner,
		LeaseUntil:         nullableBackfillTime(r.LeaseUntil),
		LastError:          r.LastError,
		StartedAt:          nullableBackfillTime(r.StartedAt),
		CompletedAt:        nullableBackfillTime(r.CompletedAt),
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

func nullableBackfillTime(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}
