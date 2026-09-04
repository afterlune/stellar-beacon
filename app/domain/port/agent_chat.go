package port

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	PublicToolSearchArticles  = "search_articles"
	PublicToolReadArticle     = "read_article"
	PublicToolReadTaxonomy    = "read_categories_tags"
	PublicToolReadVitals      = "read_vitals"
	DefaultAgentSessionTTL    = 30 * time.Minute
	DefaultAgentProfileID     = "benetnasch-public"
	DefaultAgentPromptVersion = "v1"
	MaxAgentSessionIDLength   = 128
	MaxAgentRequestIDLength   = 128
)

// AgentRhythmPhase is decided by the server clock. It is deliberately a
// small closed set so a model cannot invent a time-of-day state.
type AgentRhythmPhase string

const (
	AgentRhythmAwake AgentRhythmPhase = "awake"
	AgentRhythmDusk  AgentRhythmPhase = "dusk"
	AgentRhythmNight AgentRhythmPhase = "night"
)

// AgentRhythmSnapshot is the provider-neutral result of the deterministic
// rhythm policy. LocalTime is formatted for display only; NextTransitionAt
// remains a timestamp in the configured location.
type AgentRhythmSnapshot struct {
	Phase            AgentRhythmPhase `json:"phase"`
	Timezone         string           `json:"timezone"`
	LocalTime        string           `json:"localTime"`
	NextTransitionAt time.Time        `json:"nextTransitionAt"`
}

// AgentRhythm turns a timestamp into a deterministic server-owned state.
// The application never asks a model to infer the current phase.
type AgentRhythm interface {
	Snapshot(time.Time) AgentRhythmSnapshot
}

// AgentFeatureFlags is the public, non-secret rollout contract consumed by
// the blog frontend. Every experience, including browser TTS, has its own
// rollout bit and is fail-closed when the feature endpoint cannot be read.
type AgentFeatureFlags struct {
	PublicChat bool `json:"publicChat"`
	Vitals     bool `json:"vitals"`
	Galaxy     bool `json:"galaxy"`
	Dreams     bool `json:"dreams"`
	Capsules   bool `json:"capsules"`
	Radio      bool `json:"radio"`
	Videos     bool `json:"videos"`
	TTSEnabled bool `json:"ttsEnabled"`
}

// AgentSafetySwitch is a cross-instance emergency stop for public Agent
// entry points and autonomous workers. Implementations must fail closed when
// the switch cannot be read.
type AgentSafetySwitch interface {
	IsStopped(context.Context) (bool, error)
	SetStopped(context.Context, bool) error
}

// AgentSession is the short-lived, anonymous conversation state. It is a
// deliberately small context window: long-term memory is a later milestone
// and must never be inferred from an anonymous session.
type AgentSession struct {
	ID            string        `json:"id"`
	PromptVersion string        `json:"promptVersion"`
	Messages      []ChatMessage `json:"messages"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// AgentSessionStore is the only persistence contract for public anonymous
// sessions. The owner key is opaque to the store; implementations must scope
// the key so another visitor cannot load a known session ID.
type AgentSessionStore interface {
	Load(context.Context, string, string) (AgentSession, bool, error)
	Save(context.Context, AgentSession, string) error
	Delete(context.Context, string, string) error
}

// AgentSessionCoordinator serializes turns for one owner/session pair. A
// session read followed by model execution and save is not a single Redis
// operation, so callers need a lease to prevent concurrent turns from
// overwriting each other's context. Implementations must scope the lease by
// both owner and session and release it safely when ownership changes.
type AgentSessionCoordinator interface {
	Acquire(context.Context, string, string) (AgentSessionLease, error)
}

// AgentSessionLease is an ownership token for one session turn. KeepAlive
// blocks until its context is canceled and renews the ownership token before
// it expires. A non-nil KeepAlive error means ownership can no longer be
// proven and the active turn must be canceled. Release is deliberately
// context-free so request cancellation cannot prevent cleanup; implementations
// should use their own short bounded cleanup context.
type AgentSessionLease interface {
	KeepAlive(context.Context) error
	Release() error
}

// AgentEventStore keeps only short-lived public SSE events in the cache. It
// is not a durable audit log and is scoped by owner/session/turn so a caller
// cannot replay another visitor's conversation or mix turns.
type AgentEventStore interface {
	Append(context.Context, string, AgentChatEvent) error
	Replay(context.Context, string, string, string, int64) ([]AgentChatEvent, error)
	Delete(context.Context, string, string) error
}

type AgentChatRequest struct {
	SessionID string
	Message   string
	OwnerKey  string
	RequestID string
	TimeRange KnowledgeFilterInput
	// Privileged is set only by a trusted internal/admin composition path; the
	// public HTTP decoder never accepts it from request JSON.
	Privileged bool
}

type AgentQuotaDecision struct {
	Allowed   bool
	Remaining int64
	ResetAt   time.Time
}

// AgentTurnQuota is intentionally separate from the general cache port. It
// gives application code a single atomic decision instead of allowing a
// caller to accidentally implement check-then-increment races.
type AgentTurnQuota interface {
	Allow(context.Context, string, bool) (AgentQuotaDecision, error)
}

// AgentChatEvent is the application event sent to the SSE facade. Provider
// SDK types never cross this boundary, and tool-call internals are not
// exposed to public clients.
type AgentChatEvent struct {
	EventID   string
	Seq       int64
	Replay    bool
	Kind      StreamEventKind
	SessionID string
	TurnID    string
	RunID     string
	Provider  string
	Protocol  ProviderProtocol
	Model     string
	Text      string
	Opening   string
	Citation  *Citation
	State     map[string]string
	Usage     *TokenUsage
}

type PublicAgentToolResult struct {
	Content   string
	Citations []Citation
	State     map[string]string
}

// PublicAgentToolRegistry is an explicit allowlist. There is intentionally
// no generic database/query or write method in this contract.
type PublicAgentToolRegistry interface {
	Definitions() []ToolDefinition
	Execute(context.Context, string, json.RawMessage) (PublicAgentToolResult, error)
}

type AgentVitals struct {
	Status             string             `json:"status"`
	LifeStage          string             `json:"lifeStage"`
	Emotion            string             `json:"emotion"`
	EmotionScores      map[string]float64 `json:"emotionScores,omitempty"`
	Phase              AgentRhythmPhase   `json:"phase"`
	Timezone           string             `json:"timezone"`
	LocalTime          string             `json:"localTime"`
	NextTransitionAt   time.Time          `json:"nextTransitionAt"`
	ArticleCount       int64              `json:"articleCount"`
	CategoryCount      int64              `json:"categoryCount"`
	TagCount           int64              `json:"tagCount"`
	TalkCount          int64              `json:"talkCount"`
	ContentCount       int64              `json:"contentCount"`
	RecentContentCount int64              `json:"recentContentCount"`
	ViewCount          int64              `json:"viewCount"`
	UniqueVisitorCount int64              `json:"uniqueVisitorCount"`
	UpdatedAt          time.Time          `json:"updatedAt"`
}

type AgentVitalsProvider interface {
	Snapshot(context.Context) (AgentVitals, error)
}

func NormalizeAgentSessionID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > MaxAgentSessionIDLength {
		return "", errors.New("session id is too long")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return "", errors.New("session id contains invalid characters")
	}
	return value, nil
}

func NormalizeAgentRequestID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > MaxAgentRequestIDLength {
		return "", errors.New("request id is too long")
	}
	return value, nil
}
