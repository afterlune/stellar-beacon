package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
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
		_, _ = io.WriteString(w, `{"hits":[{"id":147,"articleTitle":"title","articleContent":"content","isDelete":0,"status":1,"_formatted":{"articleTitle":"<mark>title</mark>"}}],"estimatedTotalHits":12}`)
	}))
	defer server.Close()

	searcher := NewMeiliSearcher(&config.MeiliSearch{URL: server.URL, ApiKey: "test-key"})
	page, err := searcher.Search(context.Background(), "keyword", 5, 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if page.Total != 12 || len(page.Hits) != 1 || page.Hits[0].Id != 147 || page.Hits[0].HighlightedTitle == "" {
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
