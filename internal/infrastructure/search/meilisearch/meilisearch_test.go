package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/config"
)

func TestMeiliSearcherSearchRequiresConfiguredClient(t *testing.T) {
	_, err := (&MeiliSearcher{}).Search(context.Background(), "keyword", 0, 20)
	if !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("Search() error kind = %v, want %v", errors.KindOf(err), errors.KindUnavailable)
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
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := payload["hybrid"]; ok {
			t.Fatalf("legacy-incompatible hybrid field was sent: %s", body)
		}
		if payload["limit"] != float64(10) || payload["offset"] != float64(5) || payload["filter"] != publicFilter {
			t.Fatalf("paging/filter payload = %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"hits":[{"id":147,"articleTitle":"title","articleContent":"content","isDelete":0,"status":1,"_formatted":{"articleTitle":"<mark>title</mark>","articleContent":"<mark>content</mark>","author":{"id":"147","nickname":"author"}}}],"estimatedTotalHits":12}`)
	}))
	defer server.Close()

	searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL, ApiKey: "test-key"})
	page, err := searcher.Search(context.Background(), "keyword", 5, 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if page.Total != 12 || len(page.Hits) != 1 || page.Hits[0].Id != 147 || page.Hits[0].HighlightedTitle != "<mark>title</mark>" || page.Hits[0].HighlightedContent != "<mark>content</mark>" {
		t.Fatalf("unexpected search page: %#v", page)
	}
}

func TestMeiliSearcherHTTPFallsBackToTotalHits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"hits":[],"totalHits":4}`)
	}))
	defer server.Close()

	searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL})
	page, err := searcher.Search(context.Background(), "keyword", 0, 20)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if page.Total != 4 || len(page.Hits) != 0 {
		t.Fatalf("unexpected search page: %#v", page)
	}
}

func TestMeiliSearcherHTTPFailureIncludesBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"filter is not filterable"}`)
	}))
	defer server.Close()

	searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL})
	_, err := searcher.Search(context.Background(), "keyword", 0, 20)
	if !errors.IsKind(err, errors.KindUnavailable) || !strings.Contains(err.Error(), "filter is not filterable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMeiliSearcherHTTPReconcileDeletesStaleDocumentsAndUpsertsSnapshot(t *testing.T) {
	indexExists := false
	var settings []string
	var deleted []string
	var upserted []port.ArticleSearch
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/indexes":
			if indexExists {
				w.WriteHeader(http.StatusConflict)
				return
			}
			indexExists = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/indexes/articles/settings/"):
			settings = append(settings, r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && r.URL.Path == "/indexes/articles/documents":
			if r.URL.Query().Get("fields") != "id" {
				t.Errorf("document list fields = %q, want id", r.URL.Query().Get("fields"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"results":[{"id":7},{"id":8}],"total":2}`)
		case r.Method == http.MethodPost && r.URL.Path == "/indexes/articles/documents/delete-batch":
			if err := json.NewDecoder(r.Body).Decode(&deleted); err != nil {
				t.Errorf("decode stale IDs: %v", err)
			}
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodPost && r.URL.Path == "/indexes/articles/documents":
			if err := json.NewDecoder(r.Body).Decode(&upserted); err != nil {
				t.Errorf("decode documents: %v", err)
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL, ApiKey: "test-key"})
	documents := []port.ArticleSearch{
		{Id: 7, ArticleTitle: "Current article", Status: 1, ModerationStatus: "visible"},
		{Id: 9, ArticleTitle: "New article", Status: 1, ModerationStatus: "visible"},
	}
	if err := searcher.Reconcile(context.Background(), documents); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if len(settings) != 3 {
		t.Fatalf("index settings calls = %#v, want searchable/filterable/sortable", settings)
	}
	if len(deleted) != 1 || deleted[0] != "8" {
		t.Fatalf("stale document IDs = %#v, want [8]", deleted)
	}
	if len(upserted) != 2 || upserted[0].Id != 7 || upserted[1].Id != 9 {
		t.Fatalf("upserted snapshot = %#v", upserted)
	}
}

func TestMeiliSearcherHTTPReconcileReturnsSettingsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/indexes" {
			w.WriteHeader(http.StatusCreated)
			return
		}
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/indexes/articles/settings/") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"message":"filterable settings rejected"}`)
			return
		}
		t.Errorf("unexpected request after settings failure: %s %s", r.Method, r.URL.String())
		http.NotFound(w, r)
	}))
	defer server.Close()

	searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL})
	err := searcher.Reconcile(context.Background(), nil)
	if !errors.IsKind(err, errors.KindUnavailable) || !strings.Contains(err.Error(), "filterable settings rejected") {
		t.Fatalf("unexpected reconcile error: %v", err)
	}
}
