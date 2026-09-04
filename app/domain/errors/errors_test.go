package errors

import (
	"context"
	stderrors "errors"
	"testing"
)

func TestWrapUnavailablePreservesContextErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "canceled", err: context.Canceled},
		{name: "deadline", err: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			wrapped := WrapUnavailable("cache.read", test.err)
			if wrapped != test.err {
				t.Fatalf("WrapUnavailable() = %v, want original context error", wrapped)
			}
			if !stderrors.Is(wrapped, test.err) {
				t.Fatalf("context error was not preserved: %v", wrapped)
			}
		})
	}
}

func TestWrapUnavailableClassifiesInfrastructureErrors(t *testing.T) {
	backendErr := stderrors.New("redis unavailable")
	wrapped := WrapUnavailable("cache.read", backendErr)
	if !IsKind(wrapped, KindUnavailable) {
		t.Fatalf("kind = %q, want %q", KindOf(wrapped), KindUnavailable)
	}
	if !stderrors.Is(wrapped, backendErr) {
		t.Fatalf("backend error was not retained: %v", wrapped)
	}
}

func TestWrapUnavailableHandlesNil(t *testing.T) {
	if got := WrapUnavailable("cache.read", nil); got != nil {
		t.Fatalf("WrapUnavailable(nil) = %v, want nil", got)
	}
}

func TestUnavailableConstructorsPreserveContextErrors(t *testing.T) {
	constructors := []struct {
		name string
		make func(error) error
	}{
		{name: "wrap", make: func(err error) error { return Wrap(KindUnavailable, "database.read", err) }},
		{name: "unavailable", make: func(err error) error { return Unavailable("database.read", err) }},
		{name: "wrap unavailable", make: func(err error) error { return WrapUnavailable("database.read", err) }},
	}
	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			if err := constructor.make(context.Canceled); !stderrors.Is(err, context.Canceled) {
				t.Fatalf("constructor error = %v, want context canceled", err)
			}
		})
	}
}
