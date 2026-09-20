package repository

import (
	"context"
	"strings"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	pgsql "github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

var _ port.ContentAuditRepository = (*MyContentAuditRepo)(nil)

type MyContentAuditRepo struct{ engine *xorm.Engine }

func NewContentAuditRepo(engine *xorm.Engine) *MyContentAuditRepo {
	return &MyContentAuditRepo{engine: engine}
}

func (r *MyContentAuditRepo) List(ctx context.Context, filter port.ContentAuditFilter) ([]port.ContentAuditRecord, int64, error) {
	session, err := repoSession(r.engine, ctx, "content_audit.list")
	if err != nil {
		return nil, 0, err
	}
	where, args := contentAuditWhere(filter)
	var count int64
	if _, err := session.SQL("SELECT count(1) FROM t_content_operation_audit"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("content_audit.count", err)
	}
	if count == 0 {
		return []port.ContentAuditRecord{}, 0, nil
	}
	page, offset := pgsql.Page(filter.Current, filter.Size)
	var rows []entity.TContentOperationAudit
	listArgs := append(append([]interface{}{}, args...), page, offset)
	if err := session.SQL("SELECT * FROM t_content_operation_audit"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", listArgs...).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("content_audit.list", err)
	}
	result := make([]port.ContentAuditRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, port.ContentAuditRecord{
			ID: row.Id, OperatorID: row.OperatorId, OperatorNickname: row.OperatorNickname,
			ContentType: row.ContentType, Operation: row.Operation, TargetMode: row.TargetMode,
			FilterSnapshot: row.FilterSnapshot, SnapshotMaxID: row.SnapshotMaxId, RequestedCount: row.RequestedCount,
			AffectedCount: row.AffectedCount, Result: row.Result, ErrorMessage: row.ErrorMessage,
			IPAddress: row.IpAddress, IPSource: row.IpSource, CreateTime: row.CreateTime,
		})
	}
	return result, count, nil
}

func (r *MyContentAuditRepo) ListItems(ctx context.Context, auditID, current, size int) ([]port.ContentAuditItem, int64, error) {
	session, err := repoSession(r.engine, ctx, "content_audit.items")
	if err != nil {
		return nil, 0, err
	}
	var count int64
	if _, err := session.SQL("SELECT count(1) FROM t_content_operation_audit_item WHERE audit_id = ?", auditID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("content_audit.items.count", err)
	}
	if count == 0 {
		return []port.ContentAuditItem{}, 0, nil
	}
	page, offset := pgsql.Page(current, size)
	var rows []entity.TContentOperationAuditItem
	if err := session.SQL("SELECT * FROM t_content_operation_audit_item WHERE audit_id = ? ORDER BY id ASC LIMIT ? OFFSET ?", auditID, page, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("content_audit.items.list", err)
	}
	result := make([]port.ContentAuditItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, port.ContentAuditItem{ID: row.Id, ContentID: row.ContentId, Title: row.Title, PreviousStatus: row.PreviousStatus, NextStatus: row.NextStatus, Result: row.Result})
	}
	return result, count, nil
}

func contentAuditWhere(filter port.ContentAuditFilter) (string, []interface{}) {
	where := " WHERE 1 = 1"
	args := make([]interface{}, 0, 6)
	if value := strings.TrimSpace(filter.ContentType); value != "" {
		where += " AND content_type = ?"
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.Operation); value != "" {
		where += " AND operation = ?"
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.Result); value != "" {
		where += " AND result = ?"
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.Keyword); value != "" {
		where += " AND (operator_nickname ILIKE ? OR filter_snapshot::text ILIKE ?)"
		like := "%" + value + "%"
		args = append(args, like, like)
	}
	if !filter.StartDate.IsZero() {
		where += " AND created_at >= ?"
		args = append(args, filter.StartDate)
	}
	if !filter.EndDate.IsZero() {
		where += " AND created_at < ?"
		args = append(args, filter.EndDate)
	}
	return where, args
}
