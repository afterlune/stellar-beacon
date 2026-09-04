package service

import (
	"context"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

type fakeSpacePublicationRepository struct {
	items []port.SpacePublication
}

func (f *fakeSpacePublicationRepository) Create(_ context.Context, input port.SpacePublication) (port.SpacePublication, bool, error) {
	for _, item := range f.items {
		if item.IdempotencyKey != input.IdempotencyKey {
			continue
		}
		if item.RequestDigest != input.RequestDigest {
			return port.SpacePublication{}, false, apperrors.Conflict("test.space.publication", "digest mismatch")
		}
		return item, true, nil
	}
	f.items = append(f.items, input)
	return input, false, nil
}

func (f *fakeSpacePublicationRepository) GetPublic(_ context.Context, contentType port.SpaceContentType, id string) (port.SpacePublication, error) {
	for _, item := range f.items {
		if item.Type == contentType && item.ID == id {
			return item, nil
		}
	}
	return port.SpacePublication{}, apperrors.NotFound("test.space.publication")
}

func (f *fakeSpacePublicationRepository) ListPublic(_ context.Context, query port.SpacePublicationQuery) ([]port.SpacePublication, int, error) {
	items := make([]port.SpacePublication, 0, len(f.items))
	for _, item := range f.items {
		if query.Type != "" && string(item.Type) != query.Type {
			continue
		}
		items = append(items, item)
	}
	return items, len(items), nil
}

func TestSpaceCompanionPublishIsAllowlistedAndIdempotent(t *testing.T) {
	repository := &fakeSpacePublicationRepository{}
	service, err := NewSpaceCompanionService(SpaceCompanionServiceDeps{
		Publications: repository,
		AgentID:      port.MoonfeiPrincipalID,
		Enabled:      true,
		Publish:      true,
		Now:          func() time.Time { return time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("NewSpaceCompanionService() error = %v", err)
	}
	principal := port.SpacePrincipal{ID: port.MoonfeiPrincipalID, Type: port.SpacePrincipalAgent, Scopes: []string{port.SpaceScopePublish}}
	input := port.SpacePublicationInput{
		Type:            "status",
		Title:           "今日状态",
		Body:            "完成了空间边界的第一步。",
		SourceSessionID: "session-1",
		SourceRunID:     "run-1",
		IdempotencyKey:  "run-1-status-1",
	}
	first, err := service.Publish(context.Background(), principal, input)
	if err != nil {
		t.Fatalf("first Publish() error = %v", err)
	}
	if first.Existing || first.Publication.ID == "" {
		t.Fatalf("first Publish() result = %#v", first)
	}
	second, err := service.Publish(context.Background(), principal, input)
	if err != nil {
		t.Fatalf("retry Publish() error = %v", err)
	}
	if !second.Existing || second.Publication.ID != first.Publication.ID || len(repository.items) != 1 {
		t.Fatalf("idempotent retry = %#v, stored=%d", second, len(repository.items))
	}

	input.Type = "article"
	if _, err := service.Publish(context.Background(), principal, input); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("disallowed publication error kind = %v", apperrors.KindOf(err))
	}

	wrongPrincipal := principal
	wrongPrincipal.ID = "other-agent"
	if _, err := service.Publish(context.Background(), wrongPrincipal, input); !apperrors.IsKind(err, apperrors.KindForbidden) {
		t.Fatalf("wrong principal error kind = %v", apperrors.KindOf(err))
	}
}

func TestSpaceCompanionSearchOnlyReturnsRequestedPublicPublicationTypes(t *testing.T) {
	repository := &fakeSpacePublicationRepository{}
	service, err := NewSpaceCompanionService(SpaceCompanionServiceDeps{
		Publications: repository,
		AgentID:      port.MoonfeiPrincipalID,
		Enabled:      true,
		Publish:      true,
		Now:          func() time.Time { return time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("NewSpaceCompanionService() error = %v", err)
	}
	principal := port.SpacePrincipal{ID: port.MoonfeiPrincipalID, Type: port.SpacePrincipalAgent, Scopes: []string{port.SpaceScopePublish}}
	for _, typ := range []string{"status", "dream"} {
		if _, err := service.Publish(context.Background(), principal, port.SpacePublicationInput{
			Type:            typ,
			Title:           typ + " entry",
			Body:            "public text",
			SourceSessionID: "session-" + typ,
			SourceRunID:     "run-" + typ,
			IdempotencyKey:  "key-" + typ,
		}); err != nil {
			t.Fatalf("Publish(%q) error = %v", typ, err)
		}
	}
	result, err := service.Search(context.Background(), port.SpaceSearchQuery{Query: "public", Types: []string{"status"}, Limit: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Count != 1 || len(result.Items) != 1 || result.Items[0].Type != port.SpaceContentStatus {
		t.Fatalf("Search() result = %#v", result)
	}
}

func TestDisabledSpaceCompanionDoesNotExposeCapabilitiesAsEnabled(t *testing.T) {
	service := NewDisabledSpaceCompanionService()
	capabilities := service.Capabilities()
	if capabilities.ReadEnabled || capabilities.PublishEnabled {
		t.Fatalf("disabled capabilities = %#v", capabilities)
	}
	if _, err := service.Search(context.Background(), port.SpaceSearchQuery{Query: "anything"}); !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("disabled Search() error kind = %v", apperrors.KindOf(err))
	}
}
