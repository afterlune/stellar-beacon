package repository

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"xorm.io/xorm"
)

type studioBatchTarget struct {
	ID               int    `xorm:"id"`
	Title            string `xorm:"title"`
	Status           int    `xorm:"status"`
	ModerationStatus string `xorm:"moderation_status"`
}

func (r *MyPlatformRepo) PreviewOwnedContent(ctx context.Context, userID int, contentType port.StudioContentType, filter port.StudioFilter) (port.StudioBatchPreview, error) {
	table, alias, titleColumn, ok := studioContentSource(contentType)
	if !ok {
		return port.StudioBatchPreview{}, apperrors.Invalid("platform.studio.content.preview", "invalid content type")
	}
	session, err := repoSession(r.engine, ctx, "platform.studio.content.preview")
	if err != nil {
		return port.StudioBatchPreview{}, err
	}
	where, args := ownedFilter(userID, filter, alias)
	var preview port.StudioBatchPreview
	if _, err := session.SQL("SELECT count(1) FROM "+table+" "+alias+" WHERE "+where, args...).Get(&preview.Count); err != nil {
		return port.StudioBatchPreview{}, apperrors.Unavailable("platform.studio.content.preview.count", err)
	}
	if _, err := session.SQL("SELECT COALESCE(MAX("+alias+".id), 0) FROM "+table+" "+alias+" WHERE "+where, args...).Get(&preview.MaxID); err != nil {
		return port.StudioBatchPreview{}, apperrors.Unavailable("platform.studio.content.preview.max_id", err)
	}
	type statusCount struct {
		Status int `xorm:"status"`
		Count  int `xorm:"count"`
	}
	var counts []statusCount
	if err := session.SQL("SELECT "+alias+".status, count(1) AS count FROM "+table+" "+alias+" WHERE "+where+" GROUP BY "+alias+".status", args...).Find(&counts); err != nil {
		return port.StudioBatchPreview{}, apperrors.Unavailable("platform.studio.content.preview.statuses", err)
	}
	preview.StatusCounts = make(map[string]int, len(counts))
	for _, item := range counts {
		preview.StatusCounts[strconv.Itoa(item.Status)] = item.Count
	}
	if _, err := session.SQL("SELECT count(1) FROM "+table+" "+alias+" WHERE "+where+" AND "+alias+".moderation_status = 'hidden'", args...).Get(&preview.HiddenCount); err != nil {
		return port.StudioBatchPreview{}, apperrors.Unavailable("platform.studio.content.preview.hidden", err)
	}
	var rows []studioBatchTarget
	if err := session.SQL("SELECT "+alias+".id, "+alias+"."+titleColumn+" AS title, "+alias+".status, "+alias+".moderation_status FROM "+table+" "+alias+" WHERE "+where+" ORDER BY "+alias+".id DESC LIMIT 10", args...).Find(&rows); err != nil {
		return port.StudioBatchPreview{}, apperrors.Unavailable("platform.studio.content.preview.sample", err)
	}
	preview.Sample = make([]port.StudioBatchPreviewItem, 0, len(rows))
	for _, row := range rows {
		preview.Sample = append(preview.Sample, port.StudioBatchPreviewItem{ID: row.ID, Title: row.Title, Status: row.Status, ModerationStatus: row.ModerationStatus})
	}
	return preview, nil
}

func (r *MyPlatformRepo) BatchUpdateOwnedContentStatus(ctx context.Context, userID int, contentType port.StudioContentType, scope port.StudioBatchScope, status int, actor port.StudioAuditActor) (port.StudioBatchMutation, error) {
	if status < 1 || status > 3 {
		return port.StudioBatchMutation{}, apperrors.Invalid("platform.studio.content.batch-status", "invalid batch status request")
	}
	return r.mutateOwnedContent(ctx, userID, contentType, scope, actor, "batch_status", func(session *xorm.Session, table string, rows []studioBatchTarget) (int, error) {
		ids := batchTargetIDs(rows)
		update := "UPDATE " + table + " SET status = ?, update_time = ? WHERE user_id = ? AND id IN (" + questionMarks(len(ids)) + ")"
		if contentType == port.StudioContentArticle {
			update = "UPDATE t_article SET status = ?, password = '', scheduled_at = NULL, update_time = ? WHERE user_id = ? AND id IN (" + questionMarks(len(ids)) + ")"
		}
		args := make([]interface{}, 0, len(ids)+3)
		args = append(args, status, time.Now(), userID)
		for _, id := range ids {
			args = append(args, id)
		}
		result, err := session.Exec(append([]interface{}{update}, args...)...)
		if err != nil {
			return 0, apperrors.Unavailable("platform.studio.content.batch-status.update", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return 0, apperrors.Unavailable("platform.studio.content.batch-status.rows", err)
		}
		if status == 1 {
			publishedAt := time.Now()
			for _, row := range rows {
				var err error
				switch contentType {
				case port.StudioContentArticle:
					err = recordArticlePublishEvent(session, row.ID, publishedAt)
				case port.StudioContentTalk:
					err = recordTalkPublishEvent(session, row.ID, publishedAt)
				}
				if err != nil {
					return 0, err
				}
			}
		}
		return int(affected), nil
	}, func(studioBatchTarget) int { return status })
}

func (r *MyPlatformRepo) BatchDeleteOwnedContent(ctx context.Context, userID int, contentType port.StudioContentType, scope port.StudioBatchScope, actor port.StudioAuditActor) (port.StudioBatchMutation, error) {
	return r.mutateOwnedContent(ctx, userID, contentType, scope, actor, "batch_delete", func(session *xorm.Session, table string, rows []studioBatchTarget) (int, error) {
		ids := batchTargetIDs(rows)
		args := make([]interface{}, 0, len(ids)+1)
		args = append(args, userID)
		for _, id := range ids {
			args = append(args, id)
		}
		placeholders := questionMarks(len(ids))
		switch contentType {
		case port.StudioContentArticle:
			if _, err := session.Exec(append([]interface{}{"DELETE FROM t_article_tag WHERE article_id IN (SELECT id FROM t_article WHERE user_id = ? AND id IN (" + placeholders + "))"}, args...)...); err != nil {
				return 0, apperrors.Unavailable("platform.studio.content.batch-delete.tags", err)
			}
			result, err := session.Exec(append([]interface{}{"DELETE FROM t_article WHERE user_id = ? AND id IN (" + placeholders + ")"}, args...)...)
			if err != nil {
				return 0, apperrors.Unavailable("platform.studio.content.batch-delete.articles", err)
			}
			affected, _ := result.RowsAffected()
			return int(affected), nil
		case port.StudioContentTalk:
			result, err := session.Exec(append([]interface{}{"DELETE FROM t_talk WHERE user_id = ? AND id IN (" + placeholders + ")"}, args...)...)
			if err != nil {
				return 0, apperrors.Unavailable("platform.studio.content.batch-delete.talks", err)
			}
			affected, _ := result.RowsAffected()
			return int(affected), nil
		case port.StudioContentSeries:
			if _, err := session.Exec(append([]interface{}{"UPDATE t_article SET series_id = NULL, series_order = 0 WHERE user_id = ? AND series_id IN (" + placeholders + ")"}, args...)...); err != nil {
				return 0, apperrors.Unavailable("platform.studio.content.batch-delete.detach", err)
			}
			updateArgs := make([]interface{}, 0, len(ids)+2)
			updateArgs = append(updateArgs, time.Now(), userID)
			for _, id := range ids {
				updateArgs = append(updateArgs, id)
			}
			result, err := session.Exec(append([]interface{}{"UPDATE t_series SET is_delete = 1, update_time = ? WHERE user_id = ? AND id IN (" + placeholders + ")"}, updateArgs...)...)
			if err != nil {
				return 0, apperrors.Unavailable("platform.studio.content.batch-delete.series", err)
			}
			affected, _ := result.RowsAffected()
			return int(affected), nil
		default:
			return 0, apperrors.Invalid("platform.studio.content.batch-delete", "invalid content type")
		}
	}, func(studioBatchTarget) int { return 0 })
}
func (r *MyPlatformRepo) mutateOwnedContent(ctx context.Context, userID int, contentType port.StudioContentType, scope port.StudioBatchScope, actor port.StudioAuditActor, operation string, mutate func(*xorm.Session, string, []studioBatchTarget) (int, error), nextStatus func(studioBatchTarget) int) (port.StudioBatchMutation, error) {
	table, alias, _, ok := studioContentSource(contentType)
	if !ok {
		return port.StudioBatchMutation{}, apperrors.Invalid("platform.studio.content.batch", "invalid content type")
	}
	where, args, err := studioBatchWhere(userID, alias, scope)
	if err != nil {
		return port.StudioBatchMutation{}, err
	}
	scopeJSON, err := json.Marshal(scope)
	if err != nil {
		return port.StudioBatchMutation{}, apperrors.Invalid("platform.studio.content.batch", "invalid scope")
	}
	result := port.StudioBatchMutation{}
	err = ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var rows []studioBatchTarget
		if err := session.SQL("SELECT "+alias+".id, "+alias+"."+studioTitleColumn(contentType)+" AS title, "+alias+".status, "+alias+".moderation_status FROM "+table+" "+alias+" WHERE "+where+" ORDER BY "+alias+".id FOR UPDATE", args...).Find(&rows); err != nil {
			return apperrors.Unavailable("platform.studio.content.batch.lock", err)
		}
		if len(rows) == 0 {
			return apperrors.NotFound("platform.studio.content.batch")
		}
		if scope.Mode == port.StudioBatchScopeIDs && len(rows) != len(uniqueInts(scope.IDs)) {
			return apperrors.NotFound("platform.studio.content.batch")
		}
		if scope.Mode == port.StudioBatchScopeFilter && len(rows) != scope.ExpectedCount {
			return apperrors.Conflict("platform.studio.content.batch", "the selected content changed; refresh and try again")
		}
		audit := &entity.TContentOperationAudit{
			OperatorId: actor.UserID, OperatorNickname: actor.Nickname, ContentType: string(contentType), Operation: operation,
			TargetMode: scope.Mode, FilterSnapshot: string(scopeJSON), SnapshotMaxId: scope.MaxID, RequestedCount: len(rows),
			Result: "success", IpAddress: actor.IPAddress, IpSource: actor.IPSource,
		}
		if _, err := session.Insert(audit); err != nil {
			return apperrors.Unavailable("platform.studio.content.batch.audit", err)
		}
		items := make([]entity.TContentOperationAuditItem, 0, len(rows))
		for _, row := range rows {
			items = append(items, entity.TContentOperationAuditItem{AuditId: audit.Id, ContentId: row.ID, Title: row.Title, PreviousStatus: row.Status, NextStatus: nextStatus(row), Result: "success"})
		}
		if _, err := session.Insert(&items); err != nil {
			return apperrors.Unavailable("platform.studio.content.batch.audit_items", err)
		}
		affected, err := mutate(session, table, rows)
		if err != nil {
			return err
		}
		audit.AffectedCount = affected
		if _, err := session.ID(audit.Id).Cols("affected_count").Update(audit); err != nil {
			return apperrors.Unavailable("platform.studio.content.batch.audit_update", err)
		}
		result = port.StudioBatchMutation{Affected: affected, AuditID: audit.Id, ContentIDs: batchTargetIDs(rows)}
		return nil
	})
	if err != nil {
		r.writeContentAuditFailure(ctx, actor, contentType, operation, scope, string(scopeJSON), err)
		return port.StudioBatchMutation{}, err
	}
	return result, nil
}

func (r *MyPlatformRepo) writeContentAuditFailure(ctx context.Context, actor port.StudioAuditActor, contentType port.StudioContentType, operation string, scope port.StudioBatchScope, scopeJSON string, cause error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.content.audit_failure")
	if err != nil {
		return
	}
	requested := scope.ExpectedCount
	if scope.Mode == port.StudioBatchScopeIDs {
		requested = len(uniqueInts(scope.IDs))
	}
	_, _ = session.Insert(&entity.TContentOperationAudit{
		OperatorId: actor.UserID, OperatorNickname: actor.Nickname, ContentType: string(contentType), Operation: operation,
		TargetMode: scope.Mode, FilterSnapshot: scopeJSON, SnapshotMaxId: scope.MaxID, RequestedCount: requested,
		Result: "failed", ErrorMessage: cause.Error(), IpAddress: actor.IPAddress, IpSource: actor.IPSource,
	})
}

func (r *MyPlatformRepo) RetryScheduledPublication(ctx context.Context, userID, articleID int, actor port.StudioAuditActor) (port.ScheduledPublish, error) {
	if articleID <= 0 {
		return port.ScheduledPublish{}, apperrors.Invalid("platform.studio.publish_retry", "invalid article id")
	}
	var result port.ScheduledPublish
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var row struct {
			RecordID         int       `xorm:"record_id"`
			ArticleID        int       `xorm:"article_id"`
			UserID           int       `xorm:"user_id"`
			ScheduledAt      time.Time `xorm:"scheduled_at"`
			PublishedAt      time.Time `xorm:"published_at"`
			Title            string    `xorm:"title"`
			ModerationStatus string    `xorm:"moderation_status"`
		}
		found, err := session.SQL(`SELECT r.id AS record_id, r.article_id, r.user_id, r.scheduled_at, r.published_at, a.article_title AS title, a.moderation_status FROM t_article_publish_record r JOIN t_article a ON a.id = r.article_id WHERE r.user_id = ? AND r.article_id = ? AND r.notification_state = ? AND r.next_retry_at IS NULL AND a.status = 1 AND a.is_delete = 0 FOR UPDATE`, userID, articleID, port.ScheduledNotificationFailed).Get(&row)
		if err != nil {
			return apperrors.Unavailable("platform.studio.publish_retry.find", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.publish_retry")
		}
		now := time.Now()
		if _, err := session.Exec("UPDATE t_article_publish_record SET notification_state = ?, notification_attempts = 0, next_retry_at = ?, last_error = '', update_time = ? WHERE id = ?", port.ScheduledNotificationPending, now, now, row.RecordID); err != nil {
			return apperrors.Unavailable("platform.studio.publish_retry.update", err)
		}
		audit := &entity.TContentOperationAudit{
			OperatorId: actor.UserID, OperatorNickname: actor.Nickname, ContentType: string(port.StudioContentArticle), Operation: "publish_retry",
			TargetMode: "ids", FilterSnapshot: "{\"articleId\":" + strconv.Itoa(articleID) + "}", SnapshotMaxId: articleID, RequestedCount: 1,
			AffectedCount: 1, Result: "success", IpAddress: actor.IPAddress, IpSource: actor.IPSource,
		}
		if _, err := session.Insert(audit); err != nil {
			return apperrors.Unavailable("platform.studio.publish_retry.audit", err)
		}
		if _, err := session.Insert(&entity.TContentOperationAuditItem{AuditId: audit.Id, ContentId: articleID, Title: row.Title, PreviousStatus: 1, NextStatus: 1, Result: "success"}); err != nil {
			return apperrors.Unavailable("platform.studio.publish_retry.audit_item", err)
		}
		result = port.ScheduledPublish{RecordID: row.RecordID, ArticleID: row.ArticleID, UserID: row.UserID, ScheduledAt: row.ScheduledAt, PublishedAt: row.PublishedAt, ModerationStatus: row.ModerationStatus, NotificationState: port.ScheduledNotificationPending, NotificationAttempts: 0, NextRetryAt: &now, LastError: ""}
		return nil
	})
	return result, err
}
func studioBatchWhere(userID int, alias string, scope port.StudioBatchScope) (string, []interface{}, error) {
	switch scope.Mode {
	case port.StudioBatchScopeIDs:
		ids := uniqueInts(scope.IDs)
		if len(ids) == 0 || len(ids) > 500 || len(ids) != len(scope.IDs) {
			return "", nil, apperrors.Invalid("platform.studio.content.batch", "ids must contain between 1 and 500 unique values")
		}
		args := []interface{}{userID}
		for _, id := range ids {
			args = append(args, id)
		}
		return alias + ".user_id = ? AND " + alias + ".id IN (" + questionMarks(len(ids)) + ")", args, nil
	case port.StudioBatchScopeFilter:
		if scope.ExpectedCount <= 0 || scope.MaxID <= 0 {
			return "", nil, apperrors.Invalid("platform.studio.content.batch", "filter snapshot is required")
		}
		where, args := ownedFilter(userID, port.StudioFilter{Status: scope.Status, SeriesID: scope.SeriesID, Keywords: scope.Keywords}, alias)
		where += " AND " + alias + ".id <= ?"
		args = append(args, scope.MaxID)
		excludes := uniqueInts(scope.ExcludeIDs)
		if len(excludes) > 500 {
			return "", nil, apperrors.Invalid("platform.studio.content.batch", "too many excluded ids")
		}
		if len(excludes) > 0 {
			where += " AND " + alias + ".id NOT IN (" + questionMarks(len(excludes)) + ")"
			for _, id := range excludes {
				args = append(args, id)
			}
		}
		return where, args, nil
	default:
		return "", nil, apperrors.Invalid("platform.studio.content.batch", "invalid selection scope")
	}
}

func studioContentSource(contentType port.StudioContentType) (table, alias, titleColumn string, ok bool) {
	switch contentType {
	case port.StudioContentArticle:
		return "t_article", "a", "article_title", true
	case port.StudioContentTalk:
		return "t_talk", "t", "content", true
	case port.StudioContentSeries:
		return "t_series", "s", "series_name", true
	default:
		return "", "", "", false
	}
}

func studioTitleColumn(contentType port.StudioContentType) string {
	_, _, title, _ := studioContentSource(contentType)
	return title
}

func batchTargetIDs(rows []studioBatchTarget) []int {
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}
