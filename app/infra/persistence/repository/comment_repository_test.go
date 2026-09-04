package repository

import (
	"benetnasch/app/infra/persistence/pgsql"
	"strings"
	"testing"
)

func TestPublicCommentTargetFilterFailsClosedForArticleAndTalk(t *testing.T) {
	for _, test := range []struct {
		name   string
		typeID int
	}{
		{name: "article", typeID: 1},
		{name: "talk", typeID: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			filter := publicCommentTargetFilter(test.typeID)
			if !strings.Contains(filter, "status = 1") {
				t.Fatalf("public target filter = %q", filter)
			}
			if test.typeID == 1 && !strings.Contains(filter, "is_delete = 0") {
				t.Fatalf("article target filter = %q", filter)
			}
		})
	}
	if filter := publicCommentTargetFilter(2); filter != "" {
		t.Fatalf("non-topic comment filter = %q, want empty", filter)
	}
}

func TestPublicCommentQueriesExcludeDeletedAndPrivateTargets(t *testing.T) {
	topSix := strings.ToLower(strings.Join(strings.Fields(pgsql.ListTopSixComments), " "))
	for _, expected := range []string{
		"c.is_review = 1",
		"c.is_delete = 0",
		"a_public.is_delete = 0",
		"a_public.status = 1",
		"t_public.status = 1",
	} {
		if !strings.Contains(topSix, expected) {
			t.Fatalf("top-six query missing %q: %s", expected, pgsql.ListTopSixComments)
		}
	}
	parentFilter := strings.ToLower(publicParentCommentTargetFilter)
	for _, expected := range []string{"a_public.status = 1", "t_public.status = 1"} {
		if !strings.Contains(parentFilter, expected) {
			t.Fatalf("reply target filter missing %q: %s", expected, publicParentCommentTargetFilter)
		}
	}
}
