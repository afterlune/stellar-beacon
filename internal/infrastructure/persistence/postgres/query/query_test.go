package pgsql

import "testing"

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
