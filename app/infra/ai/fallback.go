package ai

import (
	"context"
	"errors"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

// fallbackChatGateway applies only the route candidates validated by
// ModelRouter. It never executes a tool and never falls back after a stream
// has emitted an event, because the caller may already have rendered output
// from the primary provider.
type fallbackChatGateway struct {
	primary   port.ChatGateway
	fallbacks []chatCandidate
	router    *ModelRouter
}

var _ port.ChatGateway = (*fallbackChatGateway)(nil)

func (g *fallbackChatGateway) Generate(ctx context.Context, request port.ChatRequest) (port.ChatResponse, error) {
	if g == nil || g.primary == nil || g.router == nil {
		return port.ChatResponse{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.fallback", errors.New("fallback gateway is not initialized"))
	}
	response, err := g.primary.Generate(ctx, request)
	if err == nil || !fallbackEligible(err) {
		return response, err
	}
	lastErr := err
	for _, candidate := range g.fallbacks {
		gateway, buildErr := g.router.resolveChatCandidate(ctx, candidate)
		if buildErr != nil {
			lastErr = buildErr
			if !fallbackEligible(buildErr) {
				return port.ChatResponse{}, buildErr
			}
			continue
		}
		response, err = gateway.Generate(ctx, request)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if !fallbackEligible(err) {
			return port.ChatResponse{}, err
		}
	}
	return port.ChatResponse{}, lastErr
}

func (g *fallbackChatGateway) Stream(ctx context.Context, request port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
	if g == nil || g.primary == nil || g.router == nil {
		return apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.fallback", errors.New("fallback gateway is not initialized"))
	}
	if emit == nil {
		return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.fallback", errors.New("stream callback is nil"))
	}

	var primaryMeta *port.ChatStreamEvent
	forwardedEvent := false
	primaryErr := g.primary.Stream(ctx, request, func(event port.ChatStreamEvent) error {
		if event.Kind == port.StreamEventMeta && !forwardedEvent {
			copy := event
			primaryMeta = &copy
			return nil
		}
		if !forwardedEvent {
			forwardedEvent = true
			if primaryMeta != nil {
				if err := emit(*primaryMeta); err != nil {
					return err
				}
			}
		}
		return emit(event)
	})
	if primaryErr == nil {
		if !forwardedEvent && primaryMeta != nil {
			return emit(*primaryMeta)
		}
		return nil
	}
	if forwardedEvent || !fallbackEligible(primaryErr) {
		return primaryErr
	}

	lastErr := primaryErr
	for _, candidate := range g.fallbacks {
		gateway, buildErr := g.router.resolveChatCandidate(ctx, candidate)
		if buildErr != nil {
			lastErr = buildErr
			if !fallbackEligible(buildErr) {
				return buildErr
			}
			continue
		}
		if err := gateway.Stream(ctx, request, emit); err == nil {
			return nil
		} else {
			lastErr = err
			if !fallbackEligible(err) {
				return err
			}
		}
	}
	return lastErr
}

func fallbackEligible(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	switch apperrors.AICodeOf(err) {
	case apperrors.AICodeProviderUnavailable,
		apperrors.AICodeCircuitOpen,
		apperrors.AICodeRateLimited:
		return true
	default:
		return false
	}
}
