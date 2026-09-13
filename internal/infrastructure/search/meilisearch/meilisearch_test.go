package search

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMeiliSearcherSearchRequiresConfiguredClient(t *testing.T) {
	_, err := (&MeiliSearcher{}).Search(context.Background(), "keyword")
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
		if strings.Contains(string(body), "hybrid") {
			t.Fatalf("legacy-incompatible hybrid field was sent: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"hits":[{"id":147,"articleTitle":"title","articleContent":"content","isDelete":0,"status":1,"_formatted":{"articleTitle":"<mark>title</mark>"}}]}`)
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
