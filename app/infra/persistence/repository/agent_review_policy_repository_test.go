package repository

import (
	"context"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestAgentReviewPolicyRepositoryRequiresDatabase(t *testing.T) {
	ctx := context.Background()
	repository := NewAgentReviewPolicyRepository(nil)
	if _, err := repository.Get(ctx, port.DefaultAgentReviewPolicyID); err == nil {
		t.Fatal("Get() unexpectedly succeeded without database")
	}
	if err := repository.Save(ctx, port.AgentReviewPolicy{ReviewRequired: true}); err == nil {
		t.Fatal("Save() unexpectedly succeeded without database")
	}
}

func TestAgentReviewPolicyRowNormalizesAndRejectsUnsafeValues(t *testing.T) {
	row := agentReviewPolicyRow{
		ID:                  "default",
		Version:             2,
		ReviewRequired:      true,
		ReviewTTLSeconds:    int64((24 * time.Hour) / time.Second),
		MaxCandidateRunes:   500,
		SimilarityThreshold: 0.82,
		DailyLimit:          3,
		PerArticleLimit:     1,
		PerActionLimit:      3,
		AllowedActions:      `["comment","talk"]`,
		SensitivePatterns:   `["secret"]`,
		UpdatedAt:           time.Now().UTC(),
	}
	policy, err := row.policy()
	if err != nil {
		t.Fatal(err)
	}
	if policy.Version != 2 || policy.ReviewTTL != 24*time.Hour || len(policy.AllowedActions) != 2 {
		t.Fatalf("policy = %+v", policy)
	}

	row.ReviewRequired = false
	if _, err := row.policy(); err == nil {
		t.Fatal("policy with review bypass unexpectedly accepted")
	}
	row.ReviewRequired = true
	row.AllowedActions = "not-json"
	if _, err := row.policy(); err == nil {
		t.Fatal("malformed actions unexpectedly accepted")
	}
	row.AllowedActions = `[]`
	row.ReviewTTLSeconds = int64(^uint64(0) >> 1)
	if _, err := row.policy(); err == nil {
		t.Fatal("overflowing ttl unexpectedly accepted")
	}
}
