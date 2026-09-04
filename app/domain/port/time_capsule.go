package port

import (
	"context"
	"errors"
	"strings"
	"time"
)

// TimeCapsuleStatus is a deliberately small, persisted state machine. A
// capsule is private while it is draft or sealed; only the owner can open it
// after it reaches due/delivered.
type TimeCapsuleStatus string

const (
	TimeCapsuleDraft          TimeCapsuleStatus = "draft"
	TimeCapsuleSealed         TimeCapsuleStatus = "sealed"
	TimeCapsuleDue            TimeCapsuleStatus = "due"
	TimeCapsuleDelivered      TimeCapsuleStatus = "delivered"
	TimeCapsuleDeliveryFailed TimeCapsuleStatus = "delivery_failed"
)

const (
	MaxTimeCapsuleTitleRunes   = 120
	MaxTimeCapsuleContentRunes = 20_000
	MaxTimeCapsuleErrorRunes   = 1_000
	MaxTimeCapsuleIDLength     = 128
	MaxTimeCapsulePageSize     = 100
)

func NormalizeTimeCapsuleStatus(value string) (TimeCapsuleStatus, error) {
	status := TimeCapsuleStatus(strings.ToLower(strings.TrimSpace(value)))
	switch status {
	case TimeCapsuleDraft, TimeCapsuleSealed, TimeCapsuleDue, TimeCapsuleDelivered, TimeCapsuleDeliveryFailed:
		return status, nil
	default:
		return "", errors.New("time capsule status is invalid")
	}
}

// CanTransitionTimeCapsule is the domain rule that persistence must enforce
// again with a conditional update. Delivery failure is retained so a future
// notification provider can retry without reopening or rewriting a capsule.
func CanTransitionTimeCapsule(from, to TimeCapsuleStatus) bool {
	if from == to {
		return from == TimeCapsuleDelivered
	}
	switch from {
	case TimeCapsuleDraft:
		return to == TimeCapsuleSealed
	case TimeCapsuleSealed:
		return to == TimeCapsuleDue
	case TimeCapsuleDue:
		return to == TimeCapsuleDelivered || to == TimeCapsuleDeliveryFailed
	case TimeCapsuleDeliveryFailed:
		return to == TimeCapsuleDue
	case TimeCapsuleDelivered:
		return false
	default:
		return false
	}
}

// TimeCapsule contains only a stable owner account ID. It intentionally has
// no nickname, email, avatar, profile snapshot, IP, session ID, or model
// context. The content is user-authored and is never assembled from profile
// data by the application.
type TimeCapsule struct {
	ID                string
	OwnerUserID       int
	Title             string
	Content           string
	DeliverAt         time.Time
	Status            TimeCapsuleStatus
	SealedAt          time.Time
	DeliveredAt       time.Time
	DeliveryAttempts  int
	LastDeliveryError string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func ValidateTimeCapsuleIdentity(ownerUserID int) error {
	if ownerUserID <= 0 {
		return errors.New("time capsule owner must be a durable user identity")
	}
	return nil
}

func ValidateTimeCapsule(input TimeCapsule) error {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" || len(input.ID) > MaxTimeCapsuleIDLength || strings.ContainsAny(input.ID, "\r\n\x00") {
		return errors.New("time capsule id is invalid")
	}
	if err := ValidateTimeCapsuleIdentity(input.OwnerUserID); err != nil {
		return err
	}
	if strings.TrimSpace(input.Title) == "" || len([]rune(input.Title)) > MaxTimeCapsuleTitleRunes || strings.ContainsAny(input.Title, "\x00") {
		return errors.New("time capsule title is invalid")
	}
	if strings.TrimSpace(input.Content) == "" || len([]rune(input.Content)) > MaxTimeCapsuleContentRunes || strings.ContainsAny(input.Content, "\x00") {
		return errors.New("time capsule content is invalid")
	}
	if input.DeliverAt.IsZero() {
		return errors.New("time capsule delivery time is required")
	}
	status, err := NormalizeTimeCapsuleStatus(string(input.Status))
	if err != nil {
		return err
	}
	switch status {
	case TimeCapsuleDraft:
		if !input.SealedAt.IsZero() || !input.DeliveredAt.IsZero() {
			return errors.New("draft time capsule timestamps are invalid")
		}
	case TimeCapsuleSealed, TimeCapsuleDue, TimeCapsuleDeliveryFailed:
		if input.SealedAt.IsZero() || !input.DeliveredAt.IsZero() {
			return errors.New("sealed time capsule timestamps are invalid")
		}
	case TimeCapsuleDelivered:
		if input.SealedAt.IsZero() || input.DeliveredAt.IsZero() {
			return errors.New("delivered time capsule timestamps are invalid")
		}
	}
	if input.DeliveryAttempts < 0 {
		return errors.New("time capsule delivery attempts are invalid")
	}
	if len([]rune(input.LastDeliveryError)) > MaxTimeCapsuleErrorRunes || strings.ContainsAny(input.LastDeliveryError, "\x00") {
		return errors.New("time capsule delivery error is invalid")
	}
	return nil
}

// TimeCapsuleRepository is the persistence boundary for both ownership and
// state transitions. GetOwned must not reveal whether an ID exists for a
// different owner; implementations should return NotFound in that case.
type TimeCapsuleRepository interface {
	Create(context.Context, TimeCapsule) error
	GetOwned(context.Context, string, int, time.Time) (TimeCapsule, error)
	Seal(context.Context, string, int, time.Time) error
	AdvanceDue(context.Context, time.Time, int) (int, error)
	Deliver(context.Context, string, int, time.Time) (TimeCapsule, error)
	MarkDeliveryFailed(context.Context, string, int, string, time.Time) error
	RetryDelivery(context.Context, string, int, time.Time) error
}
