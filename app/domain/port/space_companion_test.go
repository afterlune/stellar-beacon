package port

import (
	"testing"
	"time"
)

func TestSpacePublicationInputNormalizeAllowsOnlyResidentTypes(t *testing.T) {
	base := SpacePublicationInput{
		Type:            "status",
		Title:           "今天的状态",
		Body:            "在空间里整理了一会儿资料。",
		SourceSessionID: "session-1",
		SourceRunID:     "run-1",
		IdempotencyKey:  "run-1-status-1",
	}
	publication, err := base.Normalize(MoonfeiPrincipalID, testSpaceTime())
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if publication.AgentID != MoonfeiPrincipalID || publication.Type != SpaceContentStatus {
		t.Fatalf("normalized publication = %#v", publication)
	}

	for _, contentType := range []string{"article", "video", "comment", ""} {
		input := base
		input.Type = contentType
		if _, err := input.Normalize(MoonfeiPrincipalID, testSpaceTime()); err == nil {
			t.Errorf("Normalize(%q) unexpectedly succeeded", contentType)
		}
	}
}

func TestSpacePublicationInputNormalizeRejectsUnsafeMediaURLAndMissingSource(t *testing.T) {
	base := SpacePublicationInput{
		Type:            "dream",
		Title:           "梦",
		Body:            "一段梦境",
		SourceSessionID: "session-1",
		SourceRunID:     "run-1",
		IdempotencyKey:  "dream-1",
	}
	unsafe := base
	unsafe.MediaURL = "http://example.com/image.png"
	if _, err := unsafe.Normalize(MoonfeiPrincipalID, testSpaceTime()); err == nil {
		t.Fatal("plain HTTP media URL unexpectedly succeeded")
	}
	missingSource := base
	missingSource.SourceRunID = ""
	if _, err := missingSource.Normalize(MoonfeiPrincipalID, testSpaceTime()); err == nil {
		t.Fatal("missing source run unexpectedly succeeded")
	}
}

func TestSpacePrincipalValidateForRequiresAgentAndScope(t *testing.T) {
	valid := SpacePrincipal{ID: MoonfeiPrincipalID, Type: SpacePrincipalAgent, Scopes: []string{SpaceScopeRead}}
	if err := valid.ValidateFor(SpaceScopeRead); err != nil {
		t.Fatalf("valid principal rejected: %v", err)
	}
	for _, principal := range []SpacePrincipal{
		{ID: MoonfeiPrincipalID, Type: "user", Scopes: []string{SpaceScopeRead}},
		{ID: MoonfeiPrincipalID, Type: SpacePrincipalAgent, Scopes: nil},
		{ID: "", Type: SpacePrincipalAgent, Scopes: []string{SpaceScopeRead}},
	} {
		if err := principal.ValidateFor(SpaceScopeRead); err == nil {
			t.Errorf("invalid principal unexpectedly accepted: %#v", principal)
		}
	}
}

func testSpaceTime() (t time.Time) {
	return time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
}
