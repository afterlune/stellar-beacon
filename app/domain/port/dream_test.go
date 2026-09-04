package port

import (
	"strings"
	"testing"
	"time"
)

func TestDreamTaskPayloadValidatesIdentityAndUniqueSources(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	data, err := NewDreamTaskPayload([]int{7, 8}, "benetnasch-public", "v1", "agent-dream:key", now)
	if err != nil || len(data) == 0 {
		t.Fatalf("NewDreamTaskPayload() data=%q err=%v", data, err)
	}
	for _, invalid := range []DreamTaskPayload{
		{SchemaVersion: 1, SeedArticleIDs: []int{7, 7}, ProfileID: "profile", PromptVersion: "v1", IdempotencyKey: "key", OccurredAt: now},
		{SchemaVersion: 1, SeedArticleIDs: nil, ProfileID: "profile", PromptVersion: "v1", IdempotencyKey: "key", OccurredAt: now},
		{SchemaVersion: 1, SeedArticleIDs: []int{7}, ProfileID: "profile", PromptVersion: "v1", IdempotencyKey: "key", OccurredAt: time.Time{}},
	} {
		if err := invalid.Validate(); err == nil {
			t.Fatalf("invalid payload was accepted: %+v", invalid)
		}
	}
	if _, err := NewDreamTaskPayload([]int{1}, "profile", "v1", strings.Repeat("x", 256), now); err == nil {
		t.Fatal("oversized dream idempotency key was accepted")
	}
}
