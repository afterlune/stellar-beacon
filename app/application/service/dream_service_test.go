package service

import (
	"context"
	"testing"
	"time"

	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
)

type serviceDreamRepositoryFake struct {
	entries  []port.DreamEntry
	approved []string
}

func (f *serviceDreamRepositoryFake) Create(context.Context, port.DreamEntry) error { return nil }

func (f *serviceDreamRepositoryFake) GetByReview(_ context.Context, reviewID string) (port.DreamEntry, error) {
	for _, entry := range f.entries {
		if entry.ReviewID == reviewID {
			return entry, nil
		}
	}
	return port.DreamEntry{}, context.Canceled
}

func (f *serviceDreamRepositoryFake) Approve(_ context.Context, reviewID, content string, now time.Time) error {
	f.approved = append(f.approved, reviewID+":"+content)
	return nil
}

func (f *serviceDreamRepositoryFake) SyncReviewStatus(context.Context, string, port.DreamStatus, time.Time) error {
	return nil
}

func (f *serviceDreamRepositoryFake) ListPublic(context.Context, int, int) ([]port.DreamEntry, int, error) {
	return f.entries, len(f.entries), nil
}
func (f *serviceDreamRepositoryFake) ListPendingImages(context.Context, int, time.Time) ([]port.DreamEntry, error) {
	return nil, nil
}
func (f *serviceDreamRepositoryFake) ClaimImage(context.Context, string, string, time.Time, time.Duration) (port.DreamEntry, bool, error) {
	return port.DreamEntry{}, false, nil
}
func (f *serviceDreamRepositoryFake) CompleteImage(context.Context, string, string, port.DreamImageStatus, string, string, time.Time) error {
	return nil
}

func TestDreamReviewPublisherApprovesProjectionWithoutPublicContentWrite(t *testing.T) {
	repository := &serviceDreamRepositoryFake{}
	publisher, err := NewDreamReviewPublisher(DreamReviewPublisherDeps{
		Dreams: repository,
		Now:    func() time.Time { return time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := publisher.Publish(context.Background(), port.AgentReviewPublication{
		Review:  port.AIReview{ID: "dream-1", Operation: port.AgentDreamOperation},
		Content: "人工确认后的梦境",
	})
	if err != nil || result.Status != port.ReviewPublishSkipped || len(repository.approved) != 1 {
		t.Fatalf("result=%+v err=%v approved=%v", result, err, repository.approved)
	}
}

func TestDreamServiceExposesApprovedRecordsOnly(t *testing.T) {
	repository := &serviceDreamRepositoryFake{entries: []port.DreamEntry{
		{ID: "approved", Status: port.DreamApproved, Title: "公开梦", Content: "内容", ImageStatus: port.DreamImagePlaceholder, ImageURL: "/dream-placeholder.svg"},
		{ID: "pending", Status: port.DreamPendingReview, Title: "待审梦", Content: "不应公开"},
	}}
	service, err := NewDreamService(repository, true)
	if err != nil {
		t.Fatal(err)
	}
	result := service.List(context.Background(), DreamQuery{})
	if !result.Flag {
		t.Fatalf("unexpected result=%+v", result)
	}
	page, ok := result.Data.(model.DreamPageDTO)
	if !ok || len(page.Records) != 1 || page.Records[0].ID != "approved" {
		t.Fatalf("public dream page=%+v", result.Data)
	}
}
