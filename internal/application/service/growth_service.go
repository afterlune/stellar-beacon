package service

import (
	"context"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type GrowthService interface {
	Track(ctx context.Context, eventName string, articleID int, path, clientIP string) (rateLimited bool, err error)
	Summary(ctx context.Context, days int) ([]port.GrowthSummary, error)
}

type GrowthServiceDeps struct {
	Repo    port.GrowthRepository
	Limiter port.RateLimiter
}

type MyGrowthService struct {
	repo    port.GrowthRepository
	limiter port.RateLimiter
}

var growthEvents = map[string]struct{}{
	"share_click":       {},
	"subscribe_start":   {},
	"subscribe_confirm": {},
	"unsubscribe":       {},
}

func NewGrowthService(deps GrowthServiceDeps) (*MyGrowthService, error) {
	if deps.Repo == nil {
		return nil, missingServiceDependency("growth", "repository")
	}
	return &MyGrowthService{repo: deps.Repo, limiter: deps.Limiter}, nil
}

func (s *MyGrowthService) Track(ctx context.Context, eventName string, articleID int, path, clientIP string) (bool, error) {
	eventName = strings.TrimSpace(eventName)
	if _, ok := growthEvents[eventName]; !ok {
		return false, apperrors.Invalid("growth.track.event", "unsupported event type")
	}
	if s.limiter != nil {
		allowed, err := allowRateLimit(ctx, s.limiter, "growth:event:ip:", clientIP, 60, time.Minute)
		if err != nil {
			return false, err
		}
		if !allowed {
			return true, nil
		}
	}

	path = strings.TrimSpace(path)
	if len(path) > 255 {
		path = path[:255]
	}
	if err := s.repo.RecordEvent(ctx, entity.TGrowthEvent{EventName: eventName, ArticleId: articleID, Path: path}); err != nil {
		return false, err
	}
	return false, nil
}

func (s *MyGrowthService) Summary(ctx context.Context, days int) ([]port.GrowthSummary, error) {
	if days < 1 || days > 90 {
		days = 30
	}
	return s.repo.Summary(ctx, time.Now().AddDate(0, 0, -days))
}

var _ GrowthService = (*MyGrowthService)(nil)
