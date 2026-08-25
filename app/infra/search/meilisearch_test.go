package search

import (
	"benetnasch/app/domain/errors"
	"context"
	"testing"
)

func TestMeiliSearcherSearchRequiresConfiguredClient(t *testing.T) {
	_, err := (&MeiliSearcher{}).Search(context.Background(), "keyword")
	if !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("Search() error kind = %v, want %v", errors.KindOf(err), errors.KindUnavailable)
	}
}
