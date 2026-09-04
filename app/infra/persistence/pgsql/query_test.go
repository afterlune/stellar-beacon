package pgsql

import (
	"strings"
	"testing"
)

func TestContainsPatternEscapesLikeWildcards(t *testing.T) {
	got := ContainsPattern(`a%_\\b`)
	want := `%a\%\_\\\\b%`
	if got != want {
		t.Fatalf("ContainsPattern() = %q, want %q", got, want)
	}
}

func TestPageNormalizesInput(t *testing.T) {
	if limit, offset := Page(0, 0); limit != DefaultPageSize || offset != 0 {
		t.Fatalf("Page(0, 0) = (%d, %d)", limit, offset)
	}
	if limit, offset := Page(3, MaxPageSize+1); limit != MaxPageSize || offset != MaxPageSize*2 {
		t.Fatalf("Page(3, too large) = (%d, %d)", limit, offset)
	}
}

func TestPublicArticleQueriesExcludePrivateArticles(t *testing.T) {
	queries := map[string]string{
		"top and featured": ListTopAndFeaturedArticles,
		"list":             ListArticles,
		"category":         GetArticlesByCategoryId,
		"previous":         GetPreArticleById,
		"next":             GetNextArticleById,
		"first":            GetFirstArticle,
		"last":             GetLastArticle,
		"tag":              ListArticlesByTagId,
		"count":            CountPublicArticles,
		"category count":   CountPublicArticlesByCategoryID,
		"tag count":        CountPublicArticlesByTagID,
		"categories":       ListCategories,
		"tags":             ListTags,
		"top tags":         ListTopTenTags,
		"talk count":       CountPublicTalks,
	}

	for name, query := range queries {
		normalized := strings.ToLower(strings.Join(strings.Fields(query), " "))
		if strings.Contains(normalized, "status in (1, 2)") {
			t.Errorf("%s query still includes private articles: %s", name, query)
		}
		if !strings.Contains(normalized, "status = 1") {
			t.Errorf("%s query does not require published status: %s", name, query)
		}
	}
}

func TestArticleDetailQueryRetainsPrivateAccessForApplicationAuthorization(t *testing.T) {
	normalized := strings.ToLower(strings.Join(strings.Fields(GetArticleById), " "))
	if !strings.Contains(normalized, "status in (1, 2)") {
		t.Fatalf("article detail query no longer supports password-protected articles: %s", GetArticleById)
	}
}

func TestRoleLookupExcludesDisabledRoles(t *testing.T) {
	normalized := strings.ToLower(strings.Join(strings.Fields(ListRolesByUserInfoId), " "))
	if !strings.Contains(normalized, "r.is_disable = 0") {
		t.Fatalf("role lookup must not authorize disabled roles: %s", ListRolesByUserInfoId)
	}
}
