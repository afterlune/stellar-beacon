// Package ai contains infrastructure-level model routing. Provider SDKs are
// constructed here and never escape into application services.
package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/ai/einoadapter"
	"benetnasch/app/infra/config"
)

type ModelRouter struct {
	settings   config.AI
	providers  config.AIProviderSettingsSet
	httpClient *http.Client
	observer   port.AIRunObserver

	mu           sync.Mutex
	chat         map[string]port.ChatGateway
	resolvedChat map[string]port.ChatGateway
	embedding    map[string]port.EmbeddingGateway
}

type chatCandidate struct {
	route    port.ModelRoute
	provider config.AIProviderSettings
}

const maxFallbackRoutes = 3

var _ port.ModelRouter = (*ModelRouter)(nil)

func NewModelRouter(settings config.AI, providers config.AIProviderSettingsSet, httpClient *http.Client, observer port.AIRunObserver) *ModelRouter {
	return &ModelRouter{
		settings:     settings,
		providers:    providers,
		httpClient:   httpClient,
		observer:     observer,
		chat:         make(map[string]port.ChatGateway),
		resolvedChat: make(map[string]port.ChatGateway),
		embedding:    make(map[string]port.EmbeddingGateway),
	}
}

func (r *ModelRouter) ResolveChat(ctx context.Context, useCase port.AIUseCase) (port.ChatGateway, port.ModelRoute, error) {
	if r == nil {
		return nil, port.ModelRoute{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.router.chat", errors.New("model router is nil"))
	}
	if !r.settings.Enabled {
		return nil, port.ModelRoute{}, apperrors.AIDisabled("ai.router.chat")
	}
	if useCase == port.AIUseCaseEmbedding {
		return nil, port.ModelRoute{}, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.chat", errors.New("embedding use case requires an embedding gateway"))
	}

	candidates, err := r.chatCandidates(useCase)
	if err != nil {
		if len(candidates) == 0 {
			return nil, port.ModelRoute{UseCase: useCase}, err
		}
		return nil, candidates[0].route, err
	}
	primary := candidates[0]
	key := routeKey(primary.route)
	r.mu.Lock()
	if gateway := r.resolvedChat[key]; gateway != nil {
		r.mu.Unlock()
		return gateway, primary.route, nil
	}
	r.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}
	primaryGateway, err := r.resolveChatCandidate(ctx, primary)
	if err != nil {
		return nil, primary.route, err
	}
	gateway := port.ChatGateway(primaryGateway)
	if len(candidates) > 1 {
		gateway = &fallbackChatGateway{
			primary:   primaryGateway,
			fallbacks: append([]chatCandidate(nil), candidates[1:]...),
			router:    r,
		}
	}

	// Two concurrent first resolutions may construct equivalent adapters. The
	// map remains the single published instance, so callers never observe two
	// routes with different limits after initialization.
	r.mu.Lock()
	if existing := r.resolvedChat[key]; existing != nil {
		r.mu.Unlock()
		return existing, primary.route, nil
	}
	r.resolvedChat[key] = gateway
	r.mu.Unlock()
	return gateway, primary.route, nil
}

func (r *ModelRouter) ResolveEmbedding(ctx context.Context, useCase port.AIUseCase) (port.EmbeddingGateway, port.ModelRoute, error) {
	if r == nil {
		return nil, port.ModelRoute{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.router.embedding", errors.New("model router is nil"))
	}
	if useCase != port.AIUseCaseEmbedding {
		return nil, port.ModelRoute{UseCase: useCase}, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.embedding", errors.New("embedding gateway requires the embedding use case"))
	}
	if !r.settings.Enabled {
		return nil, port.ModelRoute{UseCase: useCase}, apperrors.AIDisabled("ai.router.embedding")
	}
	candidate, err := r.normalizeEmbeddingCandidate(r.settings.Embedding)
	if err != nil {
		return nil, candidate.route, err
	}
	key := routeKey(candidate.route)
	r.mu.Lock()
	if gateway := r.embedding[key]; gateway != nil {
		r.mu.Unlock()
		return gateway, candidate.route, nil
	}
	r.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	gateway, err := buildEmbeddingGateway(ctx, candidate.route, candidate.provider, r.settings, r.httpClient)
	if err != nil {
		return nil, candidate.route, err
	}
	r.mu.Lock()
	if existing := r.embedding[key]; existing != nil {
		r.mu.Unlock()
		return existing, candidate.route, nil
	}
	r.embedding[key] = gateway
	r.mu.Unlock()
	return gateway, candidate.route, nil
}

func (r *ModelRouter) Diagnostics() map[string]any {
	if r == nil {
		return map[string]any{"enabled": false, "error": "nil model router"}
	}
	diagnostics := map[string]any{
		"enabled": r.settings.Enabled,
		"routes": map[string]any{
			"chat":       diagnosticRoute(r.settings.Chat),
			"embedding":  diagnosticRoute(r.settings.Embedding),
			"vision":     diagnosticRoute(r.settings.Vision),
			"writing":    diagnosticRoute(r.settings.Writing),
			"moderation": diagnosticRoute(r.settings.Moderation),
			"behavior":   diagnosticRoute(r.settings.Behavior),
			"dream":      diagnosticRoute(r.settings.Dream),
		},
		"initialized_chat_routes":      r.initializedRoutes(),
		"initialized_embedding_routes": r.initializedEmbeddingRoutes(),
		"circuit_states":               r.circuitStates(),
	}
	if observer, ok := r.observer.(interface{ Snapshot() RunMetricsSnapshot }); ok {
		diagnostics["run_metrics"] = observer.Snapshot()
	}
	return diagnostics
}

// OperationalSnapshot is the typed, sanitized diagnostics contract used by
// the application observability endpoint. Diagnostics intentionally excludes
// provider endpoints, credentials, prompts, request IDs and recent payloads.
func (r *ModelRouter) OperationalSnapshot(_ context.Context) port.AIOperationalSnapshot {
	if r == nil {
		return port.AIOperationalSnapshot{}
	}
	result := port.AIOperationalSnapshot{
		Enabled:                    r.settings.Enabled,
		ConfiguredRoutes:           configuredOperationalRoutes(r.settings),
		InitializedChatRoutes:      r.initializedRoutes(),
		InitializedEmbeddingRoutes: r.initializedEmbeddingRoutes(),
		CircuitStates:              r.circuitStates(),
	}
	if observer, ok := r.observer.(interface{ Snapshot() RunMetricsSnapshot }); ok {
		result.RunMetrics = operationalRunMetrics(observer.Snapshot())
	}
	return result
}

func configuredOperationalRoutes(settings config.AI) []port.AIOperationalRoute {
	routes := []port.AIOperationalRoute{
		operationalRoute(port.AIUseCaseChat, settings.Chat),
		operationalRoute(port.AIUseCaseEmbedding, settings.Embedding),
		operationalRoute(port.AIUseCaseVision, settings.Vision),
		operationalRoute(port.AIUseCaseWriting, settings.Writing),
		operationalRoute(port.AIUseCaseModeration, settings.Moderation),
		operationalRoute(port.AIUseCaseBehavior, settings.Behavior),
		operationalRoute(port.AIUseCaseDream, settings.Dream),
	}
	return routes
}

func operationalRoute(useCase port.AIUseCase, route config.AIModelRoute) port.AIOperationalRoute {
	fallbacks := make([]port.AIOperationalRoute, 0, len(route.Fallbacks))
	for _, fallback := range route.Fallbacks {
		fallbacks = append(fallbacks, operationalRoute(useCase, fallback))
	}
	return port.AIOperationalRoute{
		UseCase:    useCase,
		Provider:   strings.TrimSpace(route.Provider),
		Protocol:   port.ProviderProtocol(strings.TrimSpace(route.Protocol)),
		Model:      strings.TrimSpace(route.Model),
		DataPolicy: strings.TrimSpace(route.DataPolicy),
		Fallbacks:  fallbacks,
	}
}

func operationalRunMetrics(snapshot RunMetricsSnapshot) port.AIRunMetricsSnapshot {
	routes := make([]port.AIRouteMetricsSnapshot, 0, len(snapshot.Routes))
	for _, route := range snapshot.Routes {
		routes = append(routes, port.AIRouteMetricsSnapshot{
			Provider:        route.Provider,
			Model:           route.Model,
			UseCase:         route.UseCase,
			Total:           route.Total,
			Succeeded:       route.Succeeded,
			Failed:          route.Failed,
			Canceled:        route.Canceled,
			RateLimited:     route.RateLimited,
			ErrorRate:       route.ErrorRate,
			DurationP95MS:   nonNegativeMilliseconds(route.DurationP95),
			FirstTokenP95MS: nonNegativeMilliseconds(route.FirstTokenP95),
			InputTokens:     route.InputTokens,
			OutputTokens:    route.OutputTokens,
			TotalTokens:     route.TotalTokens,
		})
	}
	return port.AIRunMetricsSnapshot{
		Total:                snapshot.Total,
		Succeeded:            snapshot.Succeeded,
		Failed:               snapshot.Failed,
		Canceled:             snapshot.Canceled,
		RateLimited:          snapshot.RateLimited,
		ErrorRate:            snapshot.ErrorRate,
		SuccessRate:          snapshot.SuccessRate,
		InputTokens:          snapshot.InputTokens,
		OutputTokens:         snapshot.OutputTokens,
		TotalTokens:          snapshot.TotalTokens,
		DurationP95MS:        nonNegativeMilliseconds(snapshot.DurationP95),
		FirstTokenP95MS:      nonNegativeMilliseconds(snapshot.FirstTokenP95),
		DailyTokenBudget:     snapshot.DailyTokenBudget,
		DailyTokensUsed:      snapshot.DailyTokensUsed,
		DailyTokensRemaining: snapshot.DailyTokensRemaining,
		DailyBudgetAlert:     snapshot.DailyBudgetAlert,
		DailyBudgetExceeded:  snapshot.DailyBudgetExceeded,
		Routes:               routes,
	}
}

func nonNegativeMilliseconds(value time.Duration) int64 {
	if value <= 0 {
		return 0
	}
	return value.Milliseconds()
}

func (r *ModelRouter) initializedEmbeddingRoutes() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]string, 0, len(r.embedding))
	for key := range r.embedding {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func (r *ModelRouter) initializedRoutes() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]string, 0, len(r.chat))
	for key := range r.chat {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func (r *ModelRouter) circuitStates() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[string]string, len(r.chat))
	for key, gateway := range r.chat {
		if diagnostic, ok := gateway.(interface{ CircuitState() string }); ok {
			result[key] = diagnostic.CircuitState()
		}
	}
	return result
}

func (r *ModelRouter) chatCandidates(useCase port.AIUseCase) ([]chatCandidate, error) {
	raw := config.AIModelRoute{}
	switch useCase {
	case port.AIUseCaseChat:
		raw = r.settings.Chat
	case port.AIUseCaseWriting:
		raw = r.settings.Writing
	case port.AIUseCaseVision:
		raw = r.settings.Vision
	case port.AIUseCaseModeration:
		raw = r.settings.Moderation
	case port.AIUseCaseBehavior:
		raw = r.settings.Behavior
	case port.AIUseCaseDream:
		raw = r.settings.Dream
	default:
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.chat", errors.New("unsupported chat use case"))
	}

	primary, err := r.normalizeChatCandidate(useCase, raw)
	if err != nil {
		return []chatCandidate{primary}, err
	}
	candidates := []chatCandidate{primary}
	if err := validateChatRouteCompatibility(primary.route); err != nil {
		return candidates, err
	}
	if raw.FallbacksParseError != nil {
		return candidates, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.fallback", errors.New("fallback configuration is invalid"))
	}
	if len(raw.Fallbacks) > maxFallbackRoutes {
		return candidates, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.fallback", fmt.Errorf("at most %d fallback routes are allowed", maxFallbackRoutes))
	}
	seen := map[string]struct{}{routeKey(primary.route): {}}
	for index, fallbackRaw := range raw.Fallbacks {
		fallback, fallbackErr := r.normalizeChatCandidate(useCase, fallbackRaw)
		if fallbackErr != nil {
			return candidates, apperrors.WrapAI(apperrors.AICodeInvalidRequest, fmt.Sprintf("ai.router.fallback.%d", index), fallbackErr)
		}
		if fallback.route.DataPolicy != primary.route.DataPolicy {
			return candidates, apperrors.NewAI(apperrors.AICodeInvalidRequest, fmt.Sprintf("ai.router.fallback.%d", index), errors.New("fallback data policy must match the primary route"))
		}
		if err := validateChatRouteCompatibility(fallback.route); err != nil {
			return candidates, apperrors.WrapAI(apperrors.AICodeInvalidRequest, fmt.Sprintf("ai.router.fallback.%d", index), err)
		}
		if _, exists := seen[routeKey(fallback.route)]; exists {
			return candidates, apperrors.NewAI(apperrors.AICodeInvalidRequest, fmt.Sprintf("ai.router.fallback.%d", index), errors.New("duplicate fallback route"))
		}
		seen[routeKey(fallback.route)] = struct{}{}
		candidates = append(candidates, fallback)
	}
	return candidates, nil
}

func (r *ModelRouter) normalizeChatCandidate(useCase port.AIUseCase, raw config.AIModelRoute) (chatCandidate, error) {
	provider := strings.ToLower(strings.TrimSpace(raw.Provider))
	protocol := port.ProviderProtocol(strings.TrimSpace(raw.Protocol))
	if protocol == "" && useCase == port.AIUseCaseChat {
		protocol = port.ProviderProtocol(strings.TrimSpace(r.settings.DefaultProtocol))
	}
	model := strings.TrimSpace(raw.Model)
	providerSettings, ok := providerSettingsFor(r.providers, provider)
	if model == "" && ok {
		model = strings.TrimSpace(providerSettings.Model)
	}
	dataPolicy := strings.TrimSpace(raw.DataPolicy)
	if dataPolicy == "" {
		dataPolicy = "default"
	}
	candidate := chatCandidate{route: port.ModelRoute{
		UseCase:    useCase,
		Provider:   provider,
		Protocol:   protocol,
		Model:      model,
		DataPolicy: dataPolicy,
	}, provider: providerSettings}
	if provider == "" || protocol == "" || model == "" {
		return candidate, apperrors.AIDisabled("ai.router.chat")
	}
	if !ok {
		return candidate, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.router.chat", errors.New("provider settings are not configured"))
	}
	return candidate, nil
}

func (r *ModelRouter) normalizeEmbeddingCandidate(raw config.AIModelRoute) (chatCandidate, error) {
	provider := strings.ToLower(strings.TrimSpace(raw.Provider))
	protocol := port.ProviderProtocol(strings.TrimSpace(raw.Protocol))
	if protocol == "" {
		protocol = port.ProviderProtocolOpenAIChatCompletions
	}
	model := strings.TrimSpace(raw.Model)
	providerSettings, ok := providerSettingsFor(r.providers, provider)
	if model == "" && ok {
		model = strings.TrimSpace(providerSettings.Model)
	}
	dataPolicy := strings.TrimSpace(raw.DataPolicy)
	if dataPolicy == "" {
		dataPolicy = "default"
	}
	candidate := chatCandidate{route: port.ModelRoute{
		UseCase:    port.AIUseCaseEmbedding,
		Provider:   provider,
		Protocol:   protocol,
		Model:      model,
		DataPolicy: dataPolicy,
	}, provider: providerSettings}
	if provider == "" || model == "" {
		return candidate, apperrors.AIDisabled("ai.router.embedding")
	}
	if !ok {
		return candidate, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.router.embedding", errors.New("provider settings are not configured"))
	}
	if err := validateEmbeddingRouteCompatibility(candidate.route); err != nil {
		return candidate, err
	}
	return candidate, nil
}

func (r *ModelRouter) resolveChatCandidate(ctx context.Context, candidate chatCandidate) (port.ChatGateway, error) {
	key := routeKey(candidate.route)
	r.mu.Lock()
	if gateway := r.chat[key]; gateway != nil {
		r.mu.Unlock()
		return gateway, nil
	}
	r.mu.Unlock()

	gateway, err := buildChatGateway(ctx, candidate.route, candidate.provider, r.settings, r.httpClient, r.observer)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	if existing := r.chat[key]; existing != nil {
		r.mu.Unlock()
		return existing, nil
	}
	r.chat[key] = gateway
	r.mu.Unlock()
	return gateway, nil
}

func buildEmbeddingGateway(ctx context.Context, route port.ModelRoute, provider config.AIProviderSettings, settings config.AI, httpClient *http.Client) (port.EmbeddingGateway, error) {
	if err := validateEmbeddingRouteCompatibility(route); err != nil {
		return nil, err
	}
	maxConcurrent := settings.MaxConcurrent
	if settings.LocalEmbeddingEnabled {
		// A local model shares the host/Docker memory budget with the backend.
		// Serialize requests even if the general Agent concurrency policy is
		// higher; the Qwen smoke profile is intentionally single-request.
		maxConcurrent = 1
	}
	return einoadapter.NewOpenAIEmbedding(ctx, einoadapter.EmbeddingConfig{
		Provider:      route.Provider,
		Model:         route.Model,
		APIKey:        provider.APIKey,
		BaseURL:       provider.BaseURL,
		Dimension:     settings.EmbeddingConfig.Dimension,
		Timeout:       settings.RequestTimeout,
		MaxConcurrent: maxConcurrent,
		HTTPClient:    httpClient,
	})
}

func validateChatRouteCompatibility(route port.ModelRoute) error {
	switch route.Protocol {
	case port.ProviderProtocolAnthropicMessages:
		if route.Provider != "anthropic" {
			return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.chat", errors.New("anthropic protocol requires anthropic provider"))
		}
	case port.ProviderProtocolOpenAIResponses:
		if route.Provider != "openai" {
			return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.chat", errors.New("responses protocol requires openai provider"))
		}
	case port.ProviderProtocolOpenAIChatCompletions:
		if route.Provider != "openai" && route.Provider != "sglang" {
			return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.chat", errors.New("chat completions protocol requires openai-compatible provider"))
		}
	default:
		return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.chat", errors.New("unsupported provider protocol"))
	}
	return nil
}

func validateEmbeddingRouteCompatibility(route port.ModelRoute) error {
	if route.UseCase != port.AIUseCaseEmbedding {
		return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.embedding", errors.New("embedding route has an invalid use case"))
	}
	if route.Protocol != port.ProviderProtocolOpenAIChatCompletions {
		return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.embedding", errors.New("embedding route must use the OpenAI-compatible protocol"))
	}
	if route.Provider != "openai" && route.Provider != "sglang" && route.Provider != "alibailian" {
		return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.embedding", errors.New("embedding route requires an OpenAI-compatible provider"))
	}
	return nil
}

func buildChatGateway(ctx context.Context, route port.ModelRoute, provider config.AIProviderSettings, settings config.AI, httpClient *http.Client, observer port.AIRunObserver) (port.ChatGateway, error) {
	if err := validateChatRouteCompatibility(route); err != nil {
		return nil, err
	}
	adapterConfig := einoadapter.Config{
		Provider:                route.Provider,
		Protocol:                route.Protocol,
		Model:                   route.Model,
		APIKey:                  provider.APIKey,
		BaseURL:                 provider.BaseURL,
		Timeout:                 settings.RequestTimeout,
		MaxRetries:              provider.MaxRetries,
		MaxConcurrent:           settings.MaxConcurrent,
		CircuitFailureThreshold: settings.CircuitFailureThreshold,
		CircuitResetTimeout:     settings.CircuitResetTimeout,
		HTTPClient:              httpClient,
		Observer:                observer,
	}
	switch route.Protocol {
	case port.ProviderProtocolOpenAIResponses:
		return einoadapter.NewOpenAIResponses(ctx, adapterConfig)
	case port.ProviderProtocolAnthropicMessages:
		return einoadapter.NewAnthropicMessages(ctx, adapterConfig)
	case port.ProviderProtocolOpenAIChatCompletions:
		return einoadapter.NewOpenAIChatCompletions(ctx, adapterConfig)
	default:
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.router.chat", errors.New("unsupported provider protocol"))
	}
}

func providerSettingsFor(settings config.AIProviderSettingsSet, provider string) (config.AIProviderSettings, bool) {
	switch provider {
	case "openai":
		return settings.OpenAI, true
	case "anthropic":
		return settings.Anthropic, true
	case "sglang":
		return settings.SGLang, true
	case "alibailian":
		return settings.AliBailian, true
	default:
		return config.AIProviderSettings{}, false
	}
}

func routeKey(route port.ModelRoute) string {
	return strings.Join([]string{string(route.UseCase), route.Provider, string(route.Protocol), route.Model, route.DataPolicy}, "|")
}

func diagnosticRoute(route config.AIModelRoute) map[string]any {
	fallbacks := make([]map[string]any, 0, len(route.Fallbacks))
	for _, fallback := range route.Fallbacks {
		fallbacks = append(fallbacks, diagnosticRoute(fallback))
	}
	return map[string]any{
		"provider":    strings.TrimSpace(route.Provider),
		"protocol":    strings.TrimSpace(route.Protocol),
		"model":       strings.TrimSpace(route.Model),
		"data_policy": strings.TrimSpace(route.DataPolicy),
		"fallbacks":   fallbacks,
	}
}
