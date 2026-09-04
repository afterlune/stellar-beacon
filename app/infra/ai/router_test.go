package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/ai/mock"
	"benetnasch/app/infra/config"
)

func routerSettings() (config.AI, config.AIProviderSettingsSet) {
	return config.AI{
		Enabled:         true,
		DefaultProtocol: string(port.ProviderProtocolOpenAIResponses),
		Chat: config.AIModelRoute{
			Provider: "openai",
			Protocol: string(port.ProviderProtocolOpenAIResponses),
			Model:    "test-model",
		},
		RequestTimeout: time.Second,
		MaxConcurrent:  1,
	}, config.AIProviderSettingsSet{
		OpenAI: config.AIProviderSettings{
			APIKey: "test-key",
			Model:  "provider-model",
		},
	}
}

func TestModelRouterResolvesAndCachesConfiguredChatGateway(t *testing.T) {
	settings, providers := routerSettings()
	router := NewModelRouter(settings, providers, nil, nil)

	first, route, err := router.ResolveChat(context.Background(), port.AIUseCaseChat)
	if err != nil {
		t.Fatalf("ResolveChat() error = %v", err)
	}
	second, secondRoute, err := router.ResolveChat(context.Background(), port.AIUseCaseChat)
	if err != nil {
		t.Fatalf("second ResolveChat() error = %v", err)
	}
	if first != second {
		t.Fatal("ResolveChat() did not return the cached gateway")
	}
	if route != secondRoute || route.Provider != "openai" || route.Protocol != port.ProviderProtocolOpenAIResponses || route.Model != "test-model" {
		t.Fatalf("route = %#v, want explicit chat route", route)
	}
}

func TestModelRouterOperationalSnapshotOmitsProviderCredentials(t *testing.T) {
	settings, providers := routerSettings()
	providers.OpenAI.APIKey = "super-secret-provider-key"
	observer := NewRunMetricsObserver(2)
	router := NewModelRouter(settings, providers, nil, observer)

	snapshot := router.OperationalSnapshot(context.Background())
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal operational snapshot: %v", err)
	}
	if strings.Contains(string(raw), providers.OpenAI.APIKey) {
		t.Fatalf("operational snapshot leaked provider credentials: %s", raw)
	}
	if !snapshot.Enabled || len(snapshot.ConfiguredRoutes) == 0 || snapshot.RunMetrics.Routes == nil {
		t.Fatalf("operational snapshot = %#v", snapshot)
	}
}

func TestModelRouterResolvesVisionAsSeparateChatCapability(t *testing.T) {
	settings, providers := routerSettings()
	settings.Vision = config.AIModelRoute{
		Provider: "openai",
		Protocol: string(port.ProviderProtocolOpenAIChatCompletions),
		Model:    "deepseek-v4-flash-vision-exp",
	}
	router := NewModelRouter(settings, providers, nil, nil)

	gateway, route, err := router.ResolveChat(context.Background(), port.AIUseCaseVision)
	if err != nil {
		t.Fatalf("ResolveChat(vision) error = %v", err)
	}
	if gateway == nil || route.UseCase != port.AIUseCaseVision || route.Provider != "openai" || route.Protocol != port.ProviderProtocolOpenAIChatCompletions || route.Model != "deepseek-v4-flash-vision-exp" {
		t.Fatalf("vision gateway/route = (%v, %#v)", gateway, route)
	}
}

func TestModelRouterResolvesASeparateEmbeddingGateway(t *testing.T) {
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

	chat, _, err := router.ResolveChat(context.Background(), port.AIUseCaseChat)
	if err != nil {
		t.Fatalf("ResolveChat() error = %v", err)
	}
	embedding, route, err := router.ResolveEmbedding(context.Background(), port.AIUseCaseEmbedding)
	if err != nil {
		t.Fatalf("ResolveEmbedding() error = %v", err)
	}
	if embedding == nil || chat == nil || route.Model != "embedding-model" {
		t.Fatalf("resolved gateways/route = (%v, %v, %#v)", chat, embedding, route)
	}
	if _, ok := embedding.(port.ChatGateway); ok {
		t.Fatal("embedding gateway must not be a chat gateway")
	}
	result, err := embedding.Embed(context.Background(), port.EmbeddingRequest{
		Inputs:  []string{"hello"},
		Model:   "embedding-model",
		Version: "v1",
	})
	if err != nil || result.Dimension != 3 || len(result.Vectors) != 1 {
		t.Fatalf("embedding result = %#v, error = %v", result, err)
	}
}

func TestModelRouterResolvesAliBailianEmbeddingGateway(t *testing.T) {
	server := mock.NewServer(func(_ context.Context, request mock.Request) (mock.Response, error) {
		if request.Protocol != "openai_embeddings" {
			t.Fatalf("unexpected embedding protocol: %s", request.Protocol)
		}
		return mock.Response{Body: map[string]any{
			"object": "list",
			"data":   []map[string]any{{"object": "embedding", "index": 0, "embedding": []float32{0.1, 0.2, 0.3}}},
			"model":  "qwen3.7-text-embedding",
		}}, nil
	})
	defer server.Close()

	settings, providers := routerSettings()
	settings.Embedding = config.AIModelRoute{
		Provider: "alibailian",
		Protocol: string(port.ProviderProtocolOpenAIChatCompletions),
		Model:    "qwen3.7-text-embedding",
	}
	settings.EmbeddingConfig.Dimension = 3
	providers.AliBailian = config.AIProviderSettings{
		APIKey:  "alibailian-test-key",
		BaseURL: server.URL() + "/v1",
		Model:   "qwen3.7-text-embedding",
	}
	router := NewModelRouter(settings, providers, nil, nil)

	embedding, route, err := router.ResolveEmbedding(context.Background(), port.AIUseCaseEmbedding)
	if err != nil {
		t.Fatalf("ResolveEmbedding() error = %v", err)
	}
	if embedding == nil || route.Provider != "alibailian" || route.Model != "qwen3.7-text-embedding" {
		t.Fatalf("resolved gateway/route = (%v, %#v)", embedding, route)
	}
	result, err := embedding.Embed(context.Background(), port.EmbeddingRequest{Inputs: []string{"hello"}, Model: "qwen3.7-text-embedding", Version: "qwen3.7-text-embedding"})
	if err != nil || result.Dimension != 3 || len(result.Vectors) != 1 {
		t.Fatalf("embedding result = %#v, error = %v", result, err)
	}
}

func TestModelRouterSerializesLocalEmbeddingRequests(t *testing.T) {
	var active int32
	var maxActive int32
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	server := mock.NewServer(func(ctx context.Context, request mock.Request) (mock.Response, error) {
		current := atomic.AddInt32(&active, 1)
		for {
			previous := atomic.LoadInt32(&maxActive)
			if current <= previous || atomic.CompareAndSwapInt32(&maxActive, previous, current) {
				break
			}
		}
		entered <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
		}
		atomic.AddInt32(&active, -1)
		return mock.Response{Body: map[string]any{
			"object": "list",
			"data":   []map[string]any{{"object": "embedding", "index": 0, "embedding": []float32{0.1, 0.2, 0.3}}},
			"model":  "local-model",
		}}, nil
	})
	defer server.Close()

	settings, providers := routerSettings()
	settings.LocalEmbeddingEnabled = true
	settings.Embedding = config.AIModelRoute{
		Provider: "sglang",
		Protocol: string(port.ProviderProtocolOpenAIChatCompletions),
		Model:    "local-model",
	}
	settings.EmbeddingConfig.Dimension = 3
	providers.SGLang = config.AIProviderSettings{BaseURL: server.URL() + "/v1", Model: "local-model"}
	settings.MaxConcurrent = 2
	router := NewModelRouter(settings, providers, nil, nil)
	embedding, _, err := router.ResolveEmbedding(context.Background(), port.AIUseCaseEmbedding)
	if err != nil {
		t.Fatalf("ResolveEmbedding() error = %v", err)
	}

	request := port.EmbeddingRequest{Inputs: []string{"hello"}, Model: "local-model", Version: "local-v1"}
	errorsCh := make(chan error, 2)
	for index := 0; index < 2; index++ {
		go func() {
			_, err := embedding.Embed(context.Background(), request)
			errorsCh <- err
		}()
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("local embedding request did not reach the provider")
	}
	select {
	case <-entered:
		t.Fatal("local embedding admitted concurrent provider requests")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	for index := 0; index < 2; index++ {
		select {
		case err := <-errorsCh:
			if err != nil {
				t.Fatalf("local embedding request %d error = %v", index, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("local embedding request did not finish")
		}
	}
	if got := atomic.LoadInt32(&maxActive); got != 1 {
		t.Fatalf("maximum local embedding provider concurrency = %d, want 1", got)
	}
}

func TestModelRouterRejectsProtocolProviderMismatch(t *testing.T) {
	settings, providers := routerSettings()
	settings.Chat.Provider = "anthropic"
	settings.Chat.Protocol = string(port.ProviderProtocolOpenAIResponses)
	providers.Anthropic = config.AIProviderSettings{APIKey: "test-key", Model: "claude"}
	router := NewModelRouter(settings, providers, nil, nil)

	_, route, err := router.ResolveChat(context.Background(), port.AIUseCaseChat)
	if !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
		t.Fatalf("ResolveChat() error = %v, want invalid request", err)
	}
	if route.Provider != "anthropic" {
		t.Fatalf("route = %#v, want route metadata preserved", route)
	}
}

func TestModelRouterRejectsMalformedFallbackConfiguration(t *testing.T) {
	settings, providers := routerSettings()
	settings.Chat.FallbacksParseError = errors.New("invalid fallback shape")
	router := NewModelRouter(settings, providers, nil, nil)

	_, _, err := router.ResolveChat(context.Background(), port.AIUseCaseChat)
	if !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
		t.Fatalf("ResolveChat() error = %v, want invalid request", err)
	}
}

func TestModelRouterDiagnosticsDoNotExposeProviderCredentials(t *testing.T) {
	settings, providers := routerSettings()
	providers.OpenAI.APIKey = "super-secret-key"
	router := NewModelRouter(settings, providers, nil, nil)
	diagnostics := router.Diagnostics()
	if _, ok := diagnostics["providers"]; ok {
		t.Fatal("diagnostics unexpectedly contains provider credentials")
	}
	if diagnostics["enabled"] != true {
		t.Fatalf("diagnostics enabled = %#v, want true", diagnostics["enabled"])
	}
}

func TestModelRouterWiresRunObserverAndCircuitDiagnostics(t *testing.T) {
	server := mock.NewServer(nil)
	defer server.Close()
	settings, providers := routerSettings()
	providers.OpenAI.BaseURL = server.URL() + "/v1"
	observer := NewRunMetricsObserver(4)
	router := NewModelRouter(settings, providers, nil, observer)

	gateway, _, err := router.ResolveChat(context.Background(), port.AIUseCaseChat)
	if err != nil {
		t.Fatalf("ResolveChat() error = %v", err)
	}
	if _, err := gateway.Generate(context.Background(), port.ChatRequest{
		UseCase:  port.AIUseCaseChat,
		Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}},
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	snapshot := observer.Snapshot()
	if snapshot.Total != 1 || snapshot.Succeeded != 1 || len(snapshot.Recent) != 1 {
		t.Fatalf("run metrics = %#v", snapshot)
	}
	diagnostics := router.Diagnostics()
	if _, ok := diagnostics["circuit_states"]; !ok {
		t.Fatal("diagnostics do not contain circuit states")
	}
	if _, ok := diagnostics["run_metrics"]; !ok {
		t.Fatal("diagnostics do not contain run metrics")
	}
}
