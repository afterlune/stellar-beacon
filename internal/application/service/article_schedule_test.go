package service

import (
	"testing"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
)

func TestNormalizeScheduledAtRequiresFutureTimeForScheduledArticles(t *testing.T) {
	if _, err := normalizeScheduledAt(4, time.Now().Add(-time.Minute).Format(time.RFC3339)); err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("a past release time must be rejected, got %v", err)
	}

	if _, err := normalizeScheduledAt(4, ""); err == nil {
		t.Fatal("a scheduled article needs a release time")
	}

	future := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	parsed, err := normalizeScheduledAt(4, future.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("a future release time must be accepted: %v", err)
	}
	if !parsed.Equal(future) {
		t.Fatalf("release time must round-trip as RFC3339: %q", parsed.Format(time.RFC3339))
	}
}

func TestNormalizeScheduledAtClearsOtherStatuses(t *testing.T) {
	parsed, err := normalizeScheduledAt(1, time.Now().Add(time.Hour).Format(time.RFC3339))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !parsed.IsZero() {
		t.Fatalf("published articles must not keep a release time: %q", parsed)
	}
}
