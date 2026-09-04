package search

import (
	"reflect"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestArticleChunkSchemaFieldsAreStable(t *testing.T) {
	want := []string{
		"id",
		"articleId",
		"articleTitle",
		"text",
		"category",
		"tags",
		"articleUrl",
		"publishedAt",
		"publishedAtUnix",
		"chunkIndex",
		"status",
		"isDelete",
	}
	if got := ArticleChunkSchemaFields(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ArticleChunkSchemaFields() = %v, want %v", got, want)
	}
	got := ArticleChunkSchemaFields()
	got[0] = "mutated"
	if ArticleChunkSchemaFields()[0] != "id" {
		t.Fatal("ArticleChunkSchemaFields() leaked mutable internal storage")
	}
}

func TestArticleChunkIDIsDeterministic(t *testing.T) {
	got, err := ArticleChunkID(42, 3)
	if err != nil || got != "article-42-chunk-3" {
		t.Fatalf("ArticleChunkID() = %q, %v", got, err)
	}
	for _, values := range [][2]int{{0, 0}, {-1, 0}, {1, -1}} {
		if _, err := ArticleChunkID(values[0], values[1]); err == nil {
			t.Fatalf("ArticleChunkID(%d, %d) error = nil", values[0], values[1])
		}
	}
}

func TestNormalizeArticleChunkDocumentCanonicalizesAndValidates(t *testing.T) {
	publishedAt := time.Date(2026, 8, 28, 12, 34, 56, 123, time.FixedZone("CST", 8*60*60))
	document := port.ArticleChunkDocument{
		ID:           " article-7-chunk-0 ",
		ArticleID:    7,
		ArticleTitle: "  A title  ",
		Text:         "  body chunk  ",
		Category:     "  Go  ",
		Tags:         []string{" agent ", "Go", "agent"},
		ArticleURL:   " https://example.test/articles/7 ",
		PublishedAt:  publishedAt,
		ChunkIndex:   0,
		Status:       port.PublicArticleStatus,
		IsDelete:     0,
	}
	normalized, err := NormalizeArticleChunkDocument(document)
	if err != nil {
		t.Fatalf("NormalizeArticleChunkDocument() error = %v", err)
	}
	if normalized.ArticleTitle != "A title" || normalized.Text != "body chunk" || normalized.Category != "Go" || normalized.ArticleURL != "https://example.test/articles/7" {
		t.Fatalf("normalized document = %+v", normalized)
	}
	if !reflect.DeepEqual(normalized.Tags, []string{"agent", "Go"}) {
		t.Fatalf("normalized tags = %v", normalized.Tags)
	}
	if normalized.PublishedAt.Location() != time.UTC || normalized.PublishedAt.UnixNano() != publishedAt.UnixNano() {
		t.Fatalf("normalized published time = %v", normalized.PublishedAt)
	}
	if normalized.PublishedAtUnix != publishedAt.Unix() {
		t.Fatalf("normalized published timestamp = %d, want %d", normalized.PublishedAtUnix, publishedAt.Unix())
	}

	invalid := normalized
	invalid.Status = 2
	if _, err := NormalizeArticleChunkDocument(invalid); err == nil {
		t.Fatal("private document was accepted")
	}
	invalid = normalized
	invalid.ID = "wrong-id"
	if _, err := NormalizeArticleChunkDocument(invalid); err == nil {
		t.Fatal("non-deterministic chunk id was accepted")
	}
	invalid = normalized
	invalid.Tags = []string{""}
	if _, err := NormalizeArticleChunkDocument(invalid); err == nil {
		t.Fatal("empty tag was accepted")
	}
	invalid = normalized
	invalid.IsDelete = 1
	if _, err := NormalizeArticleChunkDocument(invalid); err == nil {
		t.Fatal("deleted document was accepted")
	}
}
