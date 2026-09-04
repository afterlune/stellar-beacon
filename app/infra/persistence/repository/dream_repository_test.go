package repository

import (
	"strings"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestNormalizeDreamEntryBoundsPrivateMetadata(t *testing.T) {
	created := time.Date(2026, 8, 29, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	entry, encoded, err := normalizeDreamEntry(port.DreamEntry{
		ID:               "dream-1",
		ReviewID:         "review-1",
		Title:            "夜航",
		Content:          "一段经过审核的梦境。",
		ImagePrompt:      "quiet night",
		SourceArticleIDs: []int{7, 8},
		CreatedAt:        created,
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Status != port.DreamPendingReview || entry.ImageStatus != port.DreamImagePending || entry.CreatedAt.Location() != time.UTC || encoded != "[7,8]" {
		t.Fatalf("entry=%+v encoded=%s", entry, encoded)
	}
	if _, _, err := normalizeDreamEntry(port.DreamEntry{
		ID: "dream-2", ReviewID: "review-2", Title: "bad", Content: "content", SourceArticleIDs: []int{7, 7},
	}); err == nil {
		t.Fatal("duplicate source article ids were accepted")
	}
	if _, _, err := normalizeDreamEntry(port.DreamEntry{
		ID: "dream-3", ReviewID: "review-3", Title: strings.Repeat("x", 256), Content: "content", SourceArticleIDs: []int{7},
	}); err == nil {
		t.Fatal("oversized dream title was accepted")
	}
}

func TestValidateDreamImageResultRequiresSafeTerminalValues(t *testing.T) {
	if err := validateDreamImageResult(port.DreamImageReady, "https://cdn.example/dream.png", ""); err != nil {
		t.Fatal(err)
	}
	if err := validateDreamImageResult(port.DreamImagePlaceholder, "/dream-placeholder.svg", "provider unavailable"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		status port.DreamImageStatus
		url    string
		err    string
	}{
		{status: port.DreamImageReady, url: "javascript:alert(1)"},
		{status: port.DreamImagePlaceholder, url: "/placeholder.svg"},
		{status: port.DreamImageFailed, err: ""},
	} {
		if err := validateDreamImageResult(test.status, test.url, test.err); err == nil {
			t.Fatalf("invalid image result was accepted: %+v", test)
		}
	}
}
