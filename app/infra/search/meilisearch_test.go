package search

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeKnowledgeSearcher struct {
	hits         []port.KnowledgeHit
	fallbackHits []port.KnowledgeHit
	query        port.KnowledgeQuery
	queries      []port.KnowledgeQuery
	err          error
}

func (f *fakeKnowledgeSearcher) Search(_ context.Context, query port.KnowledgeQuery) ([]port.KnowledgeHit, error) {
	f.query = query
	f.queries = append(f.queries, query)
	if len(f.queries) == 1 && f.err != nil {
		return nil, f.err
	}
	if f.fallbackHits != nil {
		return f.fallbackHits, nil
	}
	return f.hits, nil
}

func TestMeiliSearcherSearchRequiresConfiguredClient(t *testing.T) {
	_, err := (&MeiliSearcher{}).Search(context.Background(), "keyword")
	if !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("Search() error kind = %v, want %v", errors.KindOf(err), errors.KindUnavailable)
	}
}

func TestMeiliSearcherPreservesContextErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("canceled search unexpectedly reached Meilisearch")
	}))
	defer server.Close()

	searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := searcher.Search(ctx, "agent"); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("Search() error = %v, want context canceled", err)
	}
}

func TestMeiliSearcherHTTPPayloadWorksWithLegacyServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/indexes/articles/search" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request: %v", err)
		}
		if strings.Contains(string(body), "hybrid") {
			t.Fatalf("legacy-incompatible hybrid field was sent: %s", body)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload["attributesToRetrieve"] == nil {
			t.Fatalf("attributesToRetrieve missing from request: %s", body)
		}
		if _, present := payload["attributesToSearchOn"]; present {
			t.Fatalf("legacy-compatible request must omit attributesToSearchOn: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"hits":[{"id":147,"articleTitle":"title","articleContent":"content","isDelete":0,"status":1,"_formatted":{"articleTitle":"<mark>title</mark>"}},{"id":148,"articleTitle":"private","articleContent":"private","isDelete":0,"status":2},{"id":149,"articleTitle":"deleted","articleContent":"deleted","isDelete":1,"status":1}]}`)
	}))
	defer server.Close()

	searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL, ApiKey: "test-key"})
	hits, err := searcher.Search(context.Background(), "keyword")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) != 1 || hits[0].Id != 147 || hits[0].HighlightedTitle == "" {
		t.Fatalf("unexpected search hits: %#v", hits)
	}
}

func TestMeiliSearcherRejectsMalformedHTTPEnvelopes(t *testing.T) {
	for _, responseBody := range []string{
		`{}`,
		`{"hits":null}`,
		`{"hits":{}}`,
		`{"hits":"not-an-array"}`,
	} {
		responseBody := responseBody
		t.Run(responseBody, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, responseBody)
			}))
			defer server.Close()

			searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL})
			hits, err := searcher.Search(context.Background(), "agent")
			if !errors.IsKind(err, errors.KindUnavailable) {
				t.Fatalf("malformed response %s error kind = %v, want unavailable", responseBody, errors.KindOf(err))
			}
			if hits != nil {
				t.Fatalf("malformed response %s returned hits: %#v", responseBody, hits)
			}
		})
	}
}

func TestArticleKeywordIndexContractIsStable(t *testing.T) {
	if ArticlesIndexUID != "articles" || ArticlesIndexPrimaryKey != "id" {
		t.Fatalf("index identity changed: uid=%q primaryKey=%q", ArticlesIndexUID, ArticlesIndexPrimaryKey)
	}
	if got := articleKeywordSearchableFields(); strings.Join(got, ",") != "articleTitle,articleContent" {
		t.Fatalf("searchable fields = %v", got)
	}
	if got := articleKeywordRetrievableFields(); strings.Join(got, ",") != "id,articleTitle,articleContent,isDelete,status" {
		t.Fatalf("retrievable fields = %v", got)
	}
	if isPublicArticleHit(0, 2) || isPublicArticleHit(1, 1) || !isPublicArticleHit(0, 1) {
		t.Fatal("public article filter contract is incorrect")
	}
}

func TestMeiliSearcherMapsAndDeduplicatesChunkHits(t *testing.T) {
	knowledge := &fakeKnowledgeSearcher{hits: []port.KnowledgeHit{
		{Index: "article_chunks_v1", ID: "article-7-chunk-0", Title: "first", Text: "first chunk", URL: "/articles/7", Score: 0.88, Highlights: map[string]string{
			ArticleChunkTitleField: "<mark>first</mark>",
			ArticleChunkTextField:  "<mark>first</mark> chunk",
		}, Metadata: map[string]string{
			"articleId":       "7",
			"category":        "Go",
			"tags":            "agent,go",
			"chunkIndex":      "0",
			"publishedAtUnix": "1706745600",
		}},
		{ID: "article-7-chunk-1", Title: "first", Text: "second chunk", Metadata: map[string]string{"articleId": "7"}},
		{ID: "article-8-chunk-0", Title: "second", Text: "second article"},
	}}
	searcher := &MeiliSearcher{}
	searcher.SetArticleChunkSearcher(knowledge)

	hits, err := searcher.SearchWithMode(context.Background(), "agent", port.SearchModeSemantic)
	if err != nil {
		t.Fatalf("SearchWithMode() error = %v", err)
	}
	if knowledge.query.Query != "agent" || knowledge.query.Mode != port.SearchModeSemantic {
		t.Fatalf("knowledge query = %#v", knowledge.query)
	}
	if len(hits) != 2 || hits[0].Id != 7 || hits[1].Id != 8 {
		t.Fatalf("mapped hits = %#v", hits)
	}
	if hits[0].ArticleContent != "first chunk" || hits[0].HighlightedTitle != "<mark>first</mark>" || hits[0].Status != port.PublicArticleStatus || hits[0].IsDelete != 0 {
		t.Fatalf("first mapped hit = %#v", hits[0])
	}
	if hits[0].Source == nil || hits[0].Source.ArticleID != 7 || hits[0].Source.Title != "first" || hits[0].Source.URL != "/articles/7" || hits[0].Source.ChunkID != "article-7-chunk-0" || hits[0].Source.ChunkIndex != 0 || hits[0].Source.Category != "Go" || strings.Join(hits[0].Source.Tags, ",") != "agent,go" || hits[0].Source.PublishedAtUnix != 1706745600 {
		t.Fatalf("first source = %#v", hits[0].Source)
	}
	if hits[0].Relevance == nil || hits[0].Relevance.Score != 0.88 || hits[0].Relevance.Mode != port.SearchModeSemantic || hits[0].Relevance.Index != "article_chunks_v1" {
		t.Fatalf("first relevance = %#v", hits[0].Relevance)
	}
}

func TestMeiliSearcherFallsBackToLegacyKeywordSearchWhenEmbeddingIsUnavailable(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/indexes/articles/search" {
			t.Fatalf("fallback path = %q, want legacy articles index", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"hits":[{"id":7,"articleTitle":"keyword title","articleContent":"keyword content","isDelete":0,"status":1,"_formatted":{"articleTitle":"<mark>keyword</mark>"}}]}`)
	}))
	defer server.Close()

	knowledge := &fakeKnowledgeSearcher{err: errors.Unavailable(articleChunksEmbeddingOperation, stderrors.New("embedding provider is offline"))}
	searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL})
	searcher.SetArticleChunkSearcher(knowledge)

	hits, err := searcher.SearchWithMode(context.Background(), "keyword", port.SearchModeSemantic)
	if err != nil {
		t.Fatalf("SearchWithMode() error = %v", err)
	}
	if requests != 1 || len(knowledge.queries) != 1 || knowledge.query.Mode != port.SearchModeSemantic {
		t.Fatalf("fallback calls = requests:%d knowledge:%#v", requests, knowledge.queries)
	}
	if len(hits) != 1 || hits[0].Id != 7 || hits[0].ArticleTitle != "keyword title" {
		t.Fatalf("fallback hits = %#v", hits)
	}
	if hits[0].Relevance == nil || hits[0].Relevance.Mode != port.SearchModeKeyword || hits[0].Relevance.Index != ArticlesIndexUID {
		t.Fatalf("fallback relevance = %#v", hits[0].Relevance)
	}
}

func TestMeiliSearcherKeepsStructuredFiltersDuringKeywordFallback(t *testing.T) {
	knowledge := &fakeKnowledgeSearcher{
		err: stderrors.New("embedding provider is offline"),
		fallbackHits: []port.KnowledgeHit{{
			Index: "article_chunks_v1",
			ID:    "article-7-chunk-0",
			Title: "filtered title",
			Text:  "filtered content",
			Metadata: map[string]string{
				"articleId": "7",
			},
		}},
	}
	// The actual adapter marks the failure with the embedding operation. The
	// fake uses the same typed boundary so this test exercises the public
	// fallback policy rather than a provider-specific implementation.
	knowledge.err = errors.Unavailable(articleChunksEmbeddingOperation, knowledge.err)
	searcher := &MeiliSearcher{}
	searcher.SetArticleChunkSearcher(knowledge)

	hits, err := searcher.SearchWithModeAndFilter(context.Background(), "agent", port.SearchModeHybrid, port.KnowledgeFilter{Category: "Go"})
	if err != nil {
		t.Fatalf("SearchWithModeAndFilter() error = %v", err)
	}
	if len(knowledge.queries) != 2 || knowledge.queries[0].Mode != port.SearchModeHybrid || knowledge.queries[1].Mode != port.SearchModeKeyword || knowledge.queries[1].Filter.Category != "Go" {
		t.Fatalf("fallback queries = %#v", knowledge.queries)
	}
	if len(hits) != 1 || hits[0].Relevance == nil || hits[0].Relevance.Mode != port.SearchModeKeyword {
		t.Fatalf("filtered fallback hits = %#v", hits)
	}
}

func TestMeiliSearcherDoesNotFallbackNonEmbeddingFailures(t *testing.T) {
	knowledge := &fakeKnowledgeSearcher{err: errors.Unavailable("search.article_chunks", stderrors.New("Meilisearch is offline"))}
	searcher := &MeiliSearcher{}
	searcher.SetArticleChunkSearcher(knowledge)

	_, err := searcher.SearchWithMode(context.Background(), "agent", port.SearchModeSemantic)
	if !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("error kind = %v, want unavailable", errors.KindOf(err))
	}
	if len(knowledge.queries) != 1 {
		t.Fatalf("knowledge queries = %d, want no fallback query", len(knowledge.queries))
	}
}

func TestMeiliSearcherClassifiesUnwrappedKnowledgeFailures(t *testing.T) {
	knowledge := &fakeKnowledgeSearcher{err: stderrors.New("unexpected search adapter failure")}
	searcher := &MeiliSearcher{}
	searcher.SetArticleChunkSearcher(knowledge)

	_, err := searcher.SearchWithMode(context.Background(), "agent", port.SearchModeSemantic)
	if !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("error kind = %v, want unavailable", errors.KindOf(err))
	}
	if len(knowledge.queries) != 1 {
		t.Fatalf("knowledge queries = %d, want one", len(knowledge.queries))
	}
}

func TestEmbeddingSearchFallbackEligibilityIsConservative(t *testing.T) {
	tests := []struct {
		name string
		mode port.SearchMode
		err  error
		want bool
	}{
		{name: "provider unavailable", mode: port.SearchModeSemantic, err: errors.Unavailable(articleChunksEmbeddingOperation, stderrors.New("offline")), want: true},
		{name: "circuit open", mode: port.SearchModeHybrid, err: errors.Unavailable(articleChunksEmbeddingResolveOperation, errors.NewAI(errors.AICodeCircuitOpen, "embedding", nil)), want: true},
		{name: "invalid contract", mode: port.SearchModeSemantic, err: errors.Unavailable("search.article_chunks.embedding_contract", stderrors.New("dimension mismatch")), want: false},
		{name: "meili unavailable", mode: port.SearchModeSemantic, err: errors.Unavailable("search.article_chunks", stderrors.New("offline")), want: false},
		{name: "cancelled", mode: port.SearchModeSemantic, err: errors.Unavailable(articleChunksEmbeddingOperation, context.Canceled), want: false},
		{name: "keyword mode", mode: port.SearchModeKeyword, err: errors.Unavailable(articleChunksEmbeddingOperation, stderrors.New("offline")), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := embeddingSearchFallbackEligible(tt.mode, tt.err); got != tt.want {
				t.Fatalf("embeddingSearchFallbackEligible() = %v, want %v", got, tt.want)
			}
		})
	}
}
