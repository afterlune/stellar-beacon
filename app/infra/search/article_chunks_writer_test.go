package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

func writerTestSpec() ArticleChunksIndexSpec {
	return ArticleChunksIndexSpec{
		UID:                "article_chunks_v1",
		PrimaryKey:         ArticleChunksIndexPrimaryKey,
		Provider:           "openai",
		Model:              "text-embedding-3-small",
		ModelVersion:       "2026-08-28",
		IndexVersion:       "v1",
		Dimension:          3,
		EmbeddingBatchSize: 2,
	}
}

func writerTestChunk() port.IndexedArticleChunk {
	return port.IndexedArticleChunk{
		Document: port.ArticleChunkDocument{
			ID:           "article-7-chunk-0",
			ArticleID:    7,
			ArticleTitle: "Agent notes",
			Text:         "retrieval text",
			Category:     "engineering",
			Tags:         []string{"go", "agent"},
			ArticleURL:   "/articles/7",
			PublishedAt:  time.Date(2026, 8, 28, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
			ChunkIndex:   0,
			Status:       port.PublicArticleStatus,
			IsDelete:     0,
		},
		Embedding:          []float32{0.1, 0.2, 0.3},
		EmbeddingModel:     "text-embedding-3-small",
		EmbeddingVersion:   "2026-08-28",
		EmbeddingDimension: 3,
	}
}

func TestMeiliArticleChunkIndexUpsertUsesStableVersionedDocuments(t *testing.T) {
	var received []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/indexes/article_chunks_v1/documents" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("primaryKey") != "id" {
			t.Fatalf("primaryKey query = %q", r.URL.Query().Get("primaryKey"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, &received); err != nil {
			t.Fatalf("decode documents: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"taskUid":11}`)
	}))
	defer server.Close()

	index, err := NewMeiliArticleChunkIndexWithHTTPClient(server.URL, "test-key", server.Client(), writerTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	chunk := writerTestChunk()
	if err := index.UpsertChunks(context.Background(), []port.IndexedArticleChunk{chunk}); err != nil {
		t.Fatalf("UpsertChunks() error = %v", err)
	}
	if len(received) != 1 || received[0]["id"] != "article-7-chunk-0" || received[0]["articleId"] != float64(7) {
		t.Fatalf("received documents = %#v", received)
	}
	if received[0]["embeddingModel"] != "text-embedding-3-small" || received[0]["embeddingVersion"] != "2026-08-28" {
		t.Fatalf("embedding metadata = %#v", received[0])
	}
	if received[0]["publishedAtUnix"] != float64(chunk.Document.PublishedAt.Unix()) {
		t.Fatalf("publishedAtUnix = %#v, want %d", received[0]["publishedAtUnix"], chunk.Document.PublishedAt.Unix())
	}
	vectors, ok := received[0]["_vectors"].(map[string]any)
	if !ok {
		t.Fatalf("_vectors = %#v, want object", received[0]["_vectors"])
	}
	if !reflect.DeepEqual(vectors["default"], []any{0.1, 0.2, 0.3}) {
		t.Fatalf("default vector = %#v", vectors["default"])
	}
}

func TestMeiliArticleChunkIndexDeleteUsesArticleFilter(t *testing.T) {
	var filter string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/indexes/article_chunks_v1/documents/delete" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		filter = payload["filter"]
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	index, err := NewMeiliArticleChunkIndexWithHTTPClient(server.URL, "", server.Client(), writerTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	if err := index.DeleteArticle(context.Background(), 7); err != nil {
		t.Fatalf("DeleteArticle() error = %v", err)
	}
	if filter != "articleId = 7" || strings.Contains(filter, "'") {
		t.Fatalf("filter = %q, want numeric article filter", filter)
	}
}

func TestMeiliArticleChunkIndexRejectsMismatchedEmbeddingContract(t *testing.T) {
	index, err := NewMeiliArticleChunkIndexWithHTTPClient("http://127.0.0.1:1", "", nil, writerTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	chunk := writerTestChunk()
	chunk.EmbeddingVersion = "old"
	if err := index.UpsertChunks(context.Background(), []port.IndexedArticleChunk{chunk}); err == nil {
		t.Fatal("UpsertChunks() error = nil, want contract validation error")
	}
}

func TestMeiliArticleChunkIndexRejectsNonPublicDocumentsBeforeHTTP(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	index, err := NewMeiliArticleChunkIndexWithHTTPClient(server.URL, "", server.Client(), writerTestSpec())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		status  int
		deleted int
	}{
		{name: "private", status: 2},
		{name: "deleted", status: port.PublicArticleStatus, deleted: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			chunk := writerTestChunk()
			chunk.Document.Status = test.status
			chunk.Document.IsDelete = test.deleted
			if err := index.UpsertChunks(context.Background(), []port.IndexedArticleChunk{chunk}); !apperrors.IsKind(err, apperrors.KindValidation) {
				t.Fatalf("UpsertChunks() error = %v, want validation", err)
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatalf("HTTP requests = %d, want no writes for non-public documents", requests.Load())
	}
}
