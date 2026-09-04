package ai

import (
	"context"
	"strings"
	"sync"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
)

// ConfiguredAgentReviewPolicyRepository is the safe default policy store.
// It keeps review-required behavior available even when the explicit policy
// migration has not been applied to the database.
type ConfiguredAgentReviewPolicyRepository struct {
	mu       sync.RWMutex
	current  string
	policies map[string]port.AgentReviewPolicy
	err      error
}

func NewConfiguredAgentReviewPolicyRepository(settings config.AIAgentReviewPolicySettings, behavior config.AIAgentBehaviorSettings) *ConfiguredAgentReviewPolicyRepository {
	policy := port.DefaultAgentReviewPolicy()
	policy.ID = strings.TrimSpace(settings.ID)
	if policy.ID == "" {
		policy.ID = port.DefaultAgentReviewPolicyID
	}
	policy.ReviewTTL = behavior.ReviewTTL
	policy.MaxCandidateRunes = behavior.MaxCandidateRunes
	policy.SimilarityThreshold = behavior.SimilarityThreshold
	policy.DailyLimit = behavior.DailyLimit
	policy.PerArticleLimit = behavior.PerArticleLimit
	policy.PerActionLimit = behavior.PerActionLimit
	policy.SensitivePatterns = append([]string(nil), behavior.SensitivePatterns...)
	if len(behavior.AllowedActions) > 0 {
		policy.AllowedActions = make([]port.AgentBehaviorAction, 0, len(behavior.AllowedActions))
		for _, action := range behavior.AllowedActions {
			policy.AllowedActions = append(policy.AllowedActions, port.AgentBehaviorAction(action))
		}
	}
	var configErr error
	if normalized, err := port.NormalizeAgentReviewPolicy(policy); err == nil {
		policy = normalized
	} else {
		// Configuration is untrusted input too. Retain a safe value for
		// diagnostics, but make reads fail closed instead of silently
		// widening an invalid allowlist.
		configErr = apperrors.Invalid("agent.review_policy.config", err.Error())
		policy = port.DefaultAgentReviewPolicy()
	}
	return &ConfiguredAgentReviewPolicyRepository{
		current:  policy.ID,
		policies: map[string]port.AgentReviewPolicy{policy.ID: policy},
		err:      configErr,
	}
}

var _ port.AgentReviewPolicyRepository = (*ConfiguredAgentReviewPolicyRepository)(nil)

func (r *ConfiguredAgentReviewPolicyRepository) Get(ctx context.Context, id string) (port.AgentReviewPolicy, error) {
	if err := contextError(ctx); err != nil {
		return port.AgentReviewPolicy{}, err
	}
	if r == nil {
		return port.AgentReviewPolicy{}, apperrors.Unavailable("agent.review_policy.get", nil)
	}
	if r.err != nil {
		return port.AgentReviewPolicy{}, r.err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	id = strings.TrimSpace(id)
	if id == "" {
		id = r.current
	}
	policy, ok := r.policies[id]
	if !ok {
		return port.AgentReviewPolicy{}, apperrors.NotFound("agent.review_policy.get")
	}
	return cloneReviewPolicy(policy), nil
}

func (r *ConfiguredAgentReviewPolicyRepository) Save(ctx context.Context, input port.AgentReviewPolicy) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if r == nil {
		return apperrors.Unavailable("agent.review_policy.save", nil)
	}
	if r.err != nil {
		return r.err
	}
	policy, err := port.NormalizeAgentReviewPolicy(input)
	if err != nil {
		return apperrors.Invalid("agent.review_policy.save", err.Error())
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.policies[policy.ID]; ok {
		if policy.Version > 0 && policy.Version != existing.Version {
			return apperrors.Conflict("agent.review_policy.save", "review policy version is stale")
		}
		policy.Version = existing.Version + 1
	} else if policy.Version <= 0 {
		policy.Version = port.DefaultAgentReviewPolicyVersion
	}
	policy.UpdatedAt = time.Now().UTC()
	r.policies[policy.ID] = cloneReviewPolicy(policy)
	r.current = policy.ID
	return nil
}

func cloneReviewPolicy(policy port.AgentReviewPolicy) port.AgentReviewPolicy {
	policy.AllowedActions = append([]port.AgentBehaviorAction(nil), policy.AllowedActions...)
	policy.SensitivePatterns = append([]string(nil), policy.SensitivePatterns...)
	return policy
}
