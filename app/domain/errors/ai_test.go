package errors

import (
	"context"
	"errors"
	"testing"
)

func TestAIErrorKeepsProviderDetailsOutOfPublicErrorText(t *testing.T) {
	err := NewAI(AICodeProviderUnavailable, "model.call", errors.New("api_key=secret"))
	if got := err.Error(); got != "model.call: ai_provider_unavailable" {
		t.Fatalf("public error = %q", got)
	}
	if !IsAICode(err, AICodeProviderUnavailable) {
		t.Fatalf("error code was not preserved: %v", err)
	}
}

func TestWrapAIUnavailablePreservesContextErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "canceled", err: context.Canceled},
		{name: "deadline", err: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			wrapped := WrapAIUnavailable("embedding.embed", test.err)
			if wrapped != test.err {
				t.Fatalf("WrapAIUnavailable() = %v, want original context error", wrapped)
			}
			if !errors.Is(wrapped, test.err) {
				t.Fatalf("context error was not preserved: %v", wrapped)
			}
		})
	}
}

func TestWrapAIUnavailableClassifiesProviderErrors(t *testing.T) {
	providerErr := errors.New("provider offline")
	wrapped := WrapAIUnavailable("embedding.embed", providerErr)
	if !IsAICode(wrapped, AICodeProviderUnavailable) {
		t.Fatalf("AI code = %q, want %q", AICodeOf(wrapped), AICodeProviderUnavailable)
	}
	if !errors.Is(wrapped, providerErr) {
		t.Fatalf("provider error was not retained: %v", wrapped)
	}
}
