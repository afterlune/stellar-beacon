package service

import (
	"context"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

type DreamReviewPublisherDeps struct {
	Dreams port.DreamRepository
	Now    func() time.Time
}

// DreamReviewPublisher makes approval visible only in the dream projection.
// It intentionally returns skipped: approving a dream must never create an
// article, comment, or talk as a side effect.
type DreamReviewPublisher struct {
	dreams port.DreamRepository
	now    func() time.Time
}

var _ port.AgentReviewPublisher = (*DreamReviewPublisher)(nil)

func NewDreamReviewPublisher(deps DreamReviewPublisherDeps) (*DreamReviewPublisher, error) {
	if deps.Dreams == nil {
		return nil, apperrors.Invalid("service.dream_publisher.dependencies", "dream repository is required")
	}
	return &DreamReviewPublisher{dreams: deps.Dreams, now: deps.Now}, nil
}

func (p *DreamReviewPublisher) Publish(ctx context.Context, input port.AgentReviewPublication) (port.AgentReviewPublicationResult, error) {
	if p == nil || p.dreams == nil {
		return port.AgentReviewPublicationResult{}, apperrors.Unavailable("agent.dream.publish", nil)
	}
	if input.Review.Operation != port.AgentDreamOperation {
		return port.AgentReviewPublicationResult{}, apperrors.Invalid("agent.dream.publish", "review operation is not a dream")
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return port.AgentReviewPublicationResult{}, apperrors.Invalid("agent.dream.publish", "approved dream content is required")
	}
	now := time.Now().UTC()
	if p.now != nil {
		now = p.now().UTC()
	}
	if err := p.dreams.Approve(ctx, input.Review.ID, content, now); err != nil {
		return port.AgentReviewPublicationResult{}, err
	}
	return port.AgentReviewPublicationResult{Status: port.ReviewPublishSkipped}, nil
}

// CompositeAgentReviewPublisher keeps operation-specific publication rules
// explicit while allowing AI Studio to retain one publication boundary.
type CompositeAgentReviewPublisher struct {
	behavior port.AgentReviewPublisher
	dream    port.AgentReviewPublisher
}

func NewCompositeAgentReviewPublisher(behavior, dream port.AgentReviewPublisher) (*CompositeAgentReviewPublisher, error) {
	if behavior == nil && dream == nil {
		return nil, apperrors.Invalid("service.agent_review_publisher", "at least one publisher is required")
	}
	return &CompositeAgentReviewPublisher{behavior: behavior, dream: dream}, nil
}

var _ port.AgentReviewPublisher = (*CompositeAgentReviewPublisher)(nil)

func (p *CompositeAgentReviewPublisher) Publish(ctx context.Context, input port.AgentReviewPublication) (port.AgentReviewPublicationResult, error) {
	if p == nil {
		return port.AgentReviewPublicationResult{}, apperrors.Unavailable("agent.review.publish", nil)
	}
	if input.Review.Operation == port.AgentDreamOperation {
		if p.dream == nil {
			return port.AgentReviewPublicationResult{}, apperrors.Unavailable("agent.dream.publish", nil)
		}
		return p.dream.Publish(ctx, input)
	}
	if p.behavior == nil {
		return port.AgentReviewPublicationResult{}, apperrors.Unavailable("agent.review.publish", nil)
	}
	return p.behavior.Publish(ctx, input)
}
