package repository

import (
	"strings"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestAIReviewInsertStatementMatchesReviewColumnCount(t *testing.T) {
	const expectedColumns = 25
	if got := strings.Count(aiReviewInsertSQL, "?"); got != expectedColumns {
		t.Fatalf("AI review insert has %d placeholders, want %d", got, expectedColumns)
	}
}

func TestNormalizeAIReviewFillsSafeDefaultsAndCopiesTimes(t *testing.T) {
	created := time.Date(2026, 8, 29, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	review, err := normalizeAIReview(port.AIReview{
		TargetType: "article",
		TargetID:   "42",
		Operation:  "polish",
		Content:    "preview",
		RunID:      "run-1",
		ReviewerID: "7",
		CreatedAt:  created,
	})
	if err != nil {
		t.Fatal(err)
	}
	if review.ID == "" || review.Status != port.ReviewPending || review.CreatedAt.Location() != time.UTC || review.UpdatedAt.IsZero() || review.ContentDigest != port.ReviewContentDigest("preview") || review.BoundAt.IsZero() {
		t.Fatalf("normalized review = %+v", review)
	}
}

func TestNormalizeAIReviewBindingRejectsUnsafeValues(t *testing.T) {
	base := port.AIReview{TargetType: "article", TargetID: "42", Operation: "polish", Content: "preview", RunID: "run", ReviewerID: "7"}
	for index, review := range []port.AIReview{
		func() port.AIReview {
			value := base
			value.SessionID = "bad\nvalue"
			return value
		}(),
		func() port.AIReview {
			value := base
			value.ContentDigest = "not-a-digest"
			return value
		}(),
	} {
		if _, err := normalizeAIReview(review); err == nil {
			t.Fatalf("case %d: unsafe review binding was accepted", index)
		}
	}
}

func TestNormalizeAIReviewRejectsIncompleteOrFinalizedReview(t *testing.T) {
	tests := []port.AIReview{
		{TargetType: "article", TargetID: "42", Operation: "polish", Content: "preview"},
		{TargetType: "article", TargetID: "42", Operation: "polish", Content: "preview", RunID: "run", ReviewerID: "7", Status: port.ReviewApproved},
		{TargetType: "article", TargetID: "42", Operation: "polish", Content: string(make([]byte, 100001)), RunID: "run", ReviewerID: "7"},
	}
	for index, review := range tests {
		if _, err := normalizeAIReview(review); err == nil {
			t.Fatalf("case %d: expected validation error", index)
		}
	}
}

func TestNormalizeAIReviewActionRequiresStableIdentity(t *testing.T) {
	if _, err := normalizeAIReviewAction(port.AIReviewAction{Action: port.ReviewActionAccepted, ActorID: "7"}, "review-1"); err == nil {
		t.Fatal("action without idempotency key was accepted")
	}
	if _, err := normalizeAIReviewAction(port.AIReviewAction{Action: port.ReviewActionCreated, ActorID: "7", IdempotencyKey: "key"}, "review-1"); err == nil {
		t.Fatal("reserved created action was accepted")
	}
	action, err := normalizeAIReviewAction(port.AIReviewAction{
		Action:         port.ReviewActionAccepted,
		ActorID:        "7",
		IdempotencyKey: "review-1:accepted:run-1",
		RunID:          "run-1",
	}, "review-1")
	if err != nil || action.ID == "" || action.ReviewID != "review-1" {
		t.Fatalf("normalized action=%+v error=%v", action, err)
	}
}

func TestValidateReviewActionBindingMatchesSessionTargetDigestAndExpiry(t *testing.T) {
	expires := time.Now().UTC().Add(time.Hour)
	review := aiReviewRow{
		TargetType:    "article",
		TargetID:      "42",
		SessionID:     "admin:7",
		ContentDigest: port.ReviewContentDigest("preview"),
		ExpiresAt:     &expires,
	}
	action := port.AIReviewAction{
		SessionID:     "admin:7",
		TargetType:    "article",
		TargetID:      "42",
		ContentDigest: port.ReviewContentDigest("preview"),
		ExpiresAt:     expires,
	}
	if err := validateReviewActionBinding(review, action, time.Now().UTC(), "test"); err != nil {
		t.Fatal(err)
	}
	action.ContentDigest = port.ReviewContentDigest("changed")
	if err := validateReviewActionBinding(review, action, time.Now().UTC(), "test"); err == nil {
		t.Fatal("changed content digest was accepted")
	}
	old := expires.Add(-2 * time.Hour)
	review.ExpiresAt = &old
	if err := validateReviewActionBinding(review, action, time.Now().UTC(), "test"); err == nil {
		t.Fatal("expired review binding was accepted")
	}
}

func TestNormalizeReviewFilterWhitelistsStatusAndBoundsPage(t *testing.T) {
	filter, err := normalizeReviewFilter(port.ReviewFilter{Status: port.ReviewPending, Size: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if filter.Current != 1 || filter.Size != 100 {
		t.Fatalf("normalized filter=%+v", filter)
	}
	if _, err := normalizeReviewFilter(port.ReviewFilter{Status: "unknown"}); err == nil {
		t.Fatalf("unknown status error=%v", err)
	}
}
