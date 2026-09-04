package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"log/slog"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

var _ port.AIReviewRepository = (*MyAIReviewRepository)(nil)
var _ port.AIReviewPublicationRepository = (*MyAIReviewRepository)(nil)
var _ port.AIReviewPublicationClaimRepository = (*MyAIReviewRepository)(nil)

type MyAIReviewRepository struct {
	engine *xorm.Engine
}

func NewAIReviewRepository(engine *xorm.Engine) *MyAIReviewRepository {
	return &MyAIReviewRepository{engine: engine}
}

type aiReviewRow struct {
	ID                 string     `xorm:"id"`
	TargetType         string     `xorm:"target_type"`
	TargetID           string     `xorm:"target_id"`
	SessionID          string     `xorm:"session_id"`
	ContentDigest      string     `xorm:"content_digest"`
	BoundAt            *time.Time `xorm:"bound_at"`
	Operation          string     `xorm:"operation"`
	Content            string     `xorm:"content"`
	Diff               string     `xorm:"diff"`
	Status             string     `xorm:"status"`
	RunID              string     `xorm:"run_id"`
	ReviewerID         string     `xorm:"reviewer_id"`
	AgentID            string     `xorm:"agent_id"`
	PromptVersion      string     `xorm:"prompt_version"`
	SourceArticleID    int        `xorm:"source_article_id"`
	IdempotencyKey     string     `xorm:"idempotency_key"`
	RejectReason       string     `xorm:"reject_reason"`
	PublishStatus      string     `xorm:"publish_status"`
	PublishError       string     `xorm:"publish_error"`
	PublishedContentID string     `xorm:"published_content_id"`
	PublicationKey     string     `xorm:"publication_key"`
	PublishedAt        *time.Time `xorm:"published_at"`
	ExpiresAt          *time.Time `xorm:"expires_at"`
	CreatedAt          time.Time  `xorm:"created_at"`
	UpdatedAt          time.Time  `xorm:"updated_at"`
}

const aiReviewColumns = `id, target_type, target_id, operation, content, diff, status, run_id,
COALESCE(session_id, '') AS session_id, COALESCE(content_digest, '') AS content_digest, bound_at,
COALESCE(reviewer_id, '') AS reviewer_id, COALESCE(agent_id, '') AS agent_id,
COALESCE(prompt_version, '') AS prompt_version, COALESCE(source_article_id, 0) AS source_article_id,
COALESCE(idempotency_key, '') AS idempotency_key, COALESCE(reject_reason, '') AS reject_reason,
COALESCE(publish_status, 'not_attempted') AS publish_status, COALESCE(publish_error, '') AS publish_error,
COALESCE(published_content_id, '') AS published_content_id, COALESCE(publication_key, '') AS publication_key,
published_at, expires_at, created_at, updated_at`

type aiReviewActionRow struct {
	ID             string     `xorm:"id"`
	ReviewID       string     `xorm:"review_id"`
	SessionID      string     `xorm:"session_id"`
	TargetType     string     `xorm:"target_type"`
	TargetID       string     `xorm:"target_id"`
	ContentDigest  string     `xorm:"content_digest"`
	ExpiresAt      *time.Time `xorm:"expires_at"`
	Action         string     `xorm:"action"`
	ActorID        string     `xorm:"actor_id"`
	Content        string     `xorm:"content"`
	RunID          string     `xorm:"run_id"`
	IdempotencyKey string     `xorm:"idempotency_key"`
	CreatedAt      time.Time  `xorm:"created_at"`
}

const aiReviewActionColumns = `id, review_id, COALESCE(session_id, '') AS session_id,
COALESCE(target_type, '') AS target_type, COALESCE(target_id, '') AS target_id,
COALESCE(content_digest, '') AS content_digest, expires_at, action, actor_id,
content, run_id, idempotency_key, created_at`

const aiReviewInsertSQL = `
INSERT INTO t_ai_review
	(id, target_type, target_id, session_id, content_digest, bound_at, operation, content, diff, status, run_id, reviewer_id, agent_id, prompt_version, source_article_id, idempotency_key, reject_reason, publish_status, publish_error, published_content_id, publication_key, published_at, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (r *MyAIReviewRepository) Create(ctx context.Context, input port.AIReview) error {
	review, err := normalizeAIReview(input)
	if err != nil {
		return apperrors.Invalid("ai_review.create", err.Error())
	}
	var expiresAt any
	if !review.ExpiresAt.IsZero() {
		expiresAt = review.ExpiresAt
	}
	var sessionID any
	if review.SessionID != "" {
		sessionID = review.SessionID
	}
	var contentDigest any
	if review.ContentDigest != "" {
		contentDigest = review.ContentDigest
	}
	var boundAt any
	if !review.BoundAt.IsZero() {
		boundAt = review.BoundAt
	}
	var agentID any
	if review.AgentID != "" {
		agentID = review.AgentID
	}
	var promptVersion any
	if review.PromptVersion != "" {
		promptVersion = review.PromptVersion
	}
	var sourceArticleID any
	if review.SourceArticleID > 0 {
		sourceArticleID = review.SourceArticleID
	}
	var idempotencyKey any
	if review.IdempotencyKey != "" {
		idempotencyKey = review.IdempotencyKey
	}
	var publishedAt any
	if !review.PublishedAt.IsZero() {
		publishedAt = review.PublishedAt
	}
	var publicationKey any
	if review.PublicationKey != "" {
		publicationKey = review.PublicationKey
	}
	var publishError any
	if review.PublishError != "" {
		publishError = review.PublishError
	}
	var publishedContentID any
	if review.PublishedContentID != "" {
		publishedContentID = review.PublishedContentID
	}
	return repoTx(r.engine, ctx, "ai_review.create", func(session *xorm.Session) error {
		if review.IdempotencyKey != "" {
			var existing aiReviewRow
			found, err := session.SQL("SELECT "+aiReviewColumns+" FROM t_ai_review WHERE idempotency_key = ?", review.IdempotencyKey).Get(&existing)
			if err != nil {
				logAIReviewPersistenceFailure(ctx, "idempotency_lookup", err)
				return err
			}
			if found {
				if existing.ID != review.ID || existing.TargetType != review.TargetType || existing.TargetID != review.TargetID || existing.Operation != review.Operation || existing.SessionID != review.SessionID || existing.ContentDigest != review.ContentDigest {
					return apperrors.Conflict("ai_review.create", "idempotency key belongs to another review")
				}
				return nil
			}
		}
		insertSQL := aiReviewInsertSQL
		args := []any{
			review.ID,
			review.TargetType,
			review.TargetID,
			sessionID,
			contentDigest,
			boundAt,
			review.Operation,
			review.Content,
			review.Diff,
			string(review.Status),
			review.RunID,
			review.ReviewerID,
			agentID,
			promptVersion,
			sourceArticleID,
			idempotencyKey,
			review.RejectReason,
			string(review.PublishStatus),
			publishError,
			publishedContentID,
			publicationKey,
			publishedAt,
			expiresAt,
			review.CreatedAt,
			review.UpdatedAt,
		}
		if review.IdempotencyKey != "" {
			insertSQL += " ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING"
		}
		execArgs := append([]any{insertSQL}, args...)
		result, err := session.Exec(execArgs...)
		if err != nil {
			logAIReviewPersistenceFailure(ctx, "review_insert", err)
			return err
		}
		if review.IdempotencyKey != "" {
			affected, err := result.RowsAffected()
			if err != nil {
				logAIReviewPersistenceFailure(ctx, "review_rows_affected", err)
				return err
			}
			if affected == 0 {
				return nil
			}
		}
		createdAction := port.AIReviewAction{
			ID:             uuid.NewString(),
			ReviewID:       review.ID,
			SessionID:      review.SessionID,
			TargetType:     review.TargetType,
			TargetID:       review.TargetID,
			ContentDigest:  review.ContentDigest,
			ExpiresAt:      review.ExpiresAt,
			Action:         port.ReviewActionCreated,
			ActorID:        review.ReviewerID,
			RunID:          review.RunID,
			IdempotencyKey: "review:" + review.ID + ":created",
			CreatedAt:      review.CreatedAt,
		}
		if err := insertAIReviewAction(session, createdAction); err != nil {
			logAIReviewPersistenceFailure(ctx, "action_insert", err)
			return err
		}
		return nil
	})
}

func logAIReviewPersistenceFailure(ctx context.Context, phase string, err error) {
	attrs := []any{
		"phase", phase,
		"error_code", apperrors.SafeCode(err),
	}
	if sqlState := postgresSQLState(err); sqlState != "" {
		attrs = append(attrs, "sqlstate", sqlState)
	}
	slog.ErrorContext(ctx, "AI review persistence failed", attrs...)
}

func (r *MyAIReviewRepository) Get(ctx context.Context, id string) (port.AIReview, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return port.AIReview{}, apperrors.Invalid("ai_review.get", "review id is required")
	}
	session, err := r.session(ctx, "ai_review.get")
	if err != nil {
		return port.AIReview{}, err
	}
	defer session.Close()
	var row aiReviewRow
	found, err := session.SQL("SELECT "+aiReviewColumns+" FROM t_ai_review WHERE id = ?", id).Get(&row)
	if err != nil {
		return port.AIReview{}, apperrors.Unavailable("ai_review.get", err)
	}
	if !found {
		return port.AIReview{}, apperrors.NotFound("ai_review.get")
	}
	return row.aiReview(), nil
}

func (r *MyAIReviewRepository) List(ctx context.Context, filter port.ReviewFilter) ([]port.AIReview, int, error) {
	filter, err := normalizeReviewFilter(filter)
	if err != nil {
		return nil, 0, apperrors.Invalid("ai_review.list", err.Error())
	}
	session, err := r.session(ctx, "ai_review.list")
	if err != nil {
		return nil, 0, err
	}
	defer session.Close()
	where, args := reviewFilterSQL(filter)
	var countRow struct {
		Count int `xorm:"count"`
	}
	found, err := session.SQL("SELECT COUNT(1) AS count FROM t_ai_review WHERE "+where, args...).Get(&countRow)
	if err != nil {
		return nil, 0, apperrors.Unavailable("ai_review.list.count", err)
	}
	if !found {
		return []port.AIReview{}, 0, nil
	}
	args = append(args, filter.Size, (filter.Current-1)*filter.Size)
	var rows []aiReviewRow
	if err := session.SQL("SELECT "+aiReviewColumns+" FROM t_ai_review WHERE "+where+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?", args...).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("ai_review.list", err)
	}
	reviews := make([]port.AIReview, 0, len(rows))
	for _, row := range rows {
		reviews = append(reviews, row.aiReview())
	}
	return reviews, countRow.Count, nil
}

func (r *MyAIReviewRepository) ApplyAction(ctx context.Context, reviewID string, input port.AIReviewAction) error {
	reviewID = strings.TrimSpace(reviewID)
	if reviewID == "" {
		return apperrors.Invalid("ai_review.action", "review id is required")
	}
	action, err := normalizeAIReviewAction(input, reviewID)
	if err != nil {
		return apperrors.Invalid("ai_review.action", err.Error())
	}
	return repoTx(r.engine, ctx, "ai_review.action", func(session *xorm.Session) error {
		var existing aiReviewActionRow
		exists, err := session.SQL("SELECT "+aiReviewActionColumns+" FROM t_ai_review_action WHERE idempotency_key = ?", action.IdempotencyKey).Get(&existing)
		if err != nil {
			return err
		}
		if exists {
			if existing.ReviewID == reviewID && existing.Action == string(action.Action) &&
				actionBindingRowsMatch(existing, action) {
				return nil
			}
			return apperrors.Conflict("ai_review.action", "idempotency key belongs to another action")
		}

		var review aiReviewRow
		found, err := session.SQL("SELECT "+aiReviewColumns+" FROM t_ai_review WHERE id = ? FOR UPDATE", reviewID).Get(&review)
		if err != nil {
			return err
		}
		if !found {
			return apperrors.NotFound("ai_review.action")
		}
		if err := validateReviewActionBinding(review, action, time.Now().UTC(), "ai_review.action"); err != nil {
			return err
		}
		if review.Status != string(port.ReviewPending) {
			return apperrors.Conflict("ai_review.action", "review is no longer pending")
		}

		newStatus := string(port.ReviewPending)
		switch action.Action {
		case port.ReviewActionAccepted:
			newStatus = string(port.ReviewApproved)
		case port.ReviewActionPartiallyAccepted:
			newStatus = string(port.ReviewPartiallyApproved)
		case port.ReviewActionRejected:
			newStatus = string(port.ReviewRejected)
		case port.ReviewActionRegenerated:
		default:
			return apperrors.Invalid("ai_review.action", "created action is only allowed when creating a review")
		}
		if _, err := session.Exec(`
UPDATE t_ai_review
SET status = ?, reviewer_id = ?, reject_reason = CASE WHEN ? = ? THEN ? ELSE reject_reason END, updated_at = ?
WHERE id = ? AND status = ?`,
			newStatus,
			action.ActorID,
			string(action.Action),
			string(port.ReviewActionRejected),
			action.Content,
			action.CreatedAt,
			reviewID,
			string(port.ReviewPending),
		); err != nil {
			return err
		}
		return insertAIReviewAction(session, action)
	})
}

const publicationClaimTimeout = 10 * time.Minute

// ClaimPublication serializes the only unsafe part of an approval: the
// external write. The review remains pending while its publication status is
// processing, so FinalizePublication can atomically turn the claim into the
// final audited state. A stale claim can be retried after the timeout to
// recover from a process crash; callers should keep the external operation
// bounded well below that timeout.
func (r *MyAIReviewRepository) ClaimPublication(ctx context.Context, reviewID string, input port.AIReviewAction) (bool, error) {
	reviewID = strings.TrimSpace(reviewID)
	if reviewID == "" {
		return false, apperrors.Invalid("ai_review.claim_publication", "review id is required")
	}
	if input.Action != port.ReviewActionAccepted && input.Action != port.ReviewActionPartiallyAccepted {
		return false, apperrors.Invalid("ai_review.claim_publication", "publication requires an acceptance action")
	}
	action, err := normalizeAIReviewAction(input, reviewID)
	if err != nil {
		return false, apperrors.Invalid("ai_review.claim_publication", err.Error())
	}
	now := time.Now().UTC()
	returnValue := false
	err = repoTx(r.engine, ctx, "ai_review.claim_publication", func(session *xorm.Session) error {
		var review aiReviewRow
		found, err := session.SQL("SELECT "+aiReviewColumns+" FROM t_ai_review WHERE id = ? FOR UPDATE", reviewID).Get(&review)
		if err != nil {
			return err
		}
		if !found {
			return apperrors.NotFound("ai_review.claim_publication")
		}
		if review.Status != string(port.ReviewPending) {
			if review.PublicationKey == action.IdempotencyKey &&
				(review.PublishStatus == string(port.ReviewPublishSucceeded) || review.PublishStatus == string(port.ReviewPublishSkipped)) {
				return nil
			}
			return apperrors.Conflict("ai_review.claim_publication", "review is no longer pending")
		}
		if err := validateReviewActionBinding(review, action, now, "ai_review.claim_publication"); err != nil {
			return err
		}
		if review.PublishStatus == string(port.ReviewPublishProcessing) {
			if review.PublicationKey == action.IdempotencyKey && now.Sub(review.UpdatedAt.UTC()) < publicationClaimTimeout {
				return nil
			}
			if now.Sub(review.UpdatedAt.UTC()) < publicationClaimTimeout {
				return apperrors.Conflict("ai_review.claim_publication", "publication is already processing")
			}
		}
		if review.PublishStatus != "" && review.PublishStatus != string(port.ReviewPublishNotAttempted) &&
			review.PublishStatus != string(port.ReviewPublishFailed) && review.PublishStatus != string(port.ReviewPublishProcessing) {
			return apperrors.Conflict("ai_review.claim_publication", "review publication is already finalized")
		}
		result, err := session.Exec(`
UPDATE t_ai_review
SET publish_status = ?, publish_error = '', publication_key = ?, updated_at = ?
WHERE id = ? AND status = ? AND publish_status IN (?, ?, ?)`,
			string(port.ReviewPublishProcessing), action.IdempotencyKey, now, reviewID, string(port.ReviewPending),
			string(port.ReviewPublishNotAttempted), string(port.ReviewPublishFailed), string(port.ReviewPublishProcessing))
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return apperrors.Conflict("ai_review.claim_publication", "review publication claim was lost")
		}
		returnValue = true
		return nil
	})
	return returnValue, err
}

func (r *MyAIReviewRepository) Approve(ctx context.Context, id, reviewerID string) error {
	return r.updateStatus(ctx, "ai_review.approve", id, reviewerID, port.ReviewApproved, "")
}

func (r *MyAIReviewRepository) Reject(ctx context.Context, id, reviewerID, reason string) error {
	return r.updateStatus(ctx, "ai_review.reject", id, reviewerID, port.ReviewRejected, reason)
}

func (r *MyAIReviewRepository) Expire(ctx context.Context, now time.Time) (int, error) {
	now = normalizeTime(now)
	session, err := r.session(ctx, "ai_review.expire")
	if err != nil {
		return 0, err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_ai_review
SET status = ?, updated_at = ?
WHERE status = ? AND COALESCE(publish_status, 'not_attempted') <> ? AND expires_at IS NOT NULL AND expires_at <= ?`,
		string(port.ReviewExpired), now, string(port.ReviewPending), string(port.ReviewPublishProcessing), now)
	if err != nil {
		return 0, apperrors.Unavailable("ai_review.expire", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Unavailable("ai_review.expire.rows", err)
	}
	return int(affected), nil
}

func (r *MyAIReviewRepository) ExpireOne(ctx context.Context, id, actorID string, now time.Time) error {
	id = strings.TrimSpace(id)
	actorID = strings.TrimSpace(actorID)
	if id == "" || actorID == "" {
		return apperrors.Invalid("ai_review.expire_one", "review id and actor id are required")
	}
	now = normalizeTime(now)
	return repoTx(r.engine, ctx, "ai_review.expire_one", func(session *xorm.Session) error {
		var review aiReviewRow
		found, err := session.SQL("SELECT "+aiReviewColumns+" FROM t_ai_review WHERE id = ? FOR UPDATE", id).Get(&review)
		if err != nil {
			return err
		}
		if !found {
			return apperrors.NotFound("ai_review.expire_one")
		}
		if review.Status == string(port.ReviewExpired) {
			return nil
		}
		if review.Status != string(port.ReviewPending) {
			return apperrors.Conflict("ai_review.expire_one", "review is no longer pending")
		}
		if review.PublishStatus == string(port.ReviewPublishProcessing) {
			return apperrors.Conflict("ai_review.expire_one", "publication is still processing")
		}
		if review.ExpiresAt == nil || review.ExpiresAt.After(now) {
			return apperrors.Conflict("ai_review.expire_one", "review has not expired")
		}
		if _, err := session.Exec(`
UPDATE t_ai_review SET status = ?, updated_at = ? WHERE id = ? AND status = ?`,
			string(port.ReviewExpired), now, id, string(port.ReviewPending)); err != nil {
			return err
		}
		return insertAIReviewAction(session, port.AIReviewAction{
			ID:             uuid.NewString(),
			ReviewID:       id,
			Action:         port.ReviewActionExpired,
			ActorID:        actorID,
			IdempotencyKey: "review:" + id + ":expired",
			CreatedAt:      now,
		})
	})
}

func (r *MyAIReviewRepository) RecordPublicationFailure(ctx context.Context, reviewID string, input port.AIReviewAction, reason string) error {
	reviewID = strings.TrimSpace(reviewID)
	reason = strings.TrimSpace(reason)
	if reviewID == "" || reason == "" {
		return apperrors.Invalid("ai_review.publish_failure", "review id and failure reason are required")
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return apperrors.Invalid("ai_review.publish_failure", "actor id is required")
	}
	publicationKey := strings.TrimSpace(input.IdempotencyKey)
	if publicationKey == "" {
		publicationKey = "review:" + reviewID + ":publish"
	}
	input.ReviewID = reviewID
	input.Action = port.ReviewActionPublishFailed
	input.IdempotencyKey = compactReviewActionKey(reviewID, "publish_failed", publicationKey)
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	input.Content = reason
	action, err := normalizeAIReviewAction(input, reviewID)
	if err != nil {
		return apperrors.Invalid("ai_review.publish_failure", err.Error())
	}
	return repoTx(r.engine, ctx, "ai_review.publish_failure", func(session *xorm.Session) error {
		var review aiReviewRow
		found, err := session.SQL("SELECT "+aiReviewColumns+" FROM t_ai_review WHERE id = ? FOR UPDATE", reviewID).Get(&review)
		if err != nil {
			return err
		}
		if !found {
			return apperrors.NotFound("ai_review.publish_failure")
		}
		if review.Status != string(port.ReviewPending) {
			if review.PublishStatus == string(port.ReviewPublishFailed) && review.PublicationKey == publicationKey {
				return nil
			}
			return apperrors.Conflict("ai_review.publish_failure", "review is no longer pending")
		}
		if review.PublishStatus != string(port.ReviewPublishProcessing) || review.PublicationKey != publicationKey {
			return apperrors.Conflict("ai_review.publish_failure", "publication claim is not owned by this attempt")
		}
		if err := validateReviewActionIdentity(review, action, "ai_review.publish_failure"); err != nil {
			return err
		}
		if _, err := session.Exec(`
UPDATE t_ai_review
SET publish_status = ?, publish_error = ?, publication_key = ?, updated_at = ?
WHERE id = ? AND status = ?`,
			string(port.ReviewPublishFailed), reason, publicationKey, action.CreatedAt, reviewID, string(port.ReviewPending)); err != nil {
			return err
		}
		return insertAIReviewAction(session, action)
	})
}

func (r *MyAIReviewRepository) FinalizePublication(ctx context.Context, reviewID string, input port.AIReviewAction, publication port.AIReviewPublication) error {
	reviewID = strings.TrimSpace(reviewID)
	if reviewID == "" {
		return apperrors.Invalid("ai_review.finalize_publication", "review id is required")
	}
	if input.Action != port.ReviewActionAccepted && input.Action != port.ReviewActionPartiallyAccepted {
		return apperrors.Invalid("ai_review.finalize_publication", "publication requires an acceptance action")
	}
	status, err := port.NormalizeReviewPublishStatus(string(publication.Status))
	if err != nil || (status != port.ReviewPublishSucceeded && status != port.ReviewPublishSkipped) {
		return apperrors.Invalid("ai_review.finalize_publication", "publication status is invalid")
	}
	if strings.TrimSpace(publication.PublicationKey) == "" || strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.IdempotencyKey) == "" {
		return apperrors.Invalid("ai_review.finalize_publication", "publication key is required")
	}
	input.ReviewID = reviewID
	action, err := normalizeAIReviewAction(input, reviewID)
	if err != nil {
		return apperrors.Invalid("ai_review.finalize_publication", err.Error())
	}
	publication.PublishedAt = normalizeTime(publication.PublishedAt)
	return repoTx(r.engine, ctx, "ai_review.finalize_publication", func(session *xorm.Session) error {
		var review aiReviewRow
		found, err := session.SQL("SELECT "+aiReviewColumns+" FROM t_ai_review WHERE id = ? FOR UPDATE", reviewID).Get(&review)
		if err != nil {
			return err
		}
		if !found {
			return apperrors.NotFound("ai_review.finalize_publication")
		}
		if review.Status != string(port.ReviewPending) {
			if review.PublicationKey == publication.PublicationKey && review.PublishStatus == string(publication.Status) {
				return nil
			}
			return apperrors.Conflict("ai_review.finalize_publication", "review is no longer pending")
		}
		if review.PublishStatus != string(port.ReviewPublishProcessing) || review.PublicationKey != strings.TrimSpace(publication.PublicationKey) {
			return apperrors.Conflict("ai_review.finalize_publication", "publication claim is not owned by this attempt")
		}
		if err := validateReviewActionIdentity(review, action, "ai_review.finalize_publication"); err != nil {
			return err
		}
		newStatus := port.ReviewApproved
		if action.Action == port.ReviewActionPartiallyAccepted {
			newStatus = port.ReviewPartiallyApproved
		}
		if _, err := session.Exec(`
UPDATE t_ai_review
SET status = ?, reviewer_id = ?, publish_status = ?, publish_error = '',
    published_content_id = ?, publication_key = ?, published_at = ?, updated_at = ?
WHERE id = ? AND status = ?`,
			string(newStatus), action.ActorID, string(publication.Status), strings.TrimSpace(publication.ContentID),
			publication.PublicationKey, publication.PublishedAt, action.CreatedAt, reviewID, string(port.ReviewPending)); err != nil {
			return err
		}
		if err := insertAIReviewAction(session, action); err != nil {
			return err
		}
		return insertAIReviewAction(session, port.AIReviewAction{
			ID:             uuid.NewString(),
			ReviewID:       reviewID,
			Action:         port.ReviewActionPublished,
			SessionID:      action.SessionID,
			TargetType:     action.TargetType,
			TargetID:       action.TargetID,
			ContentDigest:  action.ContentDigest,
			ExpiresAt:      action.ExpiresAt,
			ActorID:        action.ActorID,
			Content:        strings.TrimSpace(publication.ContentID),
			RunID:          action.RunID,
			IdempotencyKey: compactReviewActionKey(reviewID, "published", publication.PublicationKey),
			CreatedAt:      action.CreatedAt,
		})
	})
}

func compactReviewActionKey(reviewID, action, identity string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(identity)))
	return "review:" + strings.TrimSpace(reviewID) + ":" + strings.TrimSpace(action) + ":" + hex.EncodeToString(digest[:])[:16]
}

func (r *MyAIReviewRepository) updateStatus(ctx context.Context, operation, id, reviewerID string, status port.ReviewStatus, reason string) error {
	id = strings.TrimSpace(id)
	reviewerID = strings.TrimSpace(reviewerID)
	if id == "" || reviewerID == "" {
		return apperrors.Invalid(operation, "review id and reviewer id are required")
	}
	session, err := r.session(ctx, operation)
	if err != nil {
		return err
	}
	defer session.Close()
	now := time.Now().UTC()
	result, err := session.Exec(`
UPDATE t_ai_review
SET status = ?, reviewer_id = ?, reject_reason = ?, updated_at = ?
WHERE id = ? AND status = ?`,
		string(status), reviewerID, strings.TrimSpace(reason), now, id, string(port.ReviewPending))
	if err != nil {
		return apperrors.Unavailable(operation, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable(operation+".rows", err)
	}
	if affected != 1 {
		return apperrors.Conflict(operation, "review is not pending")
	}
	return nil
}

func (r *MyAIReviewRepository) session(ctx context.Context, operation string) (*xorm.Session, error) {
	if r == nil {
		return nil, apperrors.Unavailable(operation+".database", nil)
	}
	return repoSession(r.engine, ctx, operation)
}

func normalizeAIReview(input port.AIReview) (port.AIReview, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.TargetType = strings.TrimSpace(input.TargetType)
	input.TargetID = strings.TrimSpace(input.TargetID)
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.ContentDigest = strings.ToLower(strings.TrimSpace(input.ContentDigest))
	input.Operation = strings.TrimSpace(input.Operation)
	input.Content = strings.TrimSpace(input.Content)
	input.RunID = strings.TrimSpace(input.RunID)
	input.ReviewerID = strings.TrimSpace(input.ReviewerID)
	input.AgentID = strings.TrimSpace(input.AgentID)
	input.PromptVersion = strings.TrimSpace(input.PromptVersion)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.PublishError = strings.TrimSpace(input.PublishError)
	input.PublishedContentID = strings.TrimSpace(input.PublishedContentID)
	input.PublicationKey = strings.TrimSpace(input.PublicationKey)
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if len(input.ID) > 64 || input.TargetType == "" || len(input.TargetType) > 64 || input.TargetID == "" || len(input.TargetID) > 128 || len(input.SessionID) > 128 || strings.ContainsAny(input.SessionID, "\r\n\x00") {
		return port.AIReview{}, stderrors.New("review identity is invalid")
	}
	if input.SourceArticleID < 0 || len(input.AgentID) > 64 || len(input.PromptVersion) > 128 || len(input.IdempotencyKey) > 255 || len(input.PublishError) > 255 || len(input.PublishedContentID) > 128 || len(input.PublicationKey) > 255 {
		return port.AIReview{}, stderrors.New("review behavior metadata is invalid")
	}
	if input.Operation == "" || len(input.Operation) > 32 || input.Content == "" || input.RunID == "" || input.ReviewerID == "" {
		return port.AIReview{}, stderrors.New("review content is incomplete")
	}
	if len([]rune(input.Content)) > 100_000 || len([]rune(input.Diff)) > 200_000 {
		return port.AIReview{}, stderrors.New("review content is too long")
	}
	if input.ContentDigest == "" {
		input.ContentDigest = port.ReviewContentDigest(input.Content)
	}
	if !validReviewDigest(input.ContentDigest) {
		return port.AIReview{}, stderrors.New("review content digest is invalid")
	}
	if input.Status == "" {
		input.Status = port.ReviewPending
	}
	if input.Status != port.ReviewPending {
		return port.AIReview{}, stderrors.New("new review must be pending")
	}
	if input.PublishStatus == "" {
		input.PublishStatus = port.ReviewPublishNotAttempted
	}
	normalizedPublishStatus, err := port.NormalizeReviewPublishStatus(string(input.PublishStatus))
	if err != nil {
		return port.AIReview{}, err
	}
	if normalizedPublishStatus != port.ReviewPublishNotAttempted {
		return port.AIReview{}, stderrors.New("new review publication status must be not_attempted")
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	if input.BoundAt.IsZero() {
		input.BoundAt = input.CreatedAt
	}
	input.CreatedAt = input.CreatedAt.UTC()
	input.UpdatedAt = input.UpdatedAt.UTC()
	input.BoundAt = input.BoundAt.UTC()
	if !input.ExpiresAt.IsZero() {
		input.ExpiresAt = input.ExpiresAt.UTC()
	}
	if !input.PublishedAt.IsZero() {
		input.PublishedAt = input.PublishedAt.UTC()
	}
	return input, nil
}

func normalizeAIReviewAction(input port.AIReviewAction, reviewID string) (port.AIReviewAction, error) {
	action, err := port.NormalizeReviewAction(string(input.Action))
	if err != nil {
		return port.AIReviewAction{}, err
	}
	input.Action = action
	input.ReviewID = reviewID
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.TargetType = strings.TrimSpace(input.TargetType)
	input.TargetID = strings.TrimSpace(input.TargetID)
	input.ContentDigest = strings.ToLower(strings.TrimSpace(input.ContentDigest))
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.Content = strings.TrimSpace(input.Content)
	input.RunID = strings.TrimSpace(input.RunID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if input.ActorID == "" || input.IdempotencyKey == "" {
		return port.AIReviewAction{}, stderrors.New("review action actor and idempotency key are required")
	}
	if len(input.ID) > 64 || len(input.IdempotencyKey) > 255 || len(input.ActorID) > 64 || len(input.RunID) > 128 || len([]rune(input.Content)) > 100_000 || len(input.SessionID) > 128 || len(input.TargetType) > 64 || len(input.TargetID) > 128 || strings.ContainsAny(input.SessionID, "\r\n\x00") {
		return port.AIReviewAction{}, stderrors.New("review action value is too long")
	}
	if input.ContentDigest != "" && !validReviewDigest(input.ContentDigest) {
		return port.AIReviewAction{}, stderrors.New("review action content digest is invalid")
	}
	if action == port.ReviewActionCreated {
		return port.AIReviewAction{}, stderrors.New("created action is reserved for review creation")
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	input.CreatedAt = input.CreatedAt.UTC()
	if !input.ExpiresAt.IsZero() {
		input.ExpiresAt = input.ExpiresAt.UTC()
	}
	return input, nil
}

func validReviewDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func actionBindingRowsMatch(existing aiReviewActionRow, action port.AIReviewAction) bool {
	if existing.SessionID != action.SessionID || existing.TargetType != action.TargetType ||
		existing.TargetID != action.TargetID || existing.ContentDigest != action.ContentDigest {
		return false
	}
	if existing.ExpiresAt == nil {
		return action.ExpiresAt.IsZero()
	}
	return !action.ExpiresAt.IsZero() && existing.ExpiresAt.UTC().Equal(action.ExpiresAt.UTC())
}

func validateReviewActionIdentity(review aiReviewRow, action port.AIReviewAction, operation string) error {
	hasBinding := action.SessionID != "" || action.TargetType != "" || action.TargetID != "" || action.ContentDigest != "" || !action.ExpiresAt.IsZero()
	if !hasBinding {
		// Preserve compatibility for actions created before migration 0008. All
		// new application paths populate the binding fields below.
		return nil
	}
	if action.TargetType == "" || action.TargetID == "" || action.TargetType != review.TargetType || action.TargetID != review.TargetID {
		return apperrors.Conflict(operation, "review target binding does not match")
	}
	if review.ContentDigest != "" && action.ContentDigest != review.ContentDigest {
		return apperrors.Conflict(operation, "review content binding does not match")
	}
	if review.SessionID != "" && action.SessionID != review.SessionID {
		return apperrors.Conflict(operation, "review session binding does not match")
	}
	if review.ExpiresAt != nil && !action.ExpiresAt.IsZero() && !review.ExpiresAt.UTC().Equal(action.ExpiresAt.UTC()) {
		return apperrors.Conflict(operation, "review expiry binding does not match")
	}
	return nil
}

func validateReviewActionBinding(review aiReviewRow, action port.AIReviewAction, now time.Time, operation string) error {
	if err := validateReviewActionIdentity(review, action, operation); err != nil {
		return err
	}
	if review.ExpiresAt != nil && !now.UTC().Before(review.ExpiresAt.UTC()) {
		return apperrors.Conflict(operation, "review has expired")
	}
	return nil
}

func insertAIReviewAction(session *xorm.Session, action port.AIReviewAction) error {
	_, err := session.Exec(`
INSERT INTO t_ai_review_action
    (id, review_id, session_id, target_type, target_id, content_digest, expires_at, action, actor_id, content, run_id, idempotency_key, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (idempotency_key) DO NOTHING`,
		action.ID,
		action.ReviewID,
		nullableReviewString(action.SessionID),
		nullableReviewString(action.TargetType),
		nullableReviewString(action.TargetID),
		nullableReviewString(action.ContentDigest),
		nullableReviewTime(action.ExpiresAt),
		string(action.Action),
		action.ActorID,
		action.Content,
		action.RunID,
		action.IdempotencyKey,
		action.CreatedAt,
	)
	return err
}

func nullableReviewString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableReviewTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}

func normalizeReviewFilter(filter port.ReviewFilter) (port.ReviewFilter, error) {
	if filter.Current <= 0 {
		filter.Current = 1
	}
	if filter.Size <= 0 {
		filter.Size = 10
	}
	if filter.Size > 100 {
		filter.Size = 100
	}
	if filter.Status != "" {
		switch filter.Status {
		case port.ReviewPending, port.ReviewApproved, port.ReviewPartiallyApproved, port.ReviewRejected, port.ReviewExpired:
		default:
			return port.ReviewFilter{}, stderrors.New("review status is invalid")
		}
	}
	filter.TargetType = strings.TrimSpace(filter.TargetType)
	if len(filter.TargetType) > 64 {
		return port.ReviewFilter{}, stderrors.New("review target type is too long")
	}
	filter.TargetID = strings.TrimSpace(filter.TargetID)
	if len(filter.TargetID) > 128 {
		return port.ReviewFilter{}, stderrors.New("review target id is too long")
	}
	filter.Operation = strings.TrimSpace(filter.Operation)
	if len(filter.Operation) > 32 {
		return port.ReviewFilter{}, stderrors.New("review operation is too long")
	}
	filter.AgentID = strings.TrimSpace(filter.AgentID)
	if len(filter.AgentID) > 64 {
		return port.ReviewFilter{}, stderrors.New("review agent id is too long")
	}
	if !filter.From.IsZero() {
		filter.From = filter.From.UTC()
	}
	if !filter.To.IsZero() {
		filter.To = filter.To.UTC()
	}
	if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
		return port.ReviewFilter{}, stderrors.New("review time range is invalid")
	}
	return filter, nil
}

func reviewFilterSQL(filter port.ReviewFilter) (string, []any) {
	where := "1 = 1"
	args := make([]any, 0, 6)
	if filter.Status != "" {
		where += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	if filter.TargetType != "" {
		where += " AND target_type = ?"
		args = append(args, filter.TargetType)
	}
	if filter.TargetID != "" {
		where += " AND target_id = ?"
		args = append(args, filter.TargetID)
	}
	if filter.Operation != "" {
		where += " AND operation = ?"
		args = append(args, filter.Operation)
	}
	if filter.AgentID != "" {
		where += " AND agent_id = ?"
		args = append(args, filter.AgentID)
	}
	if !filter.From.IsZero() {
		where += " AND created_at >= ?"
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		where += " AND created_at < ?"
		args = append(args, filter.To)
	}
	return where, args
}

func (r aiReviewRow) aiReview() port.AIReview {
	review := port.AIReview{
		ID:                 r.ID,
		TargetType:         r.TargetType,
		TargetID:           r.TargetID,
		SessionID:          r.SessionID,
		ContentDigest:      r.ContentDigest,
		Operation:          r.Operation,
		Content:            r.Content,
		Diff:               r.Diff,
		Status:             port.ReviewStatus(r.Status),
		RunID:              r.RunID,
		ReviewerID:         r.ReviewerID,
		AgentID:            r.AgentID,
		PromptVersion:      r.PromptVersion,
		SourceArticleID:    r.SourceArticleID,
		IdempotencyKey:     r.IdempotencyKey,
		RejectReason:       r.RejectReason,
		PublishStatus:      port.ReviewPublishStatus(r.PublishStatus),
		PublishError:       r.PublishError,
		PublishedContentID: r.PublishedContentID,
		PublicationKey:     r.PublicationKey,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
	if r.BoundAt != nil {
		review.BoundAt = r.BoundAt.UTC()
	}
	if r.ExpiresAt != nil {
		review.ExpiresAt = r.ExpiresAt.UTC()
	}
	if r.PublishedAt != nil {
		review.PublishedAt = r.PublishedAt.UTC()
	}
	return review
}
