package repository

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"strings"
	"testing"
)

func TestAgentProfileRepositoryRequiresDatabase(t *testing.T) {
	repository := NewAgentProfileRepository(nil)
	if _, err := repository.Get(context.Background(), port.DefaultAgentProfileID); !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("Get() error kind = %v, want unavailable", errors.KindOf(err))
	}
	if err := repository.Save(context.Background(), port.AgentProfile{
		ID:            port.DefaultAgentProfileID,
		Name:          "Benetnasch",
		PromptVersion: "v1",
		SystemPrompt:  "保持公开资料边界",
	}); !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("Save() error kind = %v, want unavailable", errors.KindOf(err))
	}
}

func TestNormalizeAgentProfileAppliesDatabaseDefaultsAndRejectsUnsafeText(t *testing.T) {
	profile, err := normalizeAgentProfile(port.AgentProfile{
		ID:            " profile ",
		Name:          " Benetnasch ",
		PromptVersion: " v1 ",
		SystemPrompt:  " 保持公开边界 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID != "profile" || profile.Name != "Benetnasch" || profile.SystemPromptRef == "" || profile.UpdatedAt.IsZero() {
		t.Fatalf("normalized profile = %+v", profile)
	}

	unsafe := port.AgentProfile{
		ID:            "profile",
		Name:          "name",
		PromptVersion: "v1",
		SystemPrompt:  "bad\x00prompt",
	}
	if _, err := normalizeAgentProfile(unsafe); err == nil || !errors.IsKind(err, errors.KindValidation) {
		t.Fatalf("unsafe profile error = %v, want validation", err)
	}

	tooLong := strings.Repeat("字", 65)
	if _, err := normalizeAgentProfile(port.AgentProfile{
		ID:            "profile",
		Name:          tooLong,
		PromptVersion: "v1",
		SystemPrompt:  "prompt",
	}); err == nil || !errors.IsKind(err, errors.KindValidation) {
		t.Fatalf("oversized profile error = %v, want validation", err)
	}
}
