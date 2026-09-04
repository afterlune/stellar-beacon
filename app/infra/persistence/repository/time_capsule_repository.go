package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	stderrors "errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

var _ port.TimeCapsuleRepository = (*MyTimeCapsuleRepository)(nil)

type MyTimeCapsuleRepository struct {
	engine *xorm.Engine
}

func NewTimeCapsuleRepository(engine *xorm.Engine) *MyTimeCapsuleRepository {
	return &MyTimeCapsuleRepository{engine: engine}
}

type timeCapsuleRow struct {
	ID                string    `xorm:"id"`
	OwnerUserID       int       `xorm:"owner_user_id"`
	Title             string    `xorm:"title"`
	Content           string    `xorm:"content"`
	DeliverAt         time.Time `xorm:"deliver_at"`
	Status            string    `xorm:"status"`
	SealedAt          time.Time `xorm:"sealed_at"`
	DeliveredAt       time.Time `xorm:"delivered_at"`
	DeliveryAttempts  int       `xorm:"delivery_attempts"`
	LastDeliveryError string    `xorm:"last_delivery_error"`
	CreatedAt         time.Time `xorm:"created_at"`
	UpdatedAt         time.Time `xorm:"updated_at"`
}

const timeCapsuleColumns = `id, owner_user_id, title, content, deliver_at, status,
COALESCE(sealed_at, 'epoch'::timestamptz) AS sealed_at,
COALESCE(delivered_at, 'epoch'::timestamptz) AS delivered_at,
delivery_attempts, COALESCE(last_delivery_error, '') AS last_delivery_error,
created_at, updated_at`

func (r timeCapsuleRow) capsule() port.TimeCapsule {
	return port.TimeCapsule{
		ID:                r.ID,
		OwnerUserID:       r.OwnerUserID,
		Title:             r.Title,
		Content:           r.Content,
		DeliverAt:         r.DeliverAt.UTC(),
		Status:            port.TimeCapsuleStatus(r.Status),
		SealedAt:          postgresEpochToZero(r.SealedAt),
		DeliveredAt:       postgresEpochToZero(r.DeliveredAt),
		DeliveryAttempts:  r.DeliveryAttempts,
		LastDeliveryError: r.LastDeliveryError,
		CreatedAt:         r.CreatedAt.UTC(),
		UpdatedAt:         r.UpdatedAt.UTC(),
	}
}

func (r *MyTimeCapsuleRepository) Create(ctx context.Context, input port.TimeCapsule) error {
	capsule, err := normalizeTimeCapsule(input)
	if err != nil {
		return apperrors.Invalid("time_capsule.create", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("time_capsule.create.database", nil)
	}
	if capsule.ID == "" {
		capsule.ID = uuid.NewString()
	}
	if capsule.Status != port.TimeCapsuleDraft {
		return apperrors.Invalid("time_capsule.create", "new time capsule must be a draft")
	}
	return repoTx(r.engine, ctx, "time_capsule.create", func(session *xorm.Session) error {
		_, err := session.Exec(`
INSERT INTO t_time_capsule
    (id, owner_user_id, title, content, deliver_at, status, sealed_at, delivered_at,
     delivery_attempts, last_delivery_error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?, ?, ?)`,
			capsule.ID, capsule.OwnerUserID, capsule.Title, capsule.Content, capsule.DeliverAt,
			string(capsule.Status), capsule.DeliveryAttempts, capsule.LastDeliveryError,
			capsule.CreatedAt, capsule.UpdatedAt)
		if err != nil {
			if isUniqueViolation(err) {
				return apperrors.Conflict("time_capsule.create", "time capsule id already exists")
			}
			return err
		}
		return nil
	})
}

func (r *MyTimeCapsuleRepository) GetOwned(ctx context.Context, id string, ownerUserID int, now time.Time) (port.TimeCapsule, error) {
	if err := validateTimeCapsuleLookup(id, ownerUserID); err != nil {
		return port.TimeCapsule{}, apperrors.Invalid("time_capsule.get", err.Error())
	}
	if r == nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.get.database", nil)
	}
	now = normalizeTime(now)
	session, err := repoSession(r.engine, ctx, "time_capsule.get")
	if err != nil {
		return port.TimeCapsule{}, err
	}
	defer session.Close()
	if err := session.Begin(); err != nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.get.begin", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = session.Rollback()
		}
	}()
	// Materialize the due transition on read as well as in the worker. This
	// keeps delivery correct when the optional worker is paused or disabled.
	if _, err := session.Exec(`
UPDATE t_time_capsule
SET status = ?, updated_at = ?
WHERE id = ? AND owner_user_id = ? AND status = ? AND deliver_at <= ?`,
		string(port.TimeCapsuleDue), now, id, ownerUserID, string(port.TimeCapsuleSealed), now); err != nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.get.advance", err)
	}
	var row timeCapsuleRow
	found, err := session.SQL("SELECT "+timeCapsuleColumns+" FROM t_time_capsule WHERE id = ? AND owner_user_id = ?", id, ownerUserID).Get(&row)
	if err != nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.get", err)
	}
	if err := session.Commit(); err != nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.get.commit", err)
	}
	committed = true
	if !found {
		return port.TimeCapsule{}, apperrors.NotFound("time_capsule.get")
	}
	return row.capsule(), nil
}

func (r *MyTimeCapsuleRepository) Seal(ctx context.Context, id string, ownerUserID int, now time.Time) error {
	if err := validateTimeCapsuleLookup(id, ownerUserID); err != nil {
		return apperrors.Invalid("time_capsule.seal", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("time_capsule.seal.database", nil)
	}
	now = normalizeTime(now)
	return repoTx(r.engine, ctx, "time_capsule.seal", func(session *xorm.Session) error {
		var row timeCapsuleRow
		found, err := session.SQL("SELECT "+timeCapsuleColumns+" FROM t_time_capsule WHERE id = ? AND owner_user_id = ? FOR UPDATE", id, ownerUserID).Get(&row)
		if err != nil {
			return err
		}
		if !found {
			return apperrors.NotFound("time_capsule.seal")
		}
		capsule := row.capsule()
		if capsule.Status != port.TimeCapsuleDraft {
			return apperrors.Conflict("time_capsule.seal", "time capsule is already sealed or delivered")
		}
		if !capsule.DeliverAt.After(now) {
			return apperrors.Conflict("time_capsule.seal", "delivery time has passed")
		}
		_, err = session.Exec(`
UPDATE t_time_capsule
SET status = ?, sealed_at = ?, updated_at = ?
WHERE id = ? AND owner_user_id = ? AND status = ?`,
			string(port.TimeCapsuleSealed), now, now, id, ownerUserID, string(port.TimeCapsuleDraft))
		return err
	})
}

func (r *MyTimeCapsuleRepository) AdvanceDue(ctx context.Context, now time.Time, limit int) (int, error) {
	if r == nil {
		return 0, apperrors.Unavailable("time_capsule.advance_due.database", nil)
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > port.MaxTimeCapsulePageSize {
		limit = port.MaxTimeCapsulePageSize
	}
	now = normalizeTime(now)
	session, err := repoSession(r.engine, ctx, "time_capsule.advance_due")
	if err != nil {
		return 0, err
	}
	defer session.Close()
	result, err := session.Exec(`
WITH due_capsules AS (
    SELECT id
    FROM t_time_capsule
    WHERE status = ? AND deliver_at <= ?
    ORDER BY deliver_at ASC, id ASC
    FOR UPDATE SKIP LOCKED
    LIMIT ?
)
UPDATE t_time_capsule AS capsule
SET status = ?, updated_at = ?
FROM due_capsules
WHERE capsule.id = due_capsules.id AND capsule.status = ?`,
		string(port.TimeCapsuleSealed), now, limit, string(port.TimeCapsuleDue), now, string(port.TimeCapsuleSealed))
	if err != nil {
		return 0, apperrors.Unavailable("time_capsule.advance_due", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Unavailable("time_capsule.advance_due.rows", err)
	}
	return int(affected), nil
}

func (r *MyTimeCapsuleRepository) Deliver(ctx context.Context, id string, ownerUserID int, now time.Time) (port.TimeCapsule, error) {
	if err := validateTimeCapsuleLookup(id, ownerUserID); err != nil {
		return port.TimeCapsule{}, apperrors.Invalid("time_capsule.deliver", err.Error())
	}
	if r == nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.deliver.database", nil)
	}
	now = normalizeTime(now)
	session, err := repoSession(r.engine, ctx, "time_capsule.deliver")
	if err != nil {
		return port.TimeCapsule{}, err
	}
	defer session.Close()
	if err := session.Begin(); err != nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.deliver.begin", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = session.Rollback()
		}
	}()
	var row timeCapsuleRow
	found, err := session.SQL("SELECT "+timeCapsuleColumns+" FROM t_time_capsule WHERE id = ? AND owner_user_id = ? FOR UPDATE", id, ownerUserID).Get(&row)
	if err != nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.deliver", err)
	}
	if !found {
		return port.TimeCapsule{}, apperrors.NotFound("time_capsule.deliver")
	}
	capsule := row.capsule()
	if capsule.Status == port.TimeCapsuleDelivered {
		if err := session.Commit(); err != nil {
			return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.deliver.commit", err)
		}
		committed = true
		return capsule, nil
	}
	if capsule.Status == port.TimeCapsuleSealed && !capsule.DeliverAt.After(now) {
		capsule.Status = port.TimeCapsuleDue
		if _, err := session.Exec("UPDATE t_time_capsule SET status = ?, updated_at = ? WHERE id = ? AND owner_user_id = ? AND status = ?", string(port.TimeCapsuleDue), now, id, ownerUserID, string(port.TimeCapsuleSealed)); err != nil {
			return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.deliver.advance", err)
		}
	}
	if capsule.Status != port.TimeCapsuleDue {
		return port.TimeCapsule{}, apperrors.Conflict("time_capsule.deliver", "time capsule is not due")
	}
	if _, err := session.Exec(`
UPDATE t_time_capsule
SET status = ?, delivered_at = ?, delivery_attempts = delivery_attempts + 1,
    last_delivery_error = '', updated_at = ?
WHERE id = ? AND owner_user_id = ? AND status = ?`,
		string(port.TimeCapsuleDelivered), now, now, id, ownerUserID, string(port.TimeCapsuleDue)); err != nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.deliver.update", err)
	}
	if err := session.Commit(); err != nil {
		return port.TimeCapsule{}, apperrors.Unavailable("time_capsule.deliver.commit", err)
	}
	committed = true
	capsule.Status = port.TimeCapsuleDelivered
	capsule.DeliveredAt = now
	capsule.DeliveryAttempts++
	capsule.LastDeliveryError = ""
	capsule.UpdatedAt = now
	return capsule, nil
}

func (r *MyTimeCapsuleRepository) MarkDeliveryFailed(ctx context.Context, id string, ownerUserID int, reason string, now time.Time) error {
	if err := validateTimeCapsuleLookup(id, ownerUserID); err != nil {
		return apperrors.Invalid("time_capsule.delivery_failed", err.Error())
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len([]rune(reason)) > port.MaxTimeCapsuleErrorRunes || strings.ContainsAny(reason, "\r\n\x00") {
		return apperrors.Invalid("time_capsule.delivery_failed", "delivery failure reason is invalid")
	}
	if r == nil {
		return apperrors.Unavailable("time_capsule.delivery_failed.database", nil)
	}
	now = normalizeTime(now)
	session, err := repoSession(r.engine, ctx, "time_capsule.delivery_failed")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_time_capsule
SET status = ?, last_delivery_error = ?, delivery_attempts = delivery_attempts + 1, updated_at = ?
WHERE id = ? AND owner_user_id = ? AND status = ?`,
		string(port.TimeCapsuleDeliveryFailed), reason, now, id, ownerUserID, string(port.TimeCapsuleDue))
	if err != nil {
		return apperrors.Unavailable("time_capsule.delivery_failed", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("time_capsule.delivery_failed.rows", err)
	}
	if affected != 1 {
		return apperrors.Conflict("time_capsule.delivery_failed", "time capsule is not due or is not owned")
	}
	return nil
}

func (r *MyTimeCapsuleRepository) RetryDelivery(ctx context.Context, id string, ownerUserID int, now time.Time) error {
	if err := validateTimeCapsuleLookup(id, ownerUserID); err != nil {
		return apperrors.Invalid("time_capsule.retry", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("time_capsule.retry.database", nil)
	}
	now = normalizeTime(now)
	session, err := repoSession(r.engine, ctx, "time_capsule.retry")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_time_capsule
SET status = ?, last_delivery_error = '', updated_at = ?
WHERE id = ? AND owner_user_id = ? AND status = ?`,
		string(port.TimeCapsuleDue), now, id, ownerUserID, string(port.TimeCapsuleDeliveryFailed))
	if err != nil {
		return apperrors.Unavailable("time_capsule.retry", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("time_capsule.retry.rows", err)
	}
	if affected != 1 {
		return apperrors.Conflict("time_capsule.retry", "time capsule is not awaiting retry")
	}
	return nil
}

func normalizeTimeCapsule(input port.TimeCapsule) (port.TimeCapsule, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Status == "" {
		input.Status = port.TimeCapsuleDraft
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	input.DeliverAt = input.DeliverAt.UTC()
	input.CreatedAt = input.CreatedAt.UTC()
	input.UpdatedAt = input.UpdatedAt.UTC()
	if err := port.ValidateTimeCapsule(input); err != nil {
		return port.TimeCapsule{}, err
	}
	return input, nil
}

func validateTimeCapsuleLookup(id string, ownerUserID int) error {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > port.MaxTimeCapsuleIDLength || strings.ContainsAny(id, "\r\n\x00") {
		return stderrors.New("time capsule id is invalid")
	}
	return port.ValidateTimeCapsuleIdentity(ownerUserID)
}
