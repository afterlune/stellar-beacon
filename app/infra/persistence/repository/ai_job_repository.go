package repository

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

var _ port.AIJobRepository = (*MyAIJobRepository)(nil)

type MyAIJobRepository struct {
	engine *xorm.Engine
}

func NewAIJobRepository(engine *xorm.Engine) *MyAIJobRepository {
	return &MyAIJobRepository{engine: engine}
}

type aiJobRow struct {
	ID             string    `xorm:"id"`
	Kind           string    `xorm:"kind"`
	IdempotencyKey string    `xorm:"idempotency_key"`
	Payload        string    `xorm:"payload"`
	Status         string    `xorm:"status"`
	Attempts       int       `xorm:"attempts"`
	MaxAttempts    int       `xorm:"max_attempts"`
	RunAfter       time.Time `xorm:"run_after"`
	LastError      string    `xorm:"last_error"`
	CreatedAt      time.Time `xorm:"created_at"`
	UpdatedAt      time.Time `xorm:"updated_at"`
}

// aiJobReturningColumns is used by an UPDATE ... FROM query where both the
// target table and the candidate CTE expose an id column. Qualifying every
// returned column avoids PostgreSQL's ambiguous-column error before a job is
// even available to claim.
const aiJobReturningColumns = `job.id, job.kind, job.idempotency_key, job.payload, job.status, job.attempts, job.max_attempts, job.run_after, COALESCE(job.last_error, '') AS last_error, job.created_at, job.updated_at`

func (r *MyAIJobRepository) Enqueue(ctx context.Context, input port.AIJob) error {
	job, err := normalizeAIJob(input)
	if err != nil {
		return apperrors.Invalid("ai_job.enqueue", err.Error())
	}
	session, err := r.session(ctx, "ai_job.enqueue")
	if err != nil {
		return err
	}
	defer session.Close()
	_, err = session.Exec(`
INSERT INTO t_ai_job
    (id, kind, idempotency_key, payload, status, attempts, max_attempts, run_after, last_error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (idempotency_key) DO NOTHING`,
		job.ID,
		job.Kind,
		job.IdempotencyKey,
		string(job.Payload),
		string(job.Status),
		job.Attempts,
		job.MaxAttempts,
		job.RunAfter,
		job.LastError,
		job.CreatedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return apperrors.Unavailable("ai_job.enqueue", err)
	}
	return nil
}

func (r *MyAIJobRepository) Claim(ctx context.Context, workerID string, now time.Time, lease time.Duration) (port.AIJob, bool, error) {
	return r.claim(ctx, workerID, "", now, lease)
}

func (r *MyAIJobRepository) ClaimKind(ctx context.Context, workerID, kind string, now time.Time, lease time.Duration) (port.AIJob, bool, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return port.AIJob{}, false, apperrors.Invalid("ai_job.claim_kind", "job kind is required")
	}
	return r.claim(ctx, workerID, kind, now, lease)
}

func (r *MyAIJobRepository) claim(ctx context.Context, workerID, kind string, now time.Time, lease time.Duration) (port.AIJob, bool, error) {
	if strings.TrimSpace(workerID) == "" {
		return port.AIJob{}, false, apperrors.Invalid("ai_job.claim", "worker id is required")
	}
	if lease <= 0 {
		return port.AIJob{}, false, apperrors.Invalid("ai_job.claim", "lease must be positive")
	}
	now = normalizeTime(now)
	session, err := r.session(ctx, "ai_job.claim")
	if err != nil {
		return port.AIJob{}, false, err
	}
	defer session.Close()
	if err := session.Begin(); err != nil {
		return port.AIJob{}, false, apperrors.Unavailable("ai_job.claim.begin", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = session.Rollback()
		}
	}()

	if _, err := session.Exec(`
UPDATE t_ai_job
SET status = ?, lease_owner = NULL, lease_until = NULL,
    last_error = COALESCE(last_error, ?), updated_at = ?
WHERE status = ? AND lease_until IS NOT NULL AND lease_until <= ? AND attempts >= max_attempts`,
		string(port.AIJobDead),
		"lease expired after maximum attempts",
		now,
		string(port.AIJobRunning),
		now,
	); err != nil {
		return port.AIJob{}, false, apperrors.Unavailable("ai_job.claim.recover", err)
	}

	leaseUntil := now.Add(lease)
	var row aiJobRow
	candidateWhere := `attempts < max_attempts
      AND (
          (status = ? AND run_after <= ?)
          OR (status = ? AND lease_until IS NOT NULL AND lease_until <= ?)
      )`
	candidateArgs := []any{
		string(port.AIJobPending),
		now,
		string(port.AIJobRunning),
		now,
	}
	if kind != "" {
		candidateWhere = "kind = ? AND " + candidateWhere
		candidateArgs = append([]any{kind}, candidateArgs...)
	}
	query := `
WITH candidate AS (
    SELECT id
    FROM t_ai_job
    WHERE ` + candidateWhere + `
    ORDER BY run_after ASC, created_at ASC, id ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE t_ai_job AS job
SET status = ?, attempts = job.attempts + 1,
    lease_owner = ?, lease_until = ?, run_after = ?, updated_at = ?
FROM candidate
WHERE job.id = candidate.id
RETURNING ` + aiJobReturningColumns
	claimArgs := append(candidateArgs,
		string(port.AIJobRunning),
		workerID,
		leaseUntil,
		now,
		now,
	)
	found, err := session.SQL(query, claimArgs...).Get(&row)
	if err != nil {
		return port.AIJob{}, false, apperrors.Unavailable("ai_job.claim", err)
	}
	if err := session.Commit(); err != nil {
		return port.AIJob{}, false, apperrors.Unavailable("ai_job.claim.commit", err)
	}
	committed = true
	if !found {
		return port.AIJob{}, false, nil
	}
	return row.aiJob(), true, nil
}

func (r *MyAIJobRepository) Complete(ctx context.Context, jobID, workerID string, result port.AIJobResult) error {
	return r.transition(ctx, "ai_job.complete", jobID, workerID, func(session *xorm.Session, now time.Time) (int64, error) {
		resultInfo, err := session.Exec(`
UPDATE t_ai_job
SET status = ?, lease_owner = NULL, lease_until = NULL, last_error = NULL,
    result_run_id = ?, result_payload = ?, updated_at = ?
WHERE id = ? AND status = ? AND lease_owner = ? AND lease_until > ?`,
			string(port.AIJobSucceeded),
			result.RunID,
			string(result.Payload),
			now,
			jobID,
			string(port.AIJobRunning),
			workerID,
			now,
		)
		if err != nil {
			return 0, err
		}
		return resultInfo.RowsAffected()
	})
}

func (r *MyAIJobRepository) Retry(ctx context.Context, jobID, workerID string, runAfter time.Time, lastError string) error {
	if runAfter.IsZero() {
		runAfter = time.Now().UTC()
	}
	return r.transition(ctx, "ai_job.retry", jobID, workerID, func(session *xorm.Session, now time.Time) (int64, error) {
		result, err := session.Exec(`
UPDATE t_ai_job
SET status = ?, lease_owner = NULL, lease_until = NULL,
    run_after = ?, last_error = ?, updated_at = ?
WHERE id = ? AND status = ? AND lease_owner = ? AND lease_until > ? AND attempts < max_attempts`,
			string(port.AIJobPending),
			runAfter.UTC(),
			strings.TrimSpace(lastError),
			now,
			jobID,
			string(port.AIJobRunning),
			workerID,
			now,
		)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	})
}

func (r *MyAIJobRepository) DeadLetter(ctx context.Context, jobID, workerID, lastError string) error {
	return r.transition(ctx, "ai_job.dead_letter", jobID, workerID, func(session *xorm.Session, now time.Time) (int64, error) {
		result, err := session.Exec(`
UPDATE t_ai_job
SET status = ?, lease_owner = NULL, lease_until = NULL,
    last_error = ?, updated_at = ?
WHERE id = ? AND status = ? AND lease_owner = ? AND lease_until > ?`,
			string(port.AIJobDead),
			strings.TrimSpace(lastError),
			now,
			jobID,
			string(port.AIJobRunning),
			workerID,
			now,
		)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	})
}

func (r *MyAIJobRepository) transition(ctx context.Context, operation, jobID, workerID string, fn func(*xorm.Session, time.Time) (int64, error)) error {
	if strings.TrimSpace(jobID) == "" {
		return apperrors.Invalid(operation, "job id is required")
	}
	if strings.TrimSpace(workerID) == "" {
		return apperrors.Invalid(operation, "worker id is required")
	}
	session, err := r.session(ctx, operation)
	if err != nil {
		return err
	}
	defer session.Close()
	now := time.Now().UTC()
	affected, err := fn(session, now)
	if err != nil {
		return apperrors.Unavailable(operation, err)
	}
	if affected != 1 {
		return apperrors.Conflict(operation, "job is not owned by the active worker")
	}
	return nil
}

func (r *MyAIJobRepository) session(ctx context.Context, operation string) (*xorm.Session, error) {
	if r == nil {
		return nil, apperrors.Unavailable(operation+".database", nil)
	}
	return repoSession(r.engine, ctx, operation)
}

func normalizeAIJob(job port.AIJob) (port.AIJob, error) {
	job.ID = strings.TrimSpace(job.ID)
	job.Kind = strings.TrimSpace(job.Kind)
	job.IdempotencyKey = strings.TrimSpace(job.IdempotencyKey)
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	if len(job.ID) > 64 {
		return port.AIJob{}, stderrors.New("job id is too long")
	}
	if job.Kind == "" {
		return port.AIJob{}, stderrors.New("job kind is required")
	}
	if job.IdempotencyKey == "" {
		return port.AIJob{}, stderrors.New("job idempotency key is required")
	}
	if len(job.IdempotencyKey) > 255 {
		return port.AIJob{}, stderrors.New("job idempotency key is too long")
	}
	if len(job.Payload) == 0 {
		return port.AIJob{}, stderrors.New("job payload is required")
	}
	if job.Status == "" {
		job.Status = port.AIJobPending
	}
	if job.Status != port.AIJobPending {
		return port.AIJob{}, stderrors.New("enqueued job must be pending")
	}
	if job.Attempts < 0 {
		return port.AIJob{}, stderrors.New("job attempts cannot be negative")
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = port.DefaultAIJobMaxAttempts
	}
	if job.Attempts > job.MaxAttempts {
		return port.AIJob{}, stderrors.New("job attempts exceed maximum")
	}
	now := time.Now().UTC()
	if job.RunAfter.IsZero() {
		job.RunAfter = now
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}
	job.RunAfter = job.RunAfter.UTC()
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	job.Payload = append([]byte(nil), job.Payload...)
	return job, nil
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func (r aiJobRow) aiJob() port.AIJob {
	job := port.AIJob{
		ID:             r.ID,
		Kind:           r.Kind,
		IdempotencyKey: r.IdempotencyKey,
		Payload:        []byte(r.Payload),
		Status:         port.AIJobStatus(r.Status),
		Attempts:       r.Attempts,
		MaxAttempts:    r.MaxAttempts,
		RunAfter:       r.RunAfter,
		LastError:      r.LastError,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
	return job
}
