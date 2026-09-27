package service

import (
	"context"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type ContentAuditService interface {
	ListContentAudits(context.Context, port.ContentAuditFilter) ([]port.ContentAuditRecord, int64, error)
	ListContentAuditItems(context.Context, int, int, int) ([]port.ContentAuditItem, int64, error)
}

type MyContentAuditService struct{ repo port.ContentAuditRepository }

func NewContentAuditService(repo port.ContentAuditRepository) *MyContentAuditService {
	return &MyContentAuditService{repo: repo}
}

func (s *MyContentAuditService) ListContentAudits(ctx context.Context, filter port.ContentAuditFilter) ([]port.ContentAuditRecord, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, apperrors.Unavailable("content_audit.service", nil)
	}
	return s.repo.List(ctx, filter)
}

func (s *MyContentAuditService) ListContentAuditItems(ctx context.Context, auditID, current, size int) ([]port.ContentAuditItem, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, apperrors.Unavailable("content_audit.service", nil)
	}
	return s.repo.ListItems(ctx, auditID, current, size)
}
