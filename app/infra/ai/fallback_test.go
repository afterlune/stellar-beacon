package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/ai/mock"
	"benetnasch/app/infra/config"
)

type fallbackTestGateway struct {
	generate      func(context.Context, port.ChatRequest) (port.ChatResponse, error)
	stream        func(context.Context, port.ChatRequest, func(port.ChatStreamEvent) error) error
	generateCalls int
	streamCalls   int
}

func (g *fallbackTestGateway) Generate(ctx context.Context, request port.ChatRequest) (port.ChatResponse, error) {
	g.generateCalls++
	if g.generate == nil {
		return port.ChatResponse{}, errors.New("generate not configured")
	}
	return g.generate(ctx, request)
}

func (g *fallbackTestGateway) Stream(ctx context.Context, request port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
	g.streamCalls++
	if g.stream == nil {
		return errors.New("stream not configured")
	}
	return g.stream(ctx, request, emit)
}

func fallbackTestRoutes() (port.ModelRoute, port.ModelRoute) {
	return port.ModelRoute{
		UseCase:    port.AIUseCaseChat,
		Provider:   "primary",
		Protocol:   port.ProviderProtocolOpenAIChatCompletions,
		Model:      "primary-model",
		DataPolicy: "public",
	}, port.ModelRoute{
		UseCase:    port.AIUseCaseChat,
		Provider:   "fallback",
		Protocol:   port.ProviderProtocolOpenAIChatCompletions,
		Model:      "fallback-model",
		DataPolicy: "public",
	}
}

func newFallbackTestGateway(primary, fallback port.ChatGateway) *fallbackChatGateway {
	_, fallbackRoute := fallbackTestRoutes()
	router := &ModelRouter{
		chat:         map[string]port.ChatGateway{routeKey(fallbackRoute): fallback},
		resolvedChat: make(map[string]port.ChatGateway),
	}
	return &fallbackChatGateway{
		primary:   primary,
		fallbacks: []chatCandidate{{route: fallbackRoute}},
		router:    router,
	}
}

func TestFallbackGenerateUsesSamePolicyCandidateAfterProviderFailure(t *testing.T) {
	primary := &fallbackTestGateway{
		generate: func(context.Context, port.ChatRequest) (port.ChatResponse, error) {
			return port.ChatResponse{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "primary", errors.New("upstream unavailable"))
		},
	}
	fallback := &fallbackTestGateway{
		generate: func(context.Context, port.ChatRequest) (port.ChatResponse, error) {
			return port.ChatResponse{Provider: "fallback", Model: "fallback-model", Text: "ok"}, nil
		},
	}
	gateway := newFallbackTestGateway(primary, fallback)
	response, err := gateway.Generate(context.Background(), port.ChatRequest{Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}}})
	if err != nil || response.Provider != "fallback" || response.Text != "ok" {
		t.Fatalf("Generate() response=%#v error=%v", response, err)
	}
	if primary.generateCalls != 1 || fallback.generateCalls != 1 {
		t.Fatalf("calls primary=%d fallback=%d, want one each", primary.generateCalls, fallback.generateCalls)
	}
}

func TestFallbackGenerateDoesNotSwitchForInvalidRequest(t *testing.T) {
	primary := &fallbackTestGateway{
		generate: func(context.Context, port.ChatRequest) (port.ChatResponse, error) {
			return port.ChatResponse{}, apperrors.NewAI(apperrors.AICodeInvalidRequest, "primary", errors.New("bad request"))
		},
	}
	fallback := &fallbackTestGateway{
		generate: func(context.Context, port.ChatRequest) (port.ChatResponse, error) {
			return port.ChatResponse{Text: "must not run"}, nil
		},
	}
	gateway := newFallbackTestGateway(primary, fallback)
	if _, err := gateway.Generate(context.Background(), port.ChatRequest{}); !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
		t.Fatalf("Generate() error = %v, want invalid request", err)
	}
	if fallback.generateCalls != 0 {
		t.Fatalf("fallback calls = %d, want 0", fallback.generateCalls)
	}
}

func TestFallbackGenerateDoesNotSwitchOnCancellation(t *testing.T) {
	primary := &fallbackTestGateway{
		generate: func(context.Context, port.ChatRequest) (port.ChatResponse, error) {
			return port.ChatResponse{}, context.Canceled
		},
	}
	fallback := &fallbackTestGateway{
		generate: func(context.Context, port.ChatRequest) (port.ChatResponse, error) {
			return port.ChatResponse{Text: "must not run"}, nil
		},
	}
	gateway := newFallbackTestGateway(primary, fallback)
	if _, err := gateway.Generate(context.Background(), port.ChatRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Generate() error = %v, want context.Canceled", err)
	}
	if fallback.generateCalls != 0 {
		t.Fatalf("fallback calls = %d, want 0", fallback.generateCalls)
	}
}

func TestFallbackStreamUsesFallbackOnlyBeforeOutput(t *testing.T) {
	primary := &fallbackTestGateway{
		stream: func(_ context.Context, _ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
			if err := emit(port.ChatStreamEvent{Kind: port.StreamEventMeta, Provider: "primary"}); err != nil {
				return err
			}
			return apperrors.NewAI(apperrors.AICodeProviderUnavailable, "primary", errors.New("upstream unavailable"))
		},
	}
	fallback := &fallbackTestGateway{
		stream: func(_ context.Context, _ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
			for _, event := range []port.ChatStreamEvent{
				{Kind: port.StreamEventMeta, Provider: "fallback"},
				{Kind: port.StreamEventDelta, Provider: "fallback", Text: "ok"},
				{Kind: port.StreamEventDone, Provider: "fallback"},
			} {
				if err := emit(event); err != nil {
					return err
				}
			}
			return nil
		},
	}
	gateway := newFallbackTestGateway(primary, fallback)
	var events []port.ChatStreamEvent
	if err := gateway.Stream(context.Background(), port.ChatRequest{}, func(event port.ChatStreamEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if len(events) != 3 || events[0].Provider != "fallback" || events[0].Kind != port.StreamEventMeta {
		t.Fatalf("events = %#v, want fallback meta/delta/done only", events)
	}
	if primary.streamCalls != 1 || fallback.streamCalls != 1 {
		t.Fatalf("calls primary=%d fallback=%d, want one each", primary.streamCalls, fallback.streamCalls)
	}
}

func TestFallbackStreamNeverSwitchesAfterOutput(t *testing.T) {
	primary := &fallbackTestGateway{
		stream: func(_ context.Context, _ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
			if err := emit(port.ChatStreamEvent{Kind: port.StreamEventMeta, Provider: "primary"}); err != nil {
				return err
			}
			if err := emit(port.ChatStreamEvent{Kind: port.StreamEventDelta, Provider: "primary", Text: "partial"}); err != nil {
				return err
			}
			return apperrors.NewAI(apperrors.AICodeProviderUnavailable, "primary", errors.New("failed after output"))
		},
	}
	fallback := &fallbackTestGateway{
		stream: func(context.Context, port.ChatRequest, func(port.ChatStreamEvent) error) error {
			return nil
		},
	}
	gateway := newFallbackTestGateway(primary, fallback)
	var events []port.ChatStreamEvent
	err := gateway.Stream(context.Background(), port.ChatRequest{}, func(event port.ChatStreamEvent) error {
		events = append(events, event)
		return nil
	})
	if err == nil {
		t.Fatal("Stream() error = nil, want primary failure")
	}
	if fallback.streamCalls != 0 || len(events) != 2 || events[0].Provider != "primary" || events[1].Text != "partial" {
		t.Fatalf("fallback calls=%d events=%#v, want primary partial output only", fallback.streamCalls, events)
	}
}

func TestFallbackStreamDoesNotSwitchWhenConsumerDisconnects(t *testing.T) {
	primary := &fallbackTestGateway{
		stream: func(_ context.Context, _ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
			return emit(port.ChatStreamEvent{Kind: port.StreamEventDelta, Provider: "primary", Text: "partial"})
		},
	}
	fallback := &fallbackTestGateway{
		stream: func(context.Context, port.ChatRequest, func(port.ChatStreamEvent) error) error {
			return nil
		},
	}
	gateway := newFallbackTestGateway(primary, fallback)
	err := gateway.Stream(context.Background(), port.ChatRequest{}, func(port.ChatStreamEvent) error {
		return context.Canceled
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Stream() error = %v, want context.Canceled", err)
	}
	if fallback.streamCalls != 0 {
		t.Fatalf("fallback calls = %d, want 0", fallback.streamCalls)
	}
}

func TestModelRouterRejectsFallbackDataPolicyMismatch(t *testing.T) {
	settings, providers := routerSettings()
	settings.Chat.DataPolicy = "public"
	settings.Chat.Fallbacks = []config.AIModelRoute{{
		Provider:   "sglang",
		Protocol:   string(port.ProviderProtocolOpenAIChatCompletions),
		Model:      "fallback-model",
		DataPolicy: "private",
	}}
	providers.OpenAI.APIKey = "test-key"
	providers.SGLang.APIKey = "test-key"
	router := NewModelRouter(settings, providers, nil, nil)
	_, route, err := router.ResolveChat(context.Background(), port.AIUseCaseChat)
	if !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
		t.Fatalf("ResolveChat() error = %v, want invalid request", err)
	}
	if route.DataPolicy != "public" {
		t.Fatalf("primary route = %#v, want public policy", route)
	}
}

func TestModelRouterLazilyUsesConfiguredFallback(t *testing.T) {
	primaryServer := mock.NewServer(func(context.Context, mock.Request) (mock.Response, error) {
		return mock.Response{Status: 503, Body: map[string]string{"error": "primary unavailable"}}, nil
	})
	defer primaryServer.Close()
	fallbackServer := mock.NewServer(nil)
	defer fallbackServer.Close()

	settings, providers := routerSettings()
	settings.Chat = config.AIModelRoute{
		Provider:   "openai",
		Protocol:   string(port.ProviderProtocolOpenAIChatCompletions),
		Model:      "primary-model",
		DataPolicy: "public",
		Fallbacks: []config.AIModelRoute{{
			Provider:   "sglang",
			Protocol:   string(port.ProviderProtocolOpenAIChatCompletions),
			Model:      "fallback-model",
			DataPolicy: "public",
		}},
	}
	providers.OpenAI = config.AIProviderSettings{APIKey: "test-key", BaseURL: primaryServer.URL() + "/v1"}
	providers.SGLang = config.AIProviderSettings{BaseURL: fallbackServer.URL() + "/v1"}
	settings.RequestTimeout = time.Second
	router := NewModelRouter(settings, providers, nil, nil)
	gateway, route, err := router.ResolveChat(context.Background(), port.AIUseCaseChat)
	if err != nil {
		t.Fatalf("ResolveChat() error = %v", err)
	}
	if route.Provider != "openai" || route.DataPolicy != "public" {
		t.Fatalf("primary route = %#v", route)
	}
	response, err := gateway.Generate(context.Background(), port.ChatRequest{Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}}})
	if err != nil || response.Provider != "sglang" || response.Model != "fallback-model" {
		t.Fatalf("fallback response=%#v error=%v", response, err)
	}
	if len(primaryServer.Requests()) != 1 || len(fallbackServer.Requests()) != 1 {
		t.Fatalf("requests primary=%d fallback=%d, want one each", len(primaryServer.Requests()), len(fallbackServer.Requests()))
	}
	if len(router.initializedRoutes()) != 2 {
		t.Fatalf("initialized routes = %#v, want primary and fallback", router.initializedRoutes())
	}
}
