package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const (
	providerProbeBodyLimit = 4 * 1024
	providerProbeTimeout   = 65 * time.Second
)

// AIProviderProbeService exposes an explicit administrator-only Provider
// smoke check. It never accepts credentials or arbitrary provider settings
// from the request body; the infrastructure router resolves configured routes.
type AIProviderProbeService interface {
	Probe(port.Request) port.ResultVO
}

type MyAIProviderProbeService struct {
	probe   port.ModelProbe
	enabled bool
	timeout time.Duration
}

var _ AIProviderProbeService = (*MyAIProviderProbeService)(nil)

func NewAIProviderProbeService(probe port.ModelProbe, enabled bool) AIProviderProbeService {
	return &MyAIProviderProbeService{
		probe:   probe,
		enabled: enabled,
		timeout: providerProbeTimeout,
	}
}

func NewDisabledAIProviderProbeService() AIProviderProbeService {
	return &MyAIProviderProbeService{}
}

type aiProviderProbeRequest struct {
	UseCase string `json:"useCase"`
}

func (s *MyAIProviderProbeService) Probe(c port.Request) port.ResultVO {
	if s == nil || !s.enabled || s.probe == nil {
		return port.ResultFromError(apperrors.Unavailable("agent.provider_probe", nil))
	}
	if c == nil || c.HTTPRequest() == nil {
		return port.ResultFromError(apperrors.Invalid("agent.provider_probe.request", "request is invalid"))
	}
	if c.HTTPRequest().Body != nil {
		c.HTTPRequest().Body = http.MaxBytesReader(c.ResponseWriter(), c.HTTPRequest().Body, providerProbeBodyLimit)
	}
	var request aiProviderProbeRequest
	if err := c.BindJSON(&request); err != nil {
		return port.ResultFromError(apperrors.Invalid("agent.provider_probe.request", "request body is invalid"))
	}
	useCase, err := normalizeProviderProbeUseCase(request.UseCase)
	if err != nil {
		return port.ResultFromError(err)
	}

	ctx := c.Context()
	timeout := s.timeout
	if timeout <= 0 {
		timeout = providerProbeTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var result port.ModelProbeResult
	if useCase == port.AIUseCaseEmbedding {
		result, err = s.probe.ProbeEmbedding(ctx)
	} else {
		result, err = s.probe.ProbeChat(ctx, useCase)
	}
	dto := providerProbeDTO(result, useCase)
	if err != nil {
		response := port.ResultFromError(err)
		// ErrorCode is produced by the infrastructure probe from a fixed
		// allowlist. Returning it is useful for an operator without exposing
		// provider response bodies, request parameters, or credentials.
		response.Data = dto
		return response
	}
	return port.ResultOkWithData(dto)
}

func normalizeProviderProbeUseCase(raw string) (port.AIUseCase, error) {
	useCase := port.AIUseCase(strings.ToLower(strings.TrimSpace(raw)))
	switch useCase {
	case port.AIUseCaseChat, port.AIUseCaseVision, port.AIUseCaseEmbedding:
		return useCase, nil
	default:
		return "", apperrors.Invalid("agent.provider_probe.use_case", "useCase must be chat, vision, or embedding")
	}
}

func providerProbeDTO(result port.ModelProbeResult, useCase port.AIUseCase) port.AIProviderProbeDTO {
	return port.AIProviderProbeDTO{
		UseCase:             string(useCase),
		Provider:            result.Route.Provider,
		Protocol:            string(result.Route.Protocol),
		Model:               result.Route.Model,
		DataPolicy:          result.Route.DataPolicy,
		Reachable:           result.Reachable,
		Generate:            result.Capabilities.Generate,
		Streaming:           result.Capabilities.Streaming,
		Embedding:           result.Capabilities.Embedding,
		LatencyMS:           durationMilliseconds(result.Latency),
		FirstTokenLatencyMS: durationMilliseconds(result.FirstTokenLatency),
		CheckedAt:           result.CheckedAt,
		ErrorCode:           result.ErrorCode,
	}
}

func durationMilliseconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return duration.Milliseconds()
}
