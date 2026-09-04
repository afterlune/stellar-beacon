package ai

import (
	"context"
	"errors"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

// A valid, non-sensitive 1x1 PNG keeps the vision probe deterministic and
// avoids fetching an operator-supplied URL or sending any site content.
const providerProbeImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

var _ port.ModelProbe = (*ModelRouter)(nil)

// ProbeChat makes one bounded generation request and one bounded stream check.
// It is intentionally opt-in: normal traffic never performs health probes.
func (r *ModelRouter) ProbeChat(ctx context.Context, useCase port.AIUseCase) (result port.ModelProbeResult, err error) {
	result.CheckedAt = time.Now().UTC()
	startedAt := time.Now()
	defer func() {
		result.Latency = probeElapsed(startedAt)
		result.ErrorCode = probeErrorCode(err)
	}()

	if ctx == nil {
		ctx = context.Background()
	}
	gateway, route, err := r.ResolveChat(ctx, useCase)
	result.Route = route
	if err != nil {
		return result, err
	}
	response, err := gateway.Generate(ctx, providerProbeRequest(useCase, "connection test"))
	if err != nil {
		return result, err
	}
	result.Reachable = true
	result.Capabilities.Generate = true
	_ = response

	streamStartedAt := time.Now()
	firstDelta := false
	streamErr := gateway.Stream(ctx, providerProbeRequest(useCase, "stream connection test"), func(event port.ChatStreamEvent) error {
		if event.Kind == port.StreamEventDelta && strings.TrimSpace(event.Text) != "" {
			firstDelta = true
			result.FirstTokenLatency = probeElapsed(streamStartedAt)
			return context.Canceled
		}
		return nil
	})
	if streamErr != nil && !firstDelta {
		return result, streamErr
	}
	result.Capabilities.Streaming = true
	return result, nil
}

// ProbeEmbedding makes one bounded embedding request with a fixed, harmless
// phrase. It records only route metadata and capability state; the returned
// vector is deliberately never exposed to the application probe response.
func (r *ModelRouter) ProbeEmbedding(ctx context.Context) (result port.ModelProbeResult, err error) {
	result.CheckedAt = time.Now().UTC()
	startedAt := time.Now()
	defer func() {
		result.Latency = probeElapsed(startedAt)
		result.ErrorCode = probeErrorCode(err)
	}()

	if ctx == nil {
		ctx = context.Background()
	}
	gateway, route, err := r.ResolveEmbedding(ctx, port.AIUseCaseEmbedding)
	result.Route = route
	if err != nil {
		return result, err
	}
	response, err := gateway.Embed(ctx, port.EmbeddingRequest{
		Inputs: []string{"embedding connection test"},
		Model:  route.Model,
	})
	if err != nil {
		return result, err
	}
	if len(response.Vectors) != 1 || len(response.Vectors[0]) == 0 {
		return result, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.probe.embedding", errors.New("embedding provider returned no vector"))
	}
	result.Reachable = true
	result.Capabilities.Embedding = true
	return result, nil
}

// probeElapsed keeps completed probe timings positive even when a very fast
// local fake falls below the platform clock's observable resolution.
func probeElapsed(startedAt time.Time) time.Duration {
	elapsed := time.Since(startedAt)
	if elapsed <= 0 {
		return time.Nanosecond
	}
	return elapsed
}

func providerProbeRequest(useCase port.AIUseCase, prompt string) port.ChatRequest {
	message := port.ChatMessage{Role: port.ChatRoleUser, Content: prompt}
	if useCase == port.AIUseCaseVision {
		message.ContentParts = []port.ChatMessagePart{{
			Type:       port.ChatMessagePartTypeImageURL,
			Base64Data: providerProbeImageBase64,
			MIMEType:   "image/png",
			Detail:     "low",
		}}
	}
	return port.ChatRequest{
		UseCase:         useCase,
		Messages:        []port.ChatMessage{message},
		MaxOutputTokens: 1,
	}
}

func probeErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if code := apperrors.AICodeOf(err); code != "" {
		return string(code)
	}
	if errors.Is(err, context.Canceled) {
		return "context_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "context_deadline_exceeded"
	}
	return string(apperrors.AICodeProviderUnavailable)
}
