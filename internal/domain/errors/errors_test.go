package errors

import (
	"errors"
	"testing"
)

func TestErrorClassificationAndUnwrap(t *testing.T) {
	root := errors.New("database unavailable")
	err := Wrap(KindUnavailable, "article.list", root)
	if !IsKind(err, KindUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
	if !errors.Is(err, root) {
		t.Fatalf("expected wrapped root error")
	}
	if got := Op(err); got != "article.list" {
		t.Fatalf("unexpected operation: %q", got)
	}
}

func TestNotFoundIsDistinctFromInternal(t *testing.T) {
	if IsKind(NotFound("article.get"), KindInternal) {
		t.Fatal("not found must not be classified as internal")
	}
	if KindOf(errors.New("plain error")) != KindInternal {
		t.Fatal("plain errors should default to internal")
	}
}
