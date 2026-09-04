package service

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	stderrors "errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxTimeCapsuleLeadTime = 10 * 365 * 24 * time.Hour
)

type TimeCapsuleService interface {
	Create(context.Context, int, TimeCapsuleCreateInput) port.ResultVO
	Get(context.Context, int, string) port.ResultVO
	Seal(context.Context, int, string) port.ResultVO
}

// TimeCapsuleCreateInput intentionally contains only user-authored fields.
// The owner is always derived from the authenticated request, never from
// client JSON or an Agent profile.
type TimeCapsuleCreateInput struct {
	Title     string
	Content   string
	DeliverAt string
}

type TimeCapsuleServiceDeps struct {
	Capsules port.TimeCapsuleRepository
	Safety   port.AgentSafetySwitch
	Now      func() time.Time
	Enabled  bool
}

type MyTimeCapsuleService struct {
	capsules port.TimeCapsuleRepository
	safety   port.AgentSafetySwitch
	now      func() time.Time
	enabled  bool
}

func NewTimeCapsuleService(deps TimeCapsuleServiceDeps) (*MyTimeCapsuleService, error) {
	if deps.Enabled && deps.Capsules == nil {
		return nil, errors.Invalid("time_capsule.service.dependencies", "time capsule repository is required")
	}
	if deps.Now == nil {
		deps.Now = func() time.Time { return time.Now().UTC() }
	}
	return &MyTimeCapsuleService{
		capsules: deps.Capsules,
		safety:   deps.Safety,
		now:      deps.Now,
		enabled:  deps.Enabled,
	}, nil
}

func NewDisabledTimeCapsuleService() *MyTimeCapsuleService {
	return &MyTimeCapsuleService{}
}

func (s *MyTimeCapsuleService) Create(ctx context.Context, ownerUserID int, input TimeCapsuleCreateInput) port.ResultVO {
	if err := s.ensureAvailable(ctx); err != nil {
		return resultFromTimeCapsuleError(err)
	}
	now := s.currentTime()
	capsule, err := normalizeTimeCapsuleCreateInput(ownerUserID, input, now)
	if err != nil {
		return port.ResultFromError(err)
	}
	if err := s.capsules.Create(ctx, capsule); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(timeCapsuleDTO(capsule, true))
}

func (s *MyTimeCapsuleService) Get(ctx context.Context, ownerUserID int, id string) port.ResultVO {
	if err := s.ensureAvailable(ctx); err != nil {
		return resultFromTimeCapsuleError(err)
	}
	if err := port.ValidateTimeCapsuleIdentity(ownerUserID); err != nil {
		return port.ResultFromError(errors.New(errors.KindUnauthorized, "time_capsule.owner", err))
	}
	capsule, err := s.capsules.GetOwned(ctx, strings.TrimSpace(id), ownerUserID, s.currentTime())
	if err != nil {
		return port.ResultFromError(err)
	}
	if capsule.Status == port.TimeCapsuleDue {
		capsule, err = s.capsules.Deliver(ctx, capsule.ID, ownerUserID, s.currentTime())
		if err != nil {
			return port.ResultFromError(err)
		}
	}
	return port.ResultOkWithData(timeCapsuleDTO(capsule, capsule.Status == port.TimeCapsuleDraft || capsule.Status == port.TimeCapsuleDelivered))
}

func (s *MyTimeCapsuleService) Seal(ctx context.Context, ownerUserID int, id string) port.ResultVO {
	if err := s.ensureAvailable(ctx); err != nil {
		return resultFromTimeCapsuleError(err)
	}
	if err := port.ValidateTimeCapsuleIdentity(ownerUserID); err != nil {
		return port.ResultFromError(errors.New(errors.KindUnauthorized, "time_capsule.owner", err))
	}
	id = strings.TrimSpace(id)
	if id == "" || len(id) > port.MaxTimeCapsuleIDLength || strings.ContainsAny(id, "\r\n\x00") {
		return port.ResultFromError(errors.Invalid("time_capsule.seal", "time capsule id is invalid"))
	}
	if err := s.capsules.Seal(ctx, id, ownerUserID, s.currentTime()); err != nil {
		return port.ResultFromError(err)
	}
	capsule, err := s.capsules.GetOwned(ctx, id, ownerUserID, s.currentTime())
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(timeCapsuleDTO(capsule, false))
}

func (s *MyTimeCapsuleService) ensureAvailable(ctx context.Context) error {
	if s == nil || !s.enabled || s.capsules == nil {
		return errors.New(errors.KindForbidden, "time_capsule.feature", stderrors.New("time capsule feature is disabled"))
	}
	if s.safety != nil {
		stopped, err := s.safety.IsStopped(ctx)
		if err != nil {
			return errors.Unavailable("time_capsule.safety", err)
		}
		if stopped {
			return errors.New(errors.KindForbidden, "time_capsule.safety", stderrors.New("time capsule feature is stopped"))
		}
	}
	return nil
}

func (s *MyTimeCapsuleService) currentTime() time.Time {
	if s != nil && s.now != nil {
		if value := s.now(); !value.IsZero() {
			return value.UTC()
		}
	}
	return time.Now().UTC()
}

func normalizeTimeCapsuleCreateInput(ownerUserID int, input TimeCapsuleCreateInput, now time.Time) (port.TimeCapsule, error) {
	if err := port.ValidateTimeCapsuleIdentity(ownerUserID); err != nil {
		return port.TimeCapsule{}, errors.New(errors.KindUnauthorized, "time_capsule.owner", err)
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "未命名时间胶囊"
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return port.TimeCapsule{}, errors.Invalid("time_capsule.content", "content is required")
	}
	if !utf8.ValidString(title) || !utf8.ValidString(content) {
		return port.TimeCapsule{}, errors.Invalid("time_capsule.content", "content must be valid UTF-8")
	}
	deliverAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(input.DeliverAt))
	if err != nil {
		return port.TimeCapsule{}, errors.Invalid("time_capsule.deliver_at", "deliverAt must be RFC3339")
	}
	now = now.UTC()
	deliverAt = deliverAt.UTC()
	if !deliverAt.After(now) {
		return port.TimeCapsule{}, errors.Invalid("time_capsule.deliver_at", "deliverAt must be in the future")
	}
	if deliverAt.After(now.Add(maxTimeCapsuleLeadTime)) {
		return port.TimeCapsule{}, errors.Invalid("time_capsule.deliver_at", "deliverAt is too far in the future")
	}
	capsule := port.TimeCapsule{
		ID:          uuid.NewString(),
		OwnerUserID: ownerUserID,
		Title:       title,
		Content:     content,
		DeliverAt:   deliverAt,
		Status:      port.TimeCapsuleDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := port.ValidateTimeCapsule(capsule); err != nil {
		return port.TimeCapsule{}, errors.Invalid("time_capsule.create", err.Error())
	}
	return capsule, nil
}

func timeCapsuleDTO(capsule port.TimeCapsule, revealContent bool) port.TimeCapsuleDTO {
	content := ""
	if revealContent {
		content = capsule.Content
	}
	return port.TimeCapsuleDTO{
		ID:               capsule.ID,
		Title:            capsule.Title,
		Content:          content,
		DeliverAt:        capsule.DeliverAt.UTC(),
		Status:           string(capsule.Status),
		SealedAt:         capsule.SealedAt.UTC(),
		DeliveredAt:      capsule.DeliveredAt.UTC(),
		DeliveryAttempts: capsule.DeliveryAttempts,
		CreatedAt:        capsule.CreatedAt.UTC(),
		UpdatedAt:        capsule.UpdatedAt.UTC(),
	}
}

func resultFromTimeCapsuleError(err error) port.ResultVO {
	if errors.IsKind(err, errors.KindForbidden) {
		return port.ResultFailWithMessage("时间胶囊暂未开启")
	}
	return port.ResultFromError(err)
}
