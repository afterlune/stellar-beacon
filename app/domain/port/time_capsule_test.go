package port

import (
	"strings"
	"testing"
	"time"
)

func TestTimeCapsuleStateMachine(t *testing.T) {
	allowed := [][2]TimeCapsuleStatus{
		{TimeCapsuleDraft, TimeCapsuleSealed},
		{TimeCapsuleSealed, TimeCapsuleDue},
		{TimeCapsuleDue, TimeCapsuleDelivered},
		{TimeCapsuleDue, TimeCapsuleDeliveryFailed},
		{TimeCapsuleDeliveryFailed, TimeCapsuleDue},
	}
	for _, transition := range allowed {
		if !CanTransitionTimeCapsule(transition[0], transition[1]) {
			t.Fatalf("transition %s -> %s should be allowed", transition[0], transition[1])
		}
	}
	for _, transition := range [][2]TimeCapsuleStatus{
		{TimeCapsuleDraft, TimeCapsuleDue},
		{TimeCapsuleSealed, TimeCapsuleDelivered},
		{TimeCapsuleDelivered, TimeCapsuleDue},
	} {
		if CanTransitionTimeCapsule(transition[0], transition[1]) {
			t.Fatalf("transition %s -> %s should be rejected", transition[0], transition[1])
		}
	}
}

func TestValidateTimeCapsuleRequiresDurableOwnerAndBoundedContent(t *testing.T) {
	base := TimeCapsule{
		ID:          "capsule-1",
		OwnerUserID: 7,
		Title:       "给未来的自己",
		Content:     "记得回来看看。",
		DeliverAt:   time.Now().Add(time.Hour),
		Status:      TimeCapsuleDraft,
	}
	if err := ValidateTimeCapsule(base); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []TimeCapsule{
		baseWith(base, func(value *TimeCapsule) { value.OwnerUserID = 0 }),
		baseWith(base, func(value *TimeCapsule) { value.Content = strings.Repeat("中", MaxTimeCapsuleContentRunes+1) }),
		baseWith(base, func(value *TimeCapsule) { value.Status = TimeCapsuleDelivered }),
	} {
		if err := ValidateTimeCapsule(invalid); err == nil {
			t.Fatalf("invalid capsule was accepted: %+v", invalid)
		}
	}
}

func baseWith(base TimeCapsule, mutate func(*TimeCapsule)) TimeCapsule {
	mutated := base
	mutate(&mutated)
	return mutated
}
