package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
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

func TestContentAuditListParsesFiltersAndOwnerVisibleFields(t *testing.T) {
	repo := &fakeContentAuditRepository{}
	svc := NewContentAuditService(repo)
	ctx := platformTestContext(http.MethodGet, "/v1/admin/content/audits?contentType=article&operation=batch_status&result=success&startDate=2026-09-01&endDate=2026-09-20&current=1&size=10", "")
	result := svc.ListContentAudits(ctx)
	if !result.Flag {
		t.Fatalf("list failed: %+v", result)
	}
	if repo.filter.ContentType != "article" || repo.filter.Operation != "batch_status" || repo.filter.Result != "success" {
		t.Fatalf("unexpected filter: %+v", repo.filter)
	}
	wantStart := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	wantEnd := time.Date(2026, 9, 21, 0, 0, 0, 0, time.Local)
	if !repo.filter.StartDate.Equal(wantStart) || !repo.filter.EndDate.Equal(wantEnd) {
		t.Fatalf("unexpected date range: %+v", repo.filter)
	}
	if _, ok := result.Data.(model.PageResultDTO); !ok {
		t.Fatalf("unexpected page type: %T", result.Data)
	}
}

func TestContentAuditRejectsUnsupportedOperation(t *testing.T) {
	svc := NewContentAuditService(&fakeContentAuditRepository{})
	result := svc.ListContentAudits(platformTestContext(http.MethodGet, "/v1/admin/content/audits?operation=delete_everything", ""))
	if result.Flag {
		t.Fatalf("unsupported operation must fail: %+v", result)
	}
}
