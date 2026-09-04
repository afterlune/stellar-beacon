package search

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

type fakeArticleChunkEmbeddingGateway struct {
	result port.EmbeddingResult
	err    error
	seen   []port.EmbeddingRequest
}

func (f *fakeArticleChunkEmbeddingGateway) Embed(_ context.Context, request port.EmbeddingRequest) (port.EmbeddingResult, error) {
	f.seen = append(f.seen, request)
	return f.result, f.err
}

type fakeArticleChunkModelRouter struct {
	gateway port.EmbeddingGateway
	route   port.ModelRoute
	err     error
	calls   int
}

func (f *fakeArticleChunkModelRouter) ResolveChat(context.Context, port.AIUseCase) (port.ChatGateway, port.ModelRoute, error) {
	return nil, port.ModelRoute{}, errors.New("chat is not used by article chunk search")
}

func (f *fakeArticleChunkModelRouter) ResolveEmbedding(context.Context, port.AIUseCase) (port.EmbeddingGateway, port.ModelRoute, error) {
	f.calls++
	return f.gateway, f.route, f.err
}

func TestMeiliArticleChunkSearcherKeywordFiltersStaleHits(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/indexes/article_chunks_v1/search" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"hits":[
{"id":"article-7-chunk-0","articleId":7,"articleTitle":"Agent","text":"public chunk","category":"Go","tags":["agent","go"],"articleUrl":"/articles/7","publishedAtUnix":1706745600,"chunkIndex":0,"status":1,"isDelete":0,"_rankingScore":0.88,"_formatted":{"articleTitle":"<mark>Agent</mark>","text":"<mark>public</mark> chunk","tags":["agent","go"],"status":"1"}},
{"id":"article-8-chunk-0","articleId":8,"articleTitle":"Private","text":"private chunk","status":2,"isDelete":0},
{"id":"article-9-chunk-0","articleId":9,"articleTitle":"Deleted","text":"deleted chunk","status":1,"isDelete":1},
{"id":"missing-article","articleId":0,"articleTitle":"Malformed","text":"ignored","status":1,"isDelete":0}
]}`)
	}))
	defer server.Close()

	searcher, err := NewMeiliArticleChunkSearcherWithHTTPClient(server.URL, "test-key", server.Client(), writerTestSpec(), nil)
	if err != nil {
		t.Fatal(err)
	}
	hits, err := searcher.Search(context.Background(), port.KnowledgeQuery{Query: " agent ", Mode: port.SearchModeKeyword})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits = %#v, want one public hit", hits)
	}
	if request["q"] != "agent" || request["filter"] != "isDelete = 0 AND status = 1" {
		t.Fatalf("request = %#v", request)
	}
	if _, ok := request["vector"]; ok {
		t.Fatalf("keyword request unexpectedly contains vector: %#v", request)
	}
	if _, ok := request["hybrid"]; ok {
		t.Fatalf("keyword request unexpectedly contains hybrid: %#v", request)
	}
	wantMetadata := map[string]string{
		"articleId":       "7",
		"category":        "Go",
		"tags":            "agent,go",
		"publishedAtUnix": "1706745600",
		"chunkIndex":      "0",
		"status":          "1",
		"isDelete":        "0",
	}
	if hits[0].Index != "article_chunks_v1" || hits[0].ID != "article-7-chunk-0" || hits[0].Score != 0.88 || !reflect.DeepEqual(hits[0].Metadata, wantMetadata) {
		t.Fatalf("decoded hit = %#v", hits[0])
	}
	if hits[0].Highlights[ArticleChunkTitleField] != "<mark>Agent</mark>" || hits[0].Highlights[ArticleChunkTextField] != "<mark>public</mark> chunk" {
		t.Fatalf("highlights = %#v", hits[0].Highlights)
	}
}

func TestMeiliArticleChunkSearcherRejectsMalformedHTTPEnvelopes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	searcher, err := NewMeiliArticleChunkSearcherWithHTTPClient(server.URL, "", server.Client(), writerTestSpec(), nil)
	if err != nil {
		t.Fatal(err)
	}
	hits, err := searcher.Search(context.Background(), port.KnowledgeQuery{Query: "agent", Mode: port.SearchModeKeyword})
	if !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("malformed chunk response error kind = %v, want unavailable", apperrors.KindOf(err))
	}
	if hits != nil {
		t.Fatalf("malformed chunk response returned hits: %#v", hits)
	}
}

func TestMeiliArticleChunkSearcherPreservesContextErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("canceled chunk search unexpectedly reached Meilisearch")
	}))
	defer server.Close()

	searcher, err := NewMeiliArticleChunkSearcherWithHTTPClient(server.URL, "", server.Client(), writerTestSpec(), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := searcher.Search(ctx, port.KnowledgeQuery{Query: "agent", Mode: port.SearchModeKeyword}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Search() error = %v, want context canceled", err)
	}
}

func TestMeiliArticleChunkSearcherHybridAndSemanticUseUserProvidedVector(t *testing.T) {
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		requests = append(requests, request)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"hits":[]}`)
	}))
	defer server.Close()

	spec := writerTestSpec()
	gateway := &fakeArticleChunkEmbeddingGateway{result: port.EmbeddingResult{
		Vectors:   [][]float32{{0.1, 0.2, 0.3}},
		Model:     spec.Model,
		Dimension: spec.Dimension,
		Version:   spec.ModelVersion,
	}}
	router := &fakeArticleChunkModelRouter{
		gateway: gateway,
		route: port.ModelRoute{
			UseCase:  port.AIUseCaseEmbedding,
			Provider: spec.Provider,
			Model:    spec.Model,
		},
	}
	searcher, err := NewMeiliArticleChunkSearcherWithHTTPClient(server.URL, "", server.Client(), spec, router)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := searcher.Search(context.Background(), port.KnowledgeQuery{Query: "hybrid query", Mode: port.SearchModeHybrid}); err != nil {
		t.Fatalf("hybrid Search() error = %v", err)
	}
	if _, err := searcher.Search(context.Background(), port.KnowledgeQuery{Query: "semantic query", Mode: port.SearchModeSemantic}); err != nil {
		t.Fatalf("semantic Search() error = %v", err)
	}
	if router.calls != 2 || len(gateway.seen) != 2 {
		t.Fatalf("router calls=%d, embedding requests=%d; want two", router.calls, len(gateway.seen))
	}
	for index, request := range gateway.seen {
		wantQuery := []string{"hybrid query", "semantic query"}[index]
		if !reflect.DeepEqual(request.Inputs, []string{wantQuery}) || request.Model != spec.Model || request.Version != spec.ModelVersion {
			t.Fatalf("embedding request %d = %#v", index, request)
		}
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(requests))
	}
	if requests[0]["q"] != "hybrid query" || requests[1]["q"] != "" {
		t.Fatalf("query fields = %#v, %#v", requests[0]["q"], requests[1]["q"])
	}
	for index, request := range requests {
		vector, ok := request["vector"].([]any)
		if !ok || len(vector) != 3 {
			t.Fatalf("request %d vector = %#v", index, request["vector"])
		}
		hybrid, ok := request["hybrid"].(map[string]any)
		if !ok || hybrid["embedder"] != "default" {
			t.Fatalf("request %d hybrid = %#v", index, request["hybrid"])
		}
		wantRatio := float64(port.DefaultSemanticRatio)
		if index == 1 {
			wantRatio = 1
		}
		if hybrid["semanticRatio"] != wantRatio {
			t.Fatalf("request %d semantic ratio = %#v, want %v", index, hybrid["semanticRatio"], wantRatio)
		}
		if request["filter"] != "isDelete = 0 AND status = 1" {
			t.Fatalf("request %d filter = %#v", index, request["filter"])
		}
	}
}

func TestMeiliArticleChunkSearcherUsesConfiguredHybridRatio(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"hits":[]}`)
	}))
	defer server.Close()

	spec := writerTestSpec()
	gateway := &fakeArticleChunkEmbeddingGateway{result: port.EmbeddingResult{
		Vectors:   [][]float32{{0.1, 0.2, 0.3}},
		Model:     spec.Model,
		Dimension: spec.Dimension,
		Version:   spec.ModelVersion,
	}}
	router := &fakeArticleChunkModelRouter{
		gateway: gateway,
		route:   port.ModelRoute{Provider: spec.Provider, Model: spec.Model, UseCase: port.AIUseCaseEmbedding},
	}
	searcher, err := NewMeiliArticleChunkSearcherWithHTTPClientAndSemanticRatio(server.URL, "", server.Client(), spec, router, 0.42)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := searcher.Search(context.Background(), port.KnowledgeQuery{Query: "agent", Mode: port.SearchModeHybrid}); err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	hybrid, ok := request["hybrid"].(map[string]any)
	if !ok || math.Abs(hybrid["semanticRatio"].(float64)-0.42) > 0.000001 {
		t.Fatalf("hybrid request = %#v, want ratio 0.42", request["hybrid"])
	}
}

func TestMeiliArticleChunkSearcherBuildsSafeTypedFilters(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"hits":[]}`)
	}))
	defer server.Close()

	searcher, err := NewMeiliArticleChunkSearcherWithHTTPClient(server.URL, "", server.Client(), writerTestSpec(), nil)
	if err != nil {
		t.Fatal(err)
	}
	filter := port.KnowledgeFilter{
		Category: `Go" OR isDelete = 1`,
		Tags:     []string{"agent", "go"},
		Year:     2024,
		From:     time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC),
	}
	if _, err := searcher.Search(context.Background(), port.KnowledgeQuery{Query: "agent", Mode: port.SearchModeKeyword, Filter: filter}); err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	filterText, ok := request["filter"].(string)
	if !ok {
		t.Fatalf("filter = %#v", request["filter"])
	}
	for _, expected := range []string{
		"isDelete = 0 AND status = 1",
		`category = "Go\" OR isDelete = 1"`,
		`tags = "agent"`,
		`tags = "go"`,
		"publishedAtUnix >= 1704067200",
		"publishedAtUnix < 1735689600",
		"publishedAtUnix >= 1706745600",
		"publishedAtUnix < 1709251200",
	} {
		if !strings.Contains(filterText, expected) {
			t.Fatalf("filter %q is missing %q", filterText, expected)
		}
	}
	if strings.Contains(filterText, `category = "Go" OR`) {
		t.Fatalf("category value escaped outside its quoted literal: %q", filterText)
	}
}

func TestMeiliArticleChunkSearcherRejectsInvalidConfiguredHybridRatio(t *testing.T) {
	spec := writerTestSpec()
	for _, ratio := range []float32{-0.1, 1.1} {
		if _, err := NewMeiliArticleChunkSearcherWithHTTPClientAndSemanticRatio("http://search.test", "", nil, spec, nil, ratio); !apperrors.IsKind(err, apperrors.KindValidation) {
			t.Fatalf("ratio %v error kind = %v, want validation", ratio, apperrors.KindOf(err))
		}
	}
}

func TestMeiliArticleChunkSearcherClassifiesProviderAndHTTPFailures(t *testing.T) {
	spec := writerTestSpec()
	providerRouter := &fakeArticleChunkModelRouter{err: errors.New("provider offline")}
	searcher, err := NewMeiliArticleChunkSearcherWithHTTPClient("http://127.0.0.1:1", "", nil, spec, providerRouter)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := searcher.Search(context.Background(), port.KnowledgeQuery{Query: "query", Mode: port.SearchModeHybrid}); !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("provider error kind = %v, want unavailable", apperrors.KindOf(err))
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "meili unavailable")
	}))
	defer server.Close()
	searcher, err = NewMeiliArticleChunkSearcherWithHTTPClient(server.URL, "", server.Client(), spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := searcher.Search(context.Background(), port.KnowledgeQuery{Query: "query", Mode: port.SearchModeKeyword}); !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("HTTP error kind = %v, want unavailable", apperrors.KindOf(err))
	}

	if _, err := searcher.Search(context.Background(), port.KnowledgeQuery{Query: "query", Mode: port.SearchMode("vector")}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("invalid mode kind = %v, want validation", apperrors.KindOf(err))
	}
	if !strings.Contains(searcher.spec.UID, "article_chunks_") {
		t.Fatalf("unexpected spec UID: %q", searcher.spec.UID)
	}
}
