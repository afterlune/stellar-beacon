package repository

import (
	"context"
	"strings"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/query"

	"xorm.io/xorm"
)

var validReportReasons = map[string]bool{
	"spam": true, "harassment": true, "porn": true, "illegal": true, "privacy": true, "other": true,
}

// CreateCommentReport records one reader report. The partial unique index keeps
// a reader from stacking pending reports on the same comment.
func (c *MyCommentRepo) CreateCommentReport(ctx context.Context, commentID, collectionID, reporterID int, reason, detail string) error {
	if commentID <= 0 || collectionID <= 0 || reporterID <= 0 {
		return apperrors.Invalid("comment.report.create", "invalid target")
	}
	reason = strings.TrimSpace(reason)
	if !validReportReasons[reason] {
		return apperrors.Invalid("comment.report.create", "unsupported reason")
	}
	detail = strings.TrimSpace(detail)
	if len([]rune(detail)) > 200 {
		return apperrors.Invalid("comment.report.create", "detail too long")
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		comment, err := moderationTarget(session, 0, collectionID, commentID, false)
		if err != nil {
			return err
		}
		if comment.UserId == reporterID {
			return apperrors.Invalid("comment.report.create", "cannot report your own comment")
		}
		result, err := session.Exec(`INSERT INTO t_comment_report (comment_id, collection_id, reporter_id, reason, detail)
			SELECT ?, ?, ?, ?, ? WHERE NOT EXISTS (
				SELECT 1 FROM t_comment_report WHERE comment_id = ? AND reporter_id = ? AND status = 'pending')`,
			commentID, collectionID, reporterID, reason, detail, commentID, reporterID)
		if err != nil {
			return apperrors.Unavailable("comment.report.create", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return apperrors.Conflict("comment.report.create", "report already pending")
		}
		return writeModerationLog(session, commentID, collectionID, reporterID, "reader", "report", nil, nil, nil, "", reason)
	})
}

const commentReportSelect = `SELECT r.comment_id,
		r.collection_id,
		COALESCE(collection.title, '') AS collection_title,
		COALESCE(comment.comment_content, '') AS comment_content,
		COALESCE(author.nickname, '') AS comment_nickname,
		COALESCE(comment.is_delete, 0) AS comment_is_delete,
		count(1) AS report_count,
		COALESCE(string_agg(DISTINCT r.reason, ','), '') AS reasons,
		COALESCE(max(r.detail), '') AS latest_detail,
		max(r.create_time) AS latest_create_time
	FROM t_comment_report r
	LEFT JOIN t_comment comment ON comment.id = r.comment_id
	LEFT JOIN t_collection collection ON collection.id = r.collection_id
	LEFT JOIN t_user_info author ON author.id = comment.user_id`

const commentReportGroupBy = ` GROUP BY r.comment_id, r.collection_id, collection.title, comment.comment_content, author.nickname, comment.is_delete
	ORDER BY max(r.create_time) DESC, r.comment_id DESC LIMIT ? OFFSET ?`

func listCommentReportGroups(ctx context.Context, repo *MyCommentRepo, userID int, collectionID int, current, size int) ([]*port.CommentReportGroup, int, error) {
	session, err := repo.commentSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	where := " WHERE r.status = 'pending'"
	args := []interface{}{}
	if userID > 0 {
		where += " AND collection.user_id = ?"
		args = append(args, userID)
	}
	if collectionID > 0 {
		where += " AND r.collection_id = ?"
		args = append(args, collectionID)
	}
	var total int
	if _, err := session.SQL("SELECT count(1) FROM (SELECT r.comment_id FROM t_comment_report r LEFT JOIN t_collection collection ON collection.id = r.collection_id"+where+" GROUP BY r.comment_id) grouped", args...).Get(&total); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.report.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	var rows []*port.CommentReportGroup
	if err := session.SQL(commentReportSelect+where+commentReportGroupBy, listArgs...).Find(&rows); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.report.list", err)
	}
	return rows, total, nil
}

func (c *MyCommentRepo) ListOwnerCommentReports(ctx context.Context, ownerID, collectionID, current, size int) ([]*port.CommentReportGroup, int, error) {
	if ownerID <= 0 {
		return nil, 0, apperrors.Invalid("comment.report.list", "invalid owner")
	}
	return listCommentReportGroups(ctx, c, ownerID, collectionID, current, size)
}

func (c *MyCommentRepo) ListAdminCommentReports(ctx context.Context, current, size int) ([]*port.CommentReportGroup, int, error) {
	return listCommentReportGroups(ctx, c, 0, 0, current, size)
}

// ResolveCommentReports closes every pending report on one comment and applies
// the matching moderation action in the same transaction. It returns the
// reporters that were notified so the caller can raise moderation notices.
func (c *MyCommentRepo) ResolveCommentReports(ctx context.Context, actorID int, actorRole string, collectionID, commentID int, decision, reason string) ([]int, error) {
	if actorID <= 0 || collectionID <= 0 || commentID <= 0 {
		return nil, apperrors.Invalid("comment.report.resolve", "invalid target")
	}
	ownerID := actorID
	if actorRole == "admin" {
		ownerID = 0
	}
	decision = strings.TrimSpace(decision)
	reporters := []int{}
	if err := ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		if _, err := moderationTarget(session, ownerID, collectionID, commentID, true); err != nil {
			return err
		}
		status := "dismissed"
		switch decision {
		case "dismiss":
		case "hide":
			status = "resolved"
			if err := applyCommentModeration(session, actorID, actorRole, actionDelete, ownerID, collectionID, commentID, "", reason); err != nil {
				// A comment that is already hidden still resolves its reports.
				if !apperrors.IsKind(err, apperrors.KindValidation) {
					return err
				}
			}
		case "restore":
			status = "resolved"
			if err := applyCommentModeration(session, actorID, actorRole, actionRestore, ownerID, collectionID, commentID, "", reason); err != nil {
				if !apperrors.IsKind(err, apperrors.KindValidation) {
					return err
				}
			}
		default:
			return apperrors.Invalid("comment.report.resolve", "unsupported decision")
		}
		if err := session.SQL(`SELECT DISTINCT reporter_id FROM t_comment_report WHERE comment_id = ? AND status = 'pending'`, commentID).Find(&reporters); err != nil {
			return apperrors.Unavailable("comment.report.reporters", err)
		}
		if _, err := session.Exec(`UPDATE t_comment_report
			SET status = ?, handled_by = ?, handled_at = CURRENT_TIMESTAMP, decision_reason = ?
			WHERE comment_id = ? AND status = 'pending'`, status, actorID, reason, commentID); err != nil {
			return apperrors.Unavailable("comment.report.resolve", err)
		}
		action := "report_dismiss"
		if status == "resolved" {
			action = "report_hide"
			if decision == "restore" {
				action = "restore"
			}
		}
		return writeModerationLog(session, commentID, collectionID, actorID, actorRole, action, nil, nil, nil, "", reason)
	}); err != nil {
		return nil, err
	}
	return reporters, nil
}

// CreateCommentAppeal lets the author of a hidden comment ask for a review. Only
// the comment author may appeal, and only while the comment stays hidden.
func (c *MyCommentRepo) CreateCommentAppeal(ctx context.Context, appellantID, commentID int, reason string) error {
	if appellantID <= 0 || commentID <= 0 {
		return apperrors.Invalid("comment.appeal.create", "invalid target")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len([]rune(reason)) > 500 {
		return apperrors.Invalid("comment.appeal.create", "invalid reason")
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		var comment entity.TComment
		found, err := session.ID(commentID).Get(&comment)
		if err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "comment.appeal.lookup", err)
		}
		if !found || comment.Type != 6 || comment.UserId != appellantID {
			return apperrors.NotFound("comment.appeal.create")
		}
		if comment.IsDelete == 0 {
			return apperrors.Invalid("comment.appeal.create", "comment is visible")
		}
		result, err := session.Exec(`INSERT INTO t_comment_appeal (comment_id, collection_id, appellant_id, reason)
			SELECT ?, ?, ?, ? WHERE NOT EXISTS (
				SELECT 1 FROM t_comment_appeal WHERE comment_id = ? AND appellant_id = ? AND status = 'pending')`,
			commentID, comment.TopicId, appellantID, reason, commentID, appellantID)
		if err != nil {
			return apperrors.Unavailable("comment.appeal.create", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return apperrors.Conflict("comment.appeal.create", "appeal already pending")
		}
		return writeModerationLog(session, commentID, comment.TopicId, appellantID, "commenter", "appeal_submit", nil, nil, nil, "", reason)
	})
}

const commentAppealSelect = `SELECT appeal.id,
		appeal.comment_id,
		appeal.collection_id,
		COALESCE(collection.title, '') AS collection_title,
		COALESCE(comment.comment_content, '') AS comment_content,
		appeal.appellant_id,
		COALESCE(appellant.nickname, '') AS appellant_name,
		appeal.reason,
		appeal.stage,
		appeal.status,
		appeal.decision_reason,
		appeal.create_time
	FROM t_comment_appeal appeal
	LEFT JOIN t_comment comment ON comment.id = appeal.comment_id
	LEFT JOIN t_collection collection ON collection.id = appeal.collection_id
	LEFT JOIN t_user_info appellant ON appellant.id = appeal.appellant_id`

func listCommentAppeals(ctx context.Context, repo *MyCommentRepo, userID, appellantID, collectionID, current, size int, onlyPending bool, stage string) ([]*port.CommentAppealItem, int, error) {
	session, err := repo.commentSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	where := " WHERE 1 = 1"
	args := []interface{}{}
	if onlyPending {
		if stage != "" {
			where += " AND appeal.stage = ?"
			args = append(args, stage)
		}
		where += " AND appeal.status = 'pending'"
	}
	if userID > 0 {
		where += " AND collection.user_id = ?"
		args = append(args, userID)
	}
	if appellantID > 0 {
		where += " AND appeal.appellant_id = ?"
		args = append(args, appellantID)
	}
	if collectionID > 0 {
		where += " AND appeal.collection_id = ?"
		args = append(args, collectionID)
	}
	var total int
	if _, err := session.SQL("SELECT count(1) FROM t_comment_appeal appeal LEFT JOIN t_collection collection ON collection.id = appeal.collection_id"+where, args...).Get(&total); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.appeal.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	var rows []*port.CommentAppealItem
	if err := session.SQL(commentAppealSelect+where+" ORDER BY appeal.create_time DESC, appeal.id DESC LIMIT ? OFFSET ?", listArgs...).Find(&rows); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.appeal.list", err)
	}
	return rows, total, nil
}

func (c *MyCommentRepo) ListOwnerCommentAppeals(ctx context.Context, ownerID, collectionID, current, size int) ([]*port.CommentAppealItem, int, error) {
	if ownerID <= 0 {
		return nil, 0, apperrors.Invalid("comment.appeal.list", "invalid owner")
	}
	return listCommentAppeals(ctx, c, ownerID, 0, collectionID, current, size, true, "owner")
}

func (c *MyCommentRepo) ListAdminCommentAppeals(ctx context.Context, current, size int) ([]*port.CommentAppealItem, int, error) {
	return listCommentAppeals(ctx, c, 0, 0, 0, current, size, true, "admin")
}

func (c *MyCommentRepo) ListMyCommentAppeals(ctx context.Context, appellantID, current, size int) ([]*port.CommentAppealItem, int, error) {
	if appellantID <= 0 {
		return nil, 0, apperrors.Invalid("comment.appeal.list", "invalid appellant")
	}
	return listCommentAppeals(ctx, c, 0, appellantID, 0, current, size, false, "")
}

// ResolveCommentAppeal records the owner's or administrator's decision. A
// decision by the reading-list owner keeps the appeal at the owner stage so the
// reader can escalate a rejection to an administrator.
func (c *MyCommentRepo) ResolveCommentAppeal(ctx context.Context, actorID int, actorRole string, appealID int, decision, reason string) (port.CommentAppealOutcome, error) {
	outcome := port.CommentAppealOutcome{}
	if actorID <= 0 || appealID <= 0 {
		return outcome, apperrors.Invalid("comment.appeal.resolve", "invalid target")
	}
	decision = strings.TrimSpace(decision)
	if decision != "restore" && decision != "reject" {
		return outcome, apperrors.Invalid("comment.appeal.resolve", "unsupported decision")
	}
	ownerID := actorID
	if actorRole == "admin" {
		ownerID = 0
	}
	err := ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		var appeal struct {
			Id           int    `xorm:"id"`
			CommentId    int    `xorm:"comment_id"`
			CollectionId int    `xorm:"collection_id"`
			AppellantId  int    `xorm:"appellant_id"`
			OwnerId      int    `xorm:"owner_id"`
			Stage        string `xorm:"stage"`
			Status       string `xorm:"status"`
		}
		found, err := session.SQL(`SELECT appeal.id, appeal.comment_id, appeal.collection_id, appeal.appellant_id,
				COALESCE(collection.user_id, 0) AS owner_id, appeal.stage, appeal.status
			FROM t_comment_appeal appeal
			LEFT JOIN t_collection collection ON collection.id = appeal.collection_id
			WHERE appeal.id = ?`, appealID).Get(&appeal)
		if err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "comment.appeal.lookup", err)
		}
		if !found {
			return apperrors.NotFound("comment.appeal.resolve")
		}
		if appeal.Status != "pending" {
			return apperrors.Conflict("comment.appeal.resolve", "appeal already handled")
		}
		if actorRole != "admin" && appeal.Stage != "owner" {
			return apperrors.Invalid("comment.appeal.resolve", "appeal escalated to admin")
		}
		if _, err := moderationTarget(session, ownerID, appeal.CollectionId, appeal.CommentId, true); err != nil {
			return err
		}
		status := "rejected"
		stage := appeal.Stage
		action := "appeal_reject"
		if decision == "restore" {
			status = "restored"
			action = "restore"
			if err := applyCommentModeration(session, actorID, actorRole, actionRestore, ownerID, appeal.CollectionId, appeal.CommentId, "", reason); err != nil {
				return err
			}
		}
		if actorRole == "admin" {
			stage = "admin"
		}
		if _, err := session.Exec(`UPDATE t_comment_appeal
			SET status = ?, stage = ?, handled_by = ?, handled_at = CURRENT_TIMESTAMP, decision_reason = ?
			WHERE id = ?`, status, stage, actorID, reason, appealID); err != nil {
			return apperrors.Unavailable("comment.appeal.resolve", err)
		}
		outcome = port.CommentAppealOutcome{AppellantID: appeal.AppellantId, CollectionOwnerID: appeal.OwnerId, CommentID: appeal.CommentId, CollectionID: appeal.CollectionId}
		if decision == "restore" {
			return nil
		}
		return writeModerationLog(session, appeal.CommentId, appeal.CollectionId, actorID, actorRole, action, nil, nil, nil, "", reason)
	})
	if err != nil {
		return port.CommentAppealOutcome{}, err
	}
	return outcome, nil
}

// EscalateCommentAppeal moves an owner rejection to the administrator queue.
func (c *MyCommentRepo) EscalateCommentAppeal(ctx context.Context, appellantID, appealID int) error {
	if appellantID <= 0 || appealID <= 0 {
		return apperrors.Invalid("comment.appeal.escalate", "invalid target")
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		var appeal struct {
			CommentId    int    `xorm:"comment_id"`
			CollectionId int    `xorm:"collection_id"`
			Stage        string `xorm:"stage"`
			Status       string `xorm:"status"`
		}
		found, err := session.SQL(`SELECT comment_id, collection_id, stage, status FROM t_comment_appeal
			WHERE id = ? AND appellant_id = ?`, appealID, appellantID).Get(&appeal)
		if err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "comment.appeal.escalate", err)
		}
		if !found {
			return apperrors.NotFound("comment.appeal.escalate")
		}
		if appeal.Status != "rejected" || appeal.Stage != "owner" {
			return apperrors.Conflict("comment.appeal.escalate", "appeal cannot be escalated")
		}
		if _, err := session.Exec(`UPDATE t_comment_appeal
			SET stage = 'admin', status = 'pending', escalated_at = CURRENT_TIMESTAMP, handled_by = 0, handled_at = NULL
			WHERE id = ?`, appealID); err != nil {
			return apperrors.Unavailable("comment.appeal.escalate", err)
		}
		return writeModerationLog(session, appeal.CommentId, appeal.CollectionId, appellantID, "commenter", "appeal_escalate", nil, nil, nil, "", "")
	})
}

// CreateModerationNotification raises an in-app notice for a moderation outcome.
// It reuses the interaction notification pipeline so the account-level
// interaction preference still decides whether the notice is delivered.
func (c *MyCommentRepo) CreateModerationNotification(ctx context.Context, recipientID, actorID int, contentType string, contentID, commentID int, dedupeKey string) error {
	if recipientID <= 0 || recipientID == actorID {
		return nil
	}
	if contentType == "" {
		contentType = "collection"
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		return insertInteractionNotification(session, recipientID, actorID, "moderation", contentType, contentID, commentID, dedupeKey)
	})
}
