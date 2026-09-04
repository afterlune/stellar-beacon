package ai

import (
	"context"
	"encoding/json"
	"testing"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/ai/mock"
	"benetnasch/app/infra/config"
)

func TestModelRouterProbeChatReportsReachabilityAndStreaming(t *testing.T) {
	server := mock.NewServer(nil)
	defer server.Close()
	settings, providers := routerSettings()
	providers.OpenAI.BaseURL = server.URL() + "/v1"
	router := NewModelRouter(settings, providers, nil, nil)

	result, err := router.ProbeChat(context.Background(), port.AIUseCaseChat)
	if err != nil {
		t.Fatalf("ProbeChat() error = %v", err)
	}
	if !result.Reachable || !result.Capabilities.Generate || !result.Capabilities.Streaming {
		t.Fatalf("probe result = %#v, want reachable generate and streaming", result)
	}
	if result.Route.Provider != "openai" || result.Route.Protocol != port.ProviderProtocolOpenAIResponses {
		t.Fatalf("probe route = %#v", result.Route)
	}
	// A local mock can deliver the first delta within the same clock tick;
	// zero is a valid non-negative duration and must not make this test flaky.
	if result.Latency <= 0 || result.FirstTokenLatency < 0 {
		t.Fatalf("probe timing = %#v", result)
	}
	if len(server.Requests()) != 2 {
		t.Fatalf("probe requests = %d, want generation and stream", len(server.Requests()))
	}
}

func TestModelRouterProbeChatRespectsGlobalDisable(t *testing.T) {
	settings, providers := routerSettings()
	settings.Enabled = false
	router := NewModelRouter(settings, providers, nil, nil)

	result, err := router.ProbeChat(context.Background(), port.AIUseCaseChat)
	if !apperrors.IsAICode(err, apperrors.AICodeDisabled) {
		t.Fatalf("ProbeChat() error = %v, want disabled", err)
	}
	if result.Reachable || result.Capabilities.Generate || result.Capabilities.Streaming {
		t.Fatalf("disabled probe result = %#v", result)
	}
}

func TestModelRouterProbeVisionSendsFixedMultimodalInput(t *testing.T) {
	server := mock.NewServer(nil)
	defer server.Close()
	settings, providers := routerSettings()
	settings.Vision = config.AIModelRoute{
		Provider: "openai",
		Protocol: string(port.ProviderProtocolOpenAIChatCompletions),
		Model:    "deepseek-v4-flash-vision-exp",
	}
	providers.OpenAI.BaseURL = server.URL() + "/v1"
	router := NewModelRouter(settings, providers, nil, nil)

	result, err := router.ProbeChat(context.Background(), port.AIUseCaseVision)
	if err != nil {
		t.Fatalf("ProbeChat(vision) error = %v", err)
	}
	if !result.Reachable || !result.Capabilities.Generate || !result.Capabilities.Streaming {
		t.Fatalf("vision probe result = %#v", result)
	}
	requests := server.Requests()
	if len(requests) != 2 {
		t.Fatalf("vision probe requests = %d, want generation and stream", len(requests))
	}
	var payload struct {
		Messages []struct {
			Content []struct {
				Type     string `json:"type"`
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("decode vision probe request: %v", err)
	}
	if len(payload.Messages) != 1 || len(payload.Messages[0].Content) != 2 || payload.Messages[0].Content[1].Type != "image_url" {
		t.Fatalf("vision probe payload = %#v", payload)
	}
	if payload.Messages[0].Content[1].ImageURL.URL == "" {
		t.Fatal("vision probe image data URI is empty")
	}
}

func TestModelRouterProbeEmbeddingReportsReachability(t *testing.T) {
	server := mock.NewServer(func(_ context.Context, request mock.Request) (mock.Response, error) {
		if request.Protocol != "openai_embeddings" {
			t.Fatalf("unexpected embedding protocol: %s", request.Protocol)
		}
		return mock.Response{Body: map[string]any{
			"object": "list",
			"data":   []map[string]any{{"object": "embedding", "index": 0, "embedding": []float32{0.1, 0.2, 0.3}}},
			"model":  "embedding-model",
		}}, nil
	})
	defer server.Close()

	settings, providers := routerSettings()
	settings.Embedding = config.AIModelRoute{
		Provider: "openai",
		Protocol: string(port.ProviderProtocolOpenAIChatCompletions),
		Model:    "embedding-model",
	}
	settings.EmbeddingConfig.Dimension = 3
	providers.OpenAI.BaseURL = server.URL() + "/v1"
	router := NewModelRouter(settings, providers, nil, nil)

	result, err := router.ProbeEmbedding(context.Background())
	if err != nil {
		t.Fatalf("ProbeEmbedding() error = %v", err)
	}
	if !result.Reachable || !result.Capabilities.Embedding || result.Capabilities.Generate || result.Capabilities.Streaming {
		t.Fatalf("embedding probe result = %#v", result)
	}
	if result.Route.UseCase != port.AIUseCaseEmbedding || result.Route.Model != "embedding-model" {
		t.Fatalf("embedding probe route = %#v", result.Route)
	}
	if result.Latency <= 0 || len(server.Requests()) != 1 {
		t.Fatalf("embedding probe timing/requests = %#v/%d", result.Latency, len(server.Requests()))
	}
}
