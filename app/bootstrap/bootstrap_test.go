package bootstrap

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"context"
	"testing"
)

func TestNewAIComponentsAreDisabledByDefault(t *testing.T) {
	components := NewAIComponents()
	if components.Router == nil || components.Metrics == nil {
		t.Fatal("AI composition root returned nil components")
	}
	if components.Router.Diagnostics()["enabled"] != false {
		t.Fatal("AI components must remain disabled unless explicitly configured")
	}
}

func TestResolveArticleIndexBackfillRunID(t *testing.T) {
	tests := []struct {
		name         string
		runID        string
		indexVersion string
		want         string
		wantErr      bool
	}{
		{name: "explicit", runID: "  migration-2026-08  ", indexVersion: "v1", want: "migration-2026-08"},
		{name: "default", indexVersion: "V2", want: "article-index-backfill-v2"},
		{name: "invalid default version", indexVersion: "bad/version", wantErr: true},
		{name: "oversized explicit ID", runID: string(make([]byte, 129)), indexVersion: "v1", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveArticleIndexBackfillRunID(test.runID, test.indexVersion)
			if (err != nil) != test.wantErr {
				t.Fatalf("resolveArticleIndexBackfillRunID() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("resolveArticleIndexBackfillRunID() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestConfigureAgentProfileRepositoryIsOptIn(t *testing.T) {
	repository, err := configureAgentProfileRepository(nil, config.AIAgentProfileSettings{
		ID:            port.DefaultAgentProfileID,
		Name:          "Benetnasch",
		PromptVersion: "v1",
		SystemPrompt:  "保持公开资料边界",
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := repository.Get(context.Background(), port.DefaultAgentProfileID)
	if err != nil || profile.Name != "Benetnasch" {
		t.Fatalf("configuration-backed profile = %+v, error=%v", profile, err)
	}

	_, err = configureAgentProfileRepository(nil, config.AIAgentProfileSettings{
		ID:                 port.DefaultAgentProfileID,
		PersistenceEnabled: true,
	})
	if err == nil || !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("persistent profile without database error = %v, want unavailable", err)
	}
}

func TestConfigureAgentReviewPolicyRepositoryIsOptIn(t *testing.T) {
	repository, err := configureAgentReviewPolicyRepository(nil, config.AIAgentReviewPolicySettings{ID: port.DefaultAgentReviewPolicyID}, config.AIAgentBehaviorSettings{})
	if err != nil {
		t.Fatal(err)
	}
	policy, err := repository.Get(context.Background(), port.DefaultAgentReviewPolicyID)
	if err != nil || !policy.ReviewRequired {
		t.Fatalf("configuration-backed policy = %+v, error=%v", policy, err)
	}

	_, err = configureAgentReviewPolicyRepository(nil, config.AIAgentReviewPolicySettings{
		ID:                 port.DefaultAgentReviewPolicyID,
		PersistenceEnabled: true,
	}, config.AIAgentBehaviorSettings{})
	if err == nil || !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("persistent policy without database error = %v, want unavailable", err)
	}
}
