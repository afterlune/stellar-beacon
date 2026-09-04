package port

import (
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestNormalizeSearchModeDefaultsToKeyword(t *testing.T) {
	mode, err := NormalizeSearchMode("  ")
	if err != nil || mode != SearchModeKeyword {
		t.Fatalf("NormalizeSearchMode() = %q, %v; want keyword", mode, err)
	}
	if _, err := NormalizeSearchMode("vector"); err == nil {
		t.Fatal("unknown search mode was accepted")
	}
}

func TestKnowledgeQueryNormalizeAppliesModeDefaults(t *testing.T) {
	query, err := (KnowledgeQuery{Query: "  agent  ", Mode: SearchModeHybrid}).Normalize("article_chunks_v1")
	if err != nil {
		t.Fatal(err)
	}
	if query.Index != "article_chunks_v1" || query.Query != "agent" || query.Limit != DefaultKnowledgeSearchLimit || query.SemanticRatio != DefaultSemanticRatio {
		t.Fatalf("normalized query = %+v", query)
	}

	query, err = (KnowledgeQuery{Query: "agent", Mode: SearchModeSemantic}).Normalize("article_chunks_v1")
	if err != nil || query.SemanticRatio != 1 {
		t.Fatalf("semantic query = %+v, error=%v", query, err)
	}
	for _, invalid := range []KnowledgeQuery{
		{Query: "", Mode: SearchModeKeyword},
		{Query: "agent", Mode: SearchModeKeyword, Limit: MaxKnowledgeSearchLimit + 1},
		{Query: "agent", Mode: SearchModeHybrid, SemanticRatio: 1.1},
	} {
		if _, err := invalid.Normalize("article_chunks_v1"); err == nil {
			t.Fatalf("invalid query was accepted: %+v", invalid)
		}
	}
}

func TestKnowledgeQueryNormalizeUsesConfiguredHybridRatio(t *testing.T) {
	query, err := (KnowledgeQuery{Query: "agent", Mode: SearchModeHybrid}).NormalizeWithDefault("article_chunks_v1", 0.42)
	if err != nil {
		t.Fatal(err)
	}
	if query.SemanticRatio != 0.42 {
		t.Fatalf("semantic ratio = %v, want 0.42", query.SemanticRatio)
	}
	if _, err := (KnowledgeQuery{Query: "agent", Mode: SearchModeHybrid}).NormalizeWithDefault("article_chunks_v1", 0); err == nil {
		t.Fatal("zero default semantic ratio was accepted")
	}
	if _, err := (KnowledgeQuery{Query: "agent", Mode: SearchModeHybrid}).NormalizeWithDefault("article_chunks_v1", 1.1); err == nil {
		t.Fatal("out-of-range default semantic ratio was accepted")
	}
}

func TestParseKnowledgeFilterNormalizesValuesAndDateOnlyEnd(t *testing.T) {
	filter, err := ParseKnowledgeFilter(KnowledgeFilterInput{
		Category: ` Go" `,
		Tags:     []string{" agent,go ", "agent"},
		Year:     "2024",
		From:     "2024-02-01",
		To:       "2024-02-29",
	})
	if err != nil {
		t.Fatal(err)
	}
	if filter.Category != `Go"` || !reflect.DeepEqual(filter.Tags, []string{"agent", "go"}) || filter.Year != 2024 {
		t.Fatalf("normalized filter = %+v", filter)
	}
	wantFrom := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)
	if !filter.From.Equal(wantFrom) || !filter.To.Equal(wantTo) {
		t.Fatalf("date range = %v to %v, want %v to %v", filter.From, filter.To, wantFrom, wantTo)
	}
}

func TestKnowledgeFilterRejectsMalformedAndUnsafeRanges(t *testing.T) {
	invalid := []KnowledgeFilterInput{
		{Year: "20x4"},
		{Year: "0"},
		{From: "2024-02-02", To: "2024-02-01"},
		{From: "not-a-date"},
	}
	for _, input := range invalid {
		if _, err := ParseKnowledgeFilter(input); err == nil {
			t.Fatalf("invalid filter was accepted: %+v", input)
		}
	}
	if _, err := (KnowledgeFilter{Category: "line\nfeed"}).Normalize(); err == nil {
		t.Fatal("control character in category was accepted")
	}
	tooManyTags := make([]string, maxKnowledgeFilterTags+1)
	for index := range tooManyTags {
		tooManyTags[index] = "tag-" + strconv.Itoa(index)
	}
	if _, err := (KnowledgeFilter{Tags: tooManyTags}).Normalize(); err == nil {
		t.Fatal("too many tag filters were accepted")
	}
}
