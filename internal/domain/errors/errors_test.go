package errors

import (
	"context"
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

// A transport error produced by a disconnected caller says nothing about
// service health, so wrapping it must not surface as unavailable.
func TestCanceledContextIsNotAnInfrastructureFailure(t *testing.T) {
	err := Wrap(KindUnavailable, "article.list", context.Canceled)
	if !IsKind(err, KindCanceled) {
		t.Fatalf("expected canceled kind, got %v", KindOf(err))
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatal("canceled cause must stay unwrappable")
	}
	if got := Op(err); got != "article.list" {
		t.Fatalf("unexpected operation: %q", got)
	}
	if !IsKind(Unavailable("article.list", context.Canceled), KindCanceled) {
		t.Fatal("Unavailable must reclassify a canceled cause")
	}
	if !IsKind(Canceled("article.list", nil), KindCanceled) {
		t.Fatal("Canceled constructor must produce a canceled kind")
	}
}

// DeadlineExceeded is deliberately kept as a real failure: without a request
// level timeout it can only come from infrastructure.
func TestDeadlineExceededStaysUnavailable(t *testing.T) {
	if !IsKind(Unavailable("article.list", context.DeadlineExceeded), KindUnavailable) {
		t.Fatal("deadline exceeded must stay unavailable")
	}
}
