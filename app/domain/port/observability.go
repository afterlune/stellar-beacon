package port

import (
	"context"
	"time"
)

// AIOperationalRoute is a sanitized view of one configured AI route. It is
// safe to return to an authenticated operator: it contains capability and
// model names only, never provider endpoints, credentials, prompts, or
// request payloads.
type AIOperationalRoute struct {
	UseCase    AIUseCase            `json:"useCase"`
	Provider   string               `json:"provider"`
	Protocol   ProviderProtocol     `json:"protocol"`
	Model      string               `json:"model"`
	DataPolicy string               `json:"dataPolicy"`
	Fallbacks  []AIOperationalRoute `json:"fallbacks,omitempty"`
}

// AIRouteMetricsSnapshot is the stable application-facing shape for one
// provider/model aggregate. P95 values are milliseconds so the HTTP contract
// is readable and does not depend on Go's duration JSON representation.
type AIRouteMetricsSnapshot struct {
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	UseCase         AIUseCase `json:"useCase"`
	Total           uint64    `json:"total"`
	Succeeded       uint64    `json:"succeeded"`
	Failed          uint64    `json:"failed"`
	Canceled        uint64    `json:"canceled"`
	RateLimited     uint64    `json:"rateLimited"`
	ErrorRate       float64   `json:"errorRate"`
	DurationP95MS   int64     `json:"durationP95Ms"`
	FirstTokenP95MS int64     `json:"firstTokenP95Ms"`
	InputTokens     uint64    `json:"inputTokens"`
	OutputTokens    uint64    `json:"outputTokens"`
	TotalTokens     uint64    `json:"totalTokens"`
}

// AIRunMetricsSnapshot contains bounded counters and recent-window latency
// percentiles. Individual prompts, completions, request IDs and provider
// response bodies are deliberately absent.
type AIRunMetricsSnapshot struct {
	Total                uint64                   `json:"total"`
	Succeeded            uint64                   `json:"succeeded"`
	Failed               uint64                   `json:"failed"`
	Canceled             uint64                   `json:"canceled"`
	RateLimited          uint64                   `json:"rateLimited"`
	ErrorRate            float64                  `json:"errorRate"`
	SuccessRate          float64                  `json:"successRate"`
	InputTokens          uint64                   `json:"inputTokens"`
	OutputTokens         uint64                   `json:"outputTokens"`
	TotalTokens          uint64                   `json:"totalTokens"`
	DurationP95MS        int64                    `json:"durationP95Ms"`
	FirstTokenP95MS      int64                    `json:"firstTokenP95Ms"`
	DailyTokenBudget     int64                    `json:"dailyTokenBudget"`
	DailyTokensUsed      int64                    `json:"dailyTokensUsed"`
	DailyTokensRemaining int64                    `json:"dailyTokensRemaining"`
	DailyBudgetAlert     bool                     `json:"dailyBudgetAlert"`
	DailyBudgetExceeded  bool                     `json:"dailyBudgetExceeded"`
	Routes               []AIRouteMetricsSnapshot `json:"routes"`
}

// AIOperationalSnapshot is the sanitized AI portion of the runtime metrics
// endpoint. Circuit state is diagnostic state, not an authorization decision;
// all endpoint access remains protected by the admin middleware.
type AIOperationalSnapshot struct {
	Enabled                    bool                 `json:"enabled"`
	ConfiguredRoutes           []AIOperationalRoute `json:"configuredRoutes"`
	InitializedChatRoutes      []string             `json:"initializedChatRoutes"`
	InitializedEmbeddingRoutes []string             `json:"initializedEmbeddingRoutes"`
	CircuitStates              map[string]string    `json:"circuitStates"`
	RunMetrics                 AIRunMetricsSnapshot `json:"runMetrics"`
}

// SearchRouteMetricsSnapshot contains one Meilisearch/index and search-mode
// aggregate. Query text and filters are never retained.
type SearchRouteMetricsSnapshot struct {
	Index         string     `json:"index"`
	Mode          SearchMode `json:"mode"`
	Total         uint64     `json:"total"`
	Succeeded     uint64     `json:"succeeded"`
	Failed        uint64     `json:"failed"`
	Canceled      uint64     `json:"canceled"`
	ErrorRate     float64    `json:"errorRate"`
	DurationP95MS int64      `json:"durationP95Ms"`
}

// SearchMetricsSnapshot is a bounded, process-local snapshot that can be
// scraped and aggregated by an external monitor on every application
// instance.
type SearchMetricsSnapshot struct {
	Total         uint64                       `json:"total"`
	Succeeded     uint64                       `json:"succeeded"`
	Failed        uint64                       `json:"failed"`
	Canceled      uint64                       `json:"canceled"`
	ErrorRate     float64                      `json:"errorRate"`
	DurationP95MS int64                        `json:"durationP95Ms"`
	Routes        []SearchRouteMetricsSnapshot `json:"routes"`
}

// OperationalMetricsSnapshot is the only application-facing aggregate. The
// concrete AI and search observers remain infrastructure concerns.
type OperationalMetricsSnapshot struct {
	GeneratedAt time.Time             `json:"generatedAt"`
	AI          AIOperationalSnapshot `json:"ai"`
	Search      SearchMetricsSnapshot `json:"search"`
}

// OperationalMetricsProvider supplies sanitized runtime diagnostics. It must
// not perform network calls, read credentials, or return request content.
type OperationalMetricsProvider interface {
	Snapshot(context.Context) OperationalMetricsSnapshot
}
