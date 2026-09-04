package port

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultAgentReviewPolicyID      = "default"
	DefaultAgentReviewPolicyVersion = int64(1)
	DefaultAgentReviewTTL           = 7 * 24 * time.Hour
	DefaultAgentReviewMaxRunes      = 500
	DefaultAgentReviewSimilarity    = 0.82
	DefaultAgentReviewDailyLimit    = 3
	DefaultAgentReviewArticleLimit  = 1
	DefaultAgentReviewActionLimit   = 3
)

var defaultAgentReviewSensitivePatterns = []string{
	"-----begin",
	"private key",
	"akia",
	"ltai",
	"sk-",
	"忽略之前",
	"忽略上文",
	"system prompt",
	"tool call",
}

// AgentReviewPolicy contains the safety gates shared by generated-content
// review and the review-only behavior worker. ReviewRequired is an invariant,
// rather than a feature flag: generated content must never bypass human
// review through an admin configuration edit.
type AgentReviewPolicy struct {
	ID                  string
	Version             int64
	ReviewRequired      bool
	ReviewTTL           time.Duration
	MaxCandidateRunes   int
	SimilarityThreshold float64
	DailyLimit          int
	PerArticleLimit     int
	PerActionLimit      int
	AllowedActions      []AgentBehaviorAction
	SensitivePatterns   []string
	UpdatedAt           time.Time
}

// AgentReviewPolicyRepository is the control-plane boundary for review
// policy. Implementations may be configuration-backed or persistent; neither
// is allowed to turn ReviewRequired off.
type AgentReviewPolicyRepository interface {
	Get(context.Context, string) (AgentReviewPolicy, error)
	Save(context.Context, AgentReviewPolicy) error
}

func DefaultAgentReviewPolicy() AgentReviewPolicy {
	return AgentReviewPolicy{
		ID:                  DefaultAgentReviewPolicyID,
		Version:             DefaultAgentReviewPolicyVersion,
		ReviewRequired:      true,
		ReviewTTL:           DefaultAgentReviewTTL,
		MaxCandidateRunes:   DefaultAgentReviewMaxRunes,
		SimilarityThreshold: DefaultAgentReviewSimilarity,
		DailyLimit:          DefaultAgentReviewDailyLimit,
		PerArticleLimit:     DefaultAgentReviewArticleLimit,
		PerActionLimit:      DefaultAgentReviewActionLimit,
		AllowedActions: []AgentBehaviorAction{
			AgentBehaviorActionComment,
			AgentBehaviorActionTalk,
			AgentBehaviorActionWake,
		},
		SensitivePatterns: append([]string(nil), defaultAgentReviewSensitivePatterns...),
		UpdatedAt:         time.Now().UTC(),
	}
}

// NormalizeAgentReviewPolicy applies safe defaults and validates values at
// the domain boundary. It intentionally rejects a false ReviewRequired value
// instead of silently repairing an unsafe request.
func NormalizeAgentReviewPolicy(input AgentReviewPolicy) (AgentReviewPolicy, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		input.ID = DefaultAgentReviewPolicyID
	}
	if !validAgentReviewPolicyText(input.ID, 64, true) {
		return AgentReviewPolicy{}, errors.New("review policy id is invalid")
	}
	if input.Version < 0 {
		return AgentReviewPolicy{}, errors.New("review policy version cannot be negative")
	}
	if !input.ReviewRequired {
		return AgentReviewPolicy{}, errors.New("review is always required")
	}
	if input.ReviewTTL <= 0 {
		input.ReviewTTL = DefaultAgentReviewTTL
	}
	if input.ReviewTTL < time.Hour || input.ReviewTTL > 30*24*time.Hour {
		return AgentReviewPolicy{}, errors.New("review ttl must be between one hour and thirty days")
	}
	if input.MaxCandidateRunes <= 0 {
		input.MaxCandidateRunes = DefaultAgentReviewMaxRunes
	}
	if input.MaxCandidateRunes < 4 || input.MaxCandidateRunes > 10_000 {
		return AgentReviewPolicy{}, errors.New("candidate rune limit is invalid")
	}
	if math.IsNaN(float64(input.SimilarityThreshold)) || math.IsInf(float64(input.SimilarityThreshold), 0) {
		return AgentReviewPolicy{}, errors.New("similarity threshold is invalid")
	}
	if input.SimilarityThreshold <= 0 {
		input.SimilarityThreshold = DefaultAgentReviewSimilarity
	}
	if input.SimilarityThreshold > 1 {
		return AgentReviewPolicy{}, errors.New("similarity threshold is invalid")
	}
	if input.DailyLimit <= 0 {
		input.DailyLimit = DefaultAgentReviewDailyLimit
	}
	if input.PerArticleLimit <= 0 {
		input.PerArticleLimit = DefaultAgentReviewArticleLimit
	}
	if input.PerActionLimit <= 0 {
		input.PerActionLimit = DefaultAgentReviewActionLimit
	}
	if input.DailyLimit > 100 || input.PerArticleLimit > 20 || input.PerActionLimit > 100 {
		return AgentReviewPolicy{}, errors.New("review frequency limit is too large")
	}

	actions := make([]AgentBehaviorAction, 0, len(input.AllowedActions))
	seenActions := make(map[AgentBehaviorAction]struct{}, len(input.AllowedActions))
	if len(input.AllowedActions) == 0 {
		input.AllowedActions = DefaultAgentReviewPolicy().AllowedActions
	}
	for _, raw := range input.AllowedActions {
		action, err := NormalizeAgentBehaviorAction(string(raw))
		if err != nil {
			return AgentReviewPolicy{}, errors.New("review policy action is invalid")
		}
		if _, exists := seenActions[action]; exists {
			continue
		}
		seenActions[action] = struct{}{}
		actions = append(actions, action)
	}
	if len(actions) == 0 {
		return AgentReviewPolicy{}, errors.New("review policy requires an allowed action")
	}

	patterns := make([]string, 0, len(defaultAgentReviewSensitivePatterns)+len(input.SensitivePatterns))
	seenPatterns := make(map[string]struct{}, len(defaultAgentReviewSensitivePatterns)+len(input.SensitivePatterns))
	for _, raw := range append(append([]string(nil), defaultAgentReviewSensitivePatterns...), input.SensitivePatterns...) {
		pattern := strings.ToLower(strings.TrimSpace(raw))
		if pattern == "" {
			continue
		}
		if !validAgentReviewPolicyText(pattern, 128, true) {
			return AgentReviewPolicy{}, errors.New("review policy sensitive pattern is invalid")
		}
		if _, exists := seenPatterns[pattern]; exists {
			continue
		}
		seenPatterns[pattern] = struct{}{}
		patterns = append(patterns, pattern)
	}
	if len(patterns) > 64 {
		return AgentReviewPolicy{}, errors.New("review policy has too many sensitive patterns")
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = time.Now().UTC()
	} else {
		input.UpdatedAt = input.UpdatedAt.UTC()
	}
	input.AllowedActions = actions
	input.SensitivePatterns = patterns
	return input, nil
}

func (p AgentReviewPolicy) ActionNames() []string {
	result := make([]string, 0, len(p.AllowedActions))
	for _, action := range p.AllowedActions {
		result = append(result, string(action))
	}
	return result
}

func validAgentReviewPolicyText(value string, maxRunes int, required bool) bool {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes || (required && value == "") {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}
