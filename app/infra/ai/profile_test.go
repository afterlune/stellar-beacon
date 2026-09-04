package ai

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"context"
	"testing"
	"time"
)

func TestConfiguredAgentProfileRepositoryProvidesVersionedProfileAndFallbacks(t *testing.T) {
	repository := NewConfiguredAgentProfileRepository(config.AIAgentProfileSettings{ID: "public-v2", PromptVersion: "v2"})
	profile, err := repository.Get(context.Background(), "public-v2")
	if err != nil {
		t.Fatal(err)
	}
	if profile.PromptVersion != "v2" || profile.SystemPrompt == "" || profile.Opening == "" || profile.SystemPromptRef == "" || len(profile.RhythmPrompts) != 3 {
		t.Fatalf("profile = %#v, want versioned prompt and opening", profile)
	}
	if _, err := repository.Get(context.Background(), "missing"); err == nil {
		t.Fatal("missing profile unexpectedly found")
	}
	if err := repository.Save(context.Background(), port.AgentProfile{ID: "public-v3", Name: "B", PromptVersion: "v3", SystemPrompt: "prompt"}); err != nil {
		t.Fatal(err)
	}
	current, err := repository.Get(context.Background(), "public-v3")
	if err != nil || current.PromptVersion != "v3" {
		t.Fatalf("saved profile = %#v, error=%v", current, err)
	}
}

func TestConfiguredAgentReviewPolicyRepositoryProvidesSafeDefaultsAndVersioning(t *testing.T) {
	repository := NewConfiguredAgentReviewPolicyRepository(
		config.AIAgentReviewPolicySettings{ID: "default"},
		config.AIAgentBehaviorSettings{},
	)
	policy, err := repository.Get(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if !policy.ReviewRequired || policy.Version != 1 || policy.ReviewTTL <= 0 || len(policy.AllowedActions) != 3 {
		t.Fatalf("policy = %+v, want safe defaults", policy)
	}
	policy.ReviewTTL = 2 * time.Hour
	if err := repository.Save(context.Background(), policy); err != nil {
		t.Fatal(err)
	}
	updated, err := repository.Get(context.Background(), "default")
	if err != nil || updated.Version != 2 || updated.ReviewTTL != 2*time.Hour {
		t.Fatalf("updated policy = %+v, error=%v", updated, err)
	}
	policy.Version = 1
	if err := repository.Save(context.Background(), policy); err == nil {
		t.Fatal("stale policy update unexpectedly succeeded")
	}
}
