package service

import (
	"testing"
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
)

func TestNormalizeScheduledAtRequiresFutureTimeForScheduledArticles(t *testing.T) {
	vo := model.ArticleVO{Status: 4, ScheduledAt: time.Now().Add(-time.Minute).Format(time.RFC3339)}
	if err := normalizeScheduledAt(&vo); err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("a past release time must be rejected, got %v", err)
	}

	vo = model.ArticleVO{Status: 4}
	if err := normalizeScheduledAt(&vo); err == nil {
		t.Fatal("a scheduled article needs a release time")
	}

	future := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	vo = model.ArticleVO{Status: 4, ScheduledAt: future.Format(time.RFC3339)}
	if err := normalizeScheduledAt(&vo); err != nil {
		t.Fatalf("a future release time must be accepted: %v", err)
	}
	if parsed, err := time.Parse(time.RFC3339, vo.ScheduledAt); err != nil || !parsed.Equal(future) {
		t.Fatalf("release time must round-trip as RFC3339: %q", vo.ScheduledAt)
	}
}

func TestNormalizeScheduledAtClearsOtherStatuses(t *testing.T) {
	vo := model.ArticleVO{Status: 1, ScheduledAt: time.Now().Add(time.Hour).Format(time.RFC3339)}
	if err := normalizeScheduledAt(&vo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vo.ScheduledAt != "" {
		t.Fatalf("published articles must not keep a release time: %q", vo.ScheduledAt)
	}
}
