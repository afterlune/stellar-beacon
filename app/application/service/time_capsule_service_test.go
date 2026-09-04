package service

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"context"
	"testing"
	"time"
)

type timeCapsuleRepositoryFake struct {
	capsules map[string]port.TimeCapsule
}

func newTimeCapsuleRepositoryFake() *timeCapsuleRepositoryFake {
	return &timeCapsuleRepositoryFake{capsules: make(map[string]port.TimeCapsule)}
}

func (f *timeCapsuleRepositoryFake) Create(_ context.Context, capsule port.TimeCapsule) error {
	f.capsules[capsule.ID] = capsule
	return nil
}

func (f *timeCapsuleRepositoryFake) GetOwned(_ context.Context, id string, ownerID int, now time.Time) (port.TimeCapsule, error) {
	capsule, ok := f.capsules[id]
	if !ok || capsule.OwnerUserID != ownerID {
		return port.TimeCapsule{}, errors.NotFound("fake.time_capsule.get")
	}
	if capsule.Status == port.TimeCapsuleSealed && !capsule.DeliverAt.After(now) {
		capsule.Status = port.TimeCapsuleDue
		capsule.UpdatedAt = now
		f.capsules[id] = capsule
	}
	return capsule, nil
}

func (f *timeCapsuleRepositoryFake) Seal(_ context.Context, id string, ownerID int, now time.Time) error {
	capsule, ok := f.capsules[id]
	if !ok || capsule.OwnerUserID != ownerID {
		return errors.NotFound("fake.time_capsule.seal")
	}
	if capsule.Status != port.TimeCapsuleDraft {
		return errors.Conflict("fake.time_capsule.seal", "capsule is not a draft")
	}
	capsule.Status = port.TimeCapsuleSealed
	capsule.SealedAt = now
	capsule.UpdatedAt = now
	f.capsules[id] = capsule
	return nil
}

func (f *timeCapsuleRepositoryFake) AdvanceDue(_ context.Context, now time.Time, _ int) (int, error) {
	count := 0
	for id, capsule := range f.capsules {
		if capsule.Status == port.TimeCapsuleSealed && !capsule.DeliverAt.After(now) {
			capsule.Status = port.TimeCapsuleDue
			capsule.UpdatedAt = now
			f.capsules[id] = capsule
			count++
		}
	}
	return count, nil
}

func (f *timeCapsuleRepositoryFake) Deliver(_ context.Context, id string, ownerID int, now time.Time) (port.TimeCapsule, error) {
	capsule, ok := f.capsules[id]
	if !ok || capsule.OwnerUserID != ownerID {
		return port.TimeCapsule{}, errors.NotFound("fake.time_capsule.deliver")
	}
	if capsule.Status == port.TimeCapsuleDelivered {
		return capsule, nil
	}
	if capsule.Status != port.TimeCapsuleDue {
		return port.TimeCapsule{}, errors.Conflict("fake.time_capsule.deliver", "capsule is not due")
	}
	capsule.Status = port.TimeCapsuleDelivered
	capsule.DeliveredAt = now
	capsule.DeliveryAttempts++
	capsule.UpdatedAt = now
	f.capsules[id] = capsule
	return capsule, nil
}

func (f *timeCapsuleRepositoryFake) MarkDeliveryFailed(_ context.Context, id string, ownerID int, reason string, now time.Time) error {
	capsule, ok := f.capsules[id]
	if !ok || capsule.OwnerUserID != ownerID {
		return errors.NotFound("fake.time_capsule.failed")
	}
	capsule.Status = port.TimeCapsuleDeliveryFailed
	capsule.LastDeliveryError = reason
	capsule.DeliveryAttempts++
	capsule.UpdatedAt = now
	f.capsules[id] = capsule
	return nil
}

func (f *timeCapsuleRepositoryFake) RetryDelivery(_ context.Context, id string, ownerID int, now time.Time) error {
	capsule, ok := f.capsules[id]
	if !ok || capsule.OwnerUserID != ownerID {
		return errors.NotFound("fake.time_capsule.retry")
	}
	capsule.Status = port.TimeCapsuleDue
	capsule.LastDeliveryError = ""
	capsule.UpdatedAt = now
	f.capsules[id] = capsule
	return nil
}

func TestTimeCapsuleServiceHidesSealedContentAndDeliversAfterDue(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	clock := now
	repository := newTimeCapsuleRepositoryFake()
	service, err := NewTimeCapsuleService(TimeCapsuleServiceDeps{
		Capsules: repository,
		Enabled:  true,
		Now:      func() time.Time { return clock },
	})
	if err != nil {
		t.Fatal(err)
	}
	created := service.Create(context.Background(), 7, TimeCapsuleCreateInput{
		Title:     "给未来的自己",
		Content:   "不要忘记这段话。",
		DeliverAt: now.Add(time.Hour).Format(time.RFC3339Nano),
	})
	if !created.Flag {
		t.Fatalf("create result = %+v", created)
	}
	draft := created.Data.(model.TimeCapsuleDTO)
	if draft.Content != "不要忘记这段话。" || draft.Status != string(port.TimeCapsuleDraft) {
		t.Fatalf("created dto = %+v", draft)
	}

	// Find the generated ID from the repository without exposing it through a
	// client-controlled owner/profile field.
	var id string
	for key := range repository.capsules {
		id = key
	}
	if id == "" || draft.ID != id {
		t.Fatalf("created capsule id = %q, repository id = %q", draft.ID, id)
	}
	sealed := service.Seal(context.Background(), 7, id)
	if !sealed.Flag {
		t.Fatalf("seal result = %+v", sealed)
	}
	sealedDTO := sealed.Data.(model.TimeCapsuleDTO)
	if sealedDTO.Content != "" || sealedDTO.Status != string(port.TimeCapsuleSealed) {
		t.Fatalf("sealed dto exposed content: %+v", sealedDTO)
	}

	clock = now.Add(2 * time.Hour)
	delivered := service.Get(context.Background(), 7, id)
	if !delivered.Flag {
		t.Fatalf("deliver result = %+v", delivered)
	}
	result := delivered.Data.(model.TimeCapsuleDTO)
	if result.Status != string(port.TimeCapsuleDelivered) || result.Content != "不要忘记这段话。" {
		t.Fatalf("delivered dto = %+v", result)
	}
	if wrongOwner := service.Get(context.Background(), 8, id); wrongOwner.Flag {
		t.Fatal("different owner was allowed to read capsule")
	}
}

func TestTimeCapsuleServiceRejectsAnonymousOwnerAndInvalidDeliveryTime(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	service, err := NewTimeCapsuleService(TimeCapsuleServiceDeps{
		Capsules: newTimeCapsuleRepositoryFake(),
		Enabled:  true,
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if result := service.Create(context.Background(), 0, TimeCapsuleCreateInput{Content: "x", DeliverAt: now.Add(time.Hour).Format(time.RFC3339)}); result.Flag {
		t.Fatal("anonymous owner was accepted")
	}
	if result := service.Create(context.Background(), 7, TimeCapsuleCreateInput{Content: "x", DeliverAt: now.Format(time.RFC3339)}); result.Flag {
		t.Fatal("past delivery time was accepted")
	}
}
