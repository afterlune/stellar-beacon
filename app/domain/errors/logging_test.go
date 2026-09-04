package errors

import (
	"context"
	stderrors "errors"
	"testing"
)

func TestSafeCodeNeverUsesErrorDetails(t *testing.T) {
	secret := stderrors.New("provider body prompt=private-content api_key=secret")
	if got := SafeCode(secret); got != "internal_error" {
		t.Fatalf("SafeCode(generic) = %q, want internal_error", got)
	}
	if got := SafeCode(NewAI(AICodeProviderUnavailable, "provider.call", secret)); got != string(AICodeProviderUnavailable) {
		t.Fatalf("SafeCode(AI) = %q, want %q", got, AICodeProviderUnavailable)
	}
	if got := SafeCode(Wrap(KindValidation, "request.bind", secret)); got != "validation_error" {
		t.Fatalf("SafeCode(validation) = %q, want validation_error", got)
	}
	if got := SafeCode(context.Canceled); got != "context_canceled" {
		t.Fatalf("SafeCode(canceled) = %q, want context_canceled", got)
	}
}

func TestSafeCodeHandlesNil(t *testing.T) {
	if got := SafeCode(nil); got != "" {
		t.Fatalf("SafeCode(nil) = %q, want empty string", got)
	}
}
