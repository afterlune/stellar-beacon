package service

import (
	"context"
	"testing"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type fakeContentAuditRepository struct {
	filter port.ContentAuditFilter
	items  []port.ContentAuditItem
}

func (f *fakeContentAuditRepository) List(_ context.Context, filter port.ContentAuditFilter) ([]port.ContentAuditRecord, int64, error) {
	f.filter = filter
	return []port.ContentAuditRecord{{ID: 1, OperatorID: 7, ContentType: filter.ContentType, Operation: filter.Operation, Result: filter.Result}}, 1, nil
}
func (f *fakeContentAuditRepository) ListItems(context.Context, int, int, int) ([]port.ContentAuditItem, int64, error) {
	return f.items, int64(len(f.items)), nil
}

func TestContentAuditServicePassesTypedFilterToRepository(t *testing.T) {
	repo := &fakeContentAuditRepository{}
	svc := NewContentAuditService(repo)
	filter := port.ContentAuditFilter{
		Current: 1, Size: 10, ContentType: "article", Operation: "batch_status", Result: "success",
		StartDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local),
		EndDate:   time.Date(2026, 9, 21, 0, 0, 0, 0, time.Local),
	}
	records, count, err := svc.ListContentAudits(context.Background(), filter)
	if err != nil || count != 1 || len(records) != 1 {
		t.Fatalf("list failed: records=%+v count=%d err=%v", records, count, err)
	}
	if repo.filter.ContentType != "article" || repo.filter.Operation != "batch_status" || repo.filter.Result != "success" {
		t.Fatalf("unexpected filter: %+v", repo.filter)
	}
	if !repo.filter.StartDate.Equal(filter.StartDate) || !repo.filter.EndDate.Equal(filter.EndDate) {
		t.Fatalf("unexpected date range: %+v", repo.filter)
	}
}
