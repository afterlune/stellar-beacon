package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"

	"github.com/gin-gonic/gin"
)

type providerProbeFake struct {
	result         port.ModelProbeResult
	err            error
	useCase        port.AIUseCase
	embeddingCalls int
}

func (f *providerProbeFake) ProbeChat(_ context.Context, useCase port.AIUseCase) (port.ModelProbeResult, error) {
	f.useCase = useCase
	return f.result, f.err
}

func (f *providerProbeFake) ProbeEmbedding(_ context.Context) (port.ModelProbeResult, error) {
	f.useCase = port.AIUseCaseEmbedding
	f.embeddingCalls++
	return f.result, f.err
}

func providerProbeTestContext(method, body string) serviceTestRequest {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, "/admin/ai/providers/test", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return serviceTestRequest{ginContextForServiceTest: c}
}

func TestAIProviderProbeServiceUsesConfiguredUseCaseAndReturnsSafeDTO(t *testing.T) {
	fake := &providerProbeFake{result: port.ModelProbeResult{
		Route: port.ModelRoute{
			UseCase:    port.AIUseCaseVision,
			Provider:   "openai",
			Protocol:   port.ProviderProtocolOpenAIChatCompletions,
			Model:      "deepseek-v4-flash-vision-exp",
			DataPolicy: "public",
		},
		Capabilities:      port.ModelCapabilities{Generate: true, Streaming: true},
		Reachable:         true,
		Latency:           1234 * time.Millisecond,
		FirstTokenLatency: 321 * time.Millisecond,
		CheckedAt:         time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC),
	}}
	service := NewAIProviderProbeService(fake, true)
	result := service.Probe(providerProbeTestContext(http.MethodPost, `{"useCase":"vision"}`))
	if !result.Flag || fake.useCase != port.AIUseCaseVision {
		t.Fatalf("Probe() result=%+v useCase=%q", result, fake.useCase)
	}
	dto, ok := result.Data.(model.AIProviderProbeDTO)
	if !ok || dto.Provider != "openai" || dto.Model != "deepseek-v4-flash-vision-exp" || dto.LatencyMS != 1234 || dto.FirstTokenLatencyMS != 321 {
		t.Fatalf("Probe() data=%#v", result.Data)
	}
}

func TestAIProviderProbeServiceFailsClosedAndDoesNotLeakProviderError(t *testing.T) {
	secretErr := apperrors.Unavailable("provider.request", errors.New("api_key=secret response=private"))
	fake := &providerProbeFake{err: secretErr, result: port.ModelProbeResult{ErrorCode: "ai_provider_unavailable"}}
	service := NewAIProviderProbeService(fake, true)
	result := service.Probe(providerProbeTestContext(http.MethodPost, `{"useCase":"chat"}`))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" || result.Message == "api_key=secret" {
		t.Fatalf("Probe() leaked or succeeded: %+v", result)
	}
	dto, ok := result.Data.(model.AIProviderProbeDTO)
	if !ok || dto.ErrorCode != "ai_provider_unavailable" {
		t.Fatalf("safe error data=%#v", result.Data)
	}

	disabled := NewAIProviderProbeService(fake, false)
	result = disabled.Probe(providerProbeTestContext(http.MethodPost, `{"useCase":"vision"}`))
	if result.Flag || fake.useCase != port.AIUseCaseChat {
		t.Fatalf("disabled Probe() result=%+v useCase=%q", result, fake.useCase)
	}
}

func TestAIProviderProbeServiceRejectsUnknownUseCaseAndOversizedBody(t *testing.T) {
	fake := &providerProbeFake{}
	service := NewAIProviderProbeService(fake, true)
	result := service.Probe(providerProbeTestContext(http.MethodPost, `{"useCase":"unknown"}`))
	if result.Flag || result.Message != "参数格式不正确" || fake.useCase != "" {
		t.Fatalf("invalid use case result=%+v useCase=%q", result, fake.useCase)
	}

	result = service.Probe(providerProbeTestContext(http.MethodPost, `{"useCase":"vision","padding":"`+string(make([]byte, providerProbeBodyLimit))+`"}`))
	if result.Flag || result.Message != "参数格式不正确" || fake.useCase != "" {
		t.Fatalf("oversized body result=%+v useCase=%q", result, fake.useCase)
	}
}

func TestAIProviderProbeServiceSupportsEmbedding(t *testing.T) {
	fake := &providerProbeFake{result: port.ModelProbeResult{
		Route: port.ModelRoute{
			UseCase:  port.AIUseCaseEmbedding,
			Provider: "alibailian",
			Protocol: port.ProviderProtocolOpenAIChatCompletions,
			Model:    "qwen3.7-text-embedding",
		},
		Capabilities: port.ModelCapabilities{Embedding: true},
		Reachable:    true,
	}}
	service := NewAIProviderProbeService(fake, true)
	result := service.Probe(providerProbeTestContext(http.MethodPost, `{"useCase":"embedding"}`))
	if !result.Flag || fake.useCase != port.AIUseCaseEmbedding || fake.embeddingCalls != 1 {
		t.Fatalf("embedding Probe() result=%+v useCase=%q calls=%d", result, fake.useCase, fake.embeddingCalls)
	}
	dto, ok := result.Data.(model.AIProviderProbeDTO)
	if !ok || dto.UseCase != "embedding" || !dto.Embedding || dto.Generate || dto.Streaming {
		t.Fatalf("embedding Probe() data=%#v", result.Data)
	}
}

func TestAIProviderProbeDTOHasNoCredentialFields(t *testing.T) {
	payload, err := json.Marshal(model.AIProviderProbeDTO{Provider: "openai", Model: "model"})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(payload, []byte("apiKey")) || bytes.Contains(payload, []byte("secret")) {
		t.Fatalf("probe DTO contains credential fields: %s", payload)
	}
}
