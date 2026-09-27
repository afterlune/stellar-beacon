package repository

import (
	"strings"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

func TestStudioBatchWhereBuildsFilterSnapshot(t *testing.T) {
	where, args, err := studioBatchWhere(7, "a", port.StudioBatchScope{
		Mode: port.StudioBatchScopeFilter, Status: 3, Keywords: "Go", SeriesID: 4, MaxID: 99, ExcludeIDs: []int{8}, ExpectedCount: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(where, "a.id <= ?") || !strings.Contains(where, "a.id NOT IN (?)") {
		t.Fatalf("unexpected where: %s", where)
	}
	if len(args) != 6 || args[0] != 7 || args[5] != 8 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestStudioBatchWhereRejectsDuplicateOrOversizedIDs(t *testing.T) {
	if _, _, err := studioBatchWhere(7, "a", port.StudioBatchScope{Mode: port.StudioBatchScopeIDs, IDs: []int{1, 1}}); err == nil {
		t.Fatal("duplicate ids should be rejected")
	}
	ids := make([]int, 501)
	for index := range ids {
		ids[index] = index + 1
	}
	if _, _, err := studioBatchWhere(7, "a", port.StudioBatchScope{Mode: port.StudioBatchScopeIDs, IDs: ids}); err == nil {
		t.Fatal("oversized ids should be rejected")
	}
}
