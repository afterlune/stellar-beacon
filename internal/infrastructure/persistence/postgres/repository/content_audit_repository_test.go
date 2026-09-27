package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

func TestContentAuditWhereIncludesAllSupportedFilters(t *testing.T) {
	where, args := contentAuditWhere(port.ContentAuditFilter{
		ContentType: "article", Operation: "batch_status", Result: "success", Keyword: "test",
		StartDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	})
	for _, fragment := range []string{"content_type = ?", "operation = ?", "result = ?", "operator_nickname ILIKE ?", "created_at >= ?", "created_at < ?"} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("missing %q in %s", fragment, where)
		}
	}
	if len(args) != 7 {
		t.Fatalf("unexpected args: %#v", args)
	}
}
