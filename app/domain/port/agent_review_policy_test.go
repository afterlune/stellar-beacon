package port

import (
	"math"
	"testing"
	"time"
)

func TestNormalizeAgentReviewPolicyAppliesDefaultsAndCopiesSlices(t *testing.T) {
	input := AgentReviewPolicy{ReviewRequired: true, SensitivePatterns: []string{"Custom", "custom"}}
	policy, err := NormalizeAgentReviewPolicy(input)
	if err != nil {
		t.Fatal(err)
	}
	if policy.ID != DefaultAgentReviewPolicyID || policy.Version != 0 || policy.ReviewTTL != DefaultAgentReviewTTL || len(policy.AllowedActions) != 3 {
		t.Fatalf("policy = %+v", policy)
	}
	input.SensitivePatterns[0] = "changed"
	if policy.SensitivePatterns[len(policy.SensitivePatterns)-1] != "custom" {
		t.Fatalf("sensitive patterns were not normalized: %+v", policy.SensitivePatterns)
	}
}

func TestNormalizeAgentReviewPolicyKeepsHumanReviewMandatory(t *testing.T) {
	if _, err := NormalizeAgentReviewPolicy(AgentReviewPolicy{}); err == nil {
		t.Fatal("review bypass was accepted")
	}
	if _, err := NormalizeAgentReviewPolicy(AgentReviewPolicy{ReviewRequired: true, ReviewTTL: 31 * 24 * time.Hour}); err == nil {
		t.Fatal("overlong review ttl was accepted")
	}
	if _, err := NormalizeAgentReviewPolicy(AgentReviewPolicy{ReviewRequired: true, SimilarityThreshold: math.NaN()}); err == nil {
		t.Fatal("NaN similarity threshold was accepted")
	}
}
