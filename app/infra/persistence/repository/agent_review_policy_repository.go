package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"xorm.io/xorm"
)

// MyAgentReviewPolicyRepository stores the administrator-controlled review
// gates. The table is intentionally introduced by an explicit migration and
// is only selected by bootstrap when persistence is opted in.
type MyAgentReviewPolicyRepository struct {
	engine *xorm.Engine
}

func NewAgentReviewPolicyRepository(engine *xorm.Engine) *MyAgentReviewPolicyRepository {
	return &MyAgentReviewPolicyRepository{engine: engine}
}

var _ port.AgentReviewPolicyRepository = (*MyAgentReviewPolicyRepository)(nil)

type agentReviewPolicyRow struct {
	ID                  string    `xorm:"id"`
	Version             int64     `xorm:"version"`
	ReviewRequired      bool      `xorm:"review_required"`
	ReviewTTLSeconds    int64     `xorm:"review_ttl_seconds"`
	MaxCandidateRunes   int       `xorm:"max_candidate_runes"`
	SimilarityThreshold float64   `xorm:"similarity_threshold"`
	DailyLimit          int       `xorm:"daily_limit"`
	PerArticleLimit     int       `xorm:"per_article_limit"`
	PerActionLimit      int       `xorm:"per_action_limit"`
	AllowedActions      string    `xorm:"allowed_actions"`
	SensitivePatterns   string    `xorm:"sensitive_patterns"`
	UpdatedAt           time.Time `xorm:"updated_at"`
}

const agentReviewPolicyColumns = `id, version, review_required, review_ttl_seconds,
max_candidate_runes, similarity_threshold, daily_limit, per_article_limit,
per_action_limit, allowed_actions::text AS allowed_actions,
sensitive_patterns::text AS sensitive_patterns, updated_at`

const maxAgentReviewTTLSeconds int64 = int64((30 * 24 * time.Hour) / time.Second)

func (r *MyAgentReviewPolicyRepository) Get(ctx context.Context, id string) (port.AgentReviewPolicy, error) {
	if r == nil {
		return port.AgentReviewPolicy{}, apperrors.Unavailable("agent.review_policy.get.database", nil)
	}
	id = strings.TrimSpace(id)
	if id == "" {
		id = port.DefaultAgentReviewPolicyID
	}
	session, err := repoSession(r.engine, ctx, "agent.review_policy.get")
	if err != nil {
		return port.AgentReviewPolicy{}, err
	}
	defer session.Close()
	var row agentReviewPolicyRow
	found, err := session.SQL("SELECT "+agentReviewPolicyColumns+" FROM t_agent_review_policy WHERE id = ?", id).Get(&row)
	if err != nil {
		return port.AgentReviewPolicy{}, apperrors.Unavailable("agent.review_policy.get", err)
	}
	if !found {
		return port.AgentReviewPolicy{}, apperrors.NotFound("agent.review_policy.get")
	}
	policy, err := row.policy()
	if err != nil {
		return port.AgentReviewPolicy{}, apperrors.Unavailable("agent.review_policy.decode", err)
	}
	return policy, nil
}

func (r *MyAgentReviewPolicyRepository) Save(ctx context.Context, input port.AgentReviewPolicy) error {
	if r == nil {
		return apperrors.Unavailable("agent.review_policy.save.database", nil)
	}
	policy, err := port.NormalizeAgentReviewPolicy(input)
	if err != nil {
		return apperrors.Invalid("agent.review_policy.save", err.Error())
	}
	allowedActions, err := json.Marshal(policy.ActionNames())
	if err != nil {
		return apperrors.Invalid("agent.review_policy.save", "allowed actions cannot be encoded")
	}
	sensitivePatterns, err := json.Marshal(policy.SensitivePatterns)
	if err != nil {
		return apperrors.Invalid("agent.review_policy.save", "sensitive patterns cannot be encoded")
	}
	return repoTx(r.engine, ctx, "agent.review_policy.save", func(session *xorm.Session) error {
		var existing agentReviewPolicyRow
		found, err := session.SQL("SELECT "+agentReviewPolicyColumns+" FROM t_agent_review_policy WHERE id = ? FOR UPDATE", policy.ID).Get(&existing)
		if err != nil {
			return err
		}
		version := policy.Version
		if found {
			if version > 0 && version != existing.Version {
				return apperrors.Conflict("agent.review_policy.save", "review policy version is stale")
			}
			version = existing.Version + 1
		} else if version <= 0 {
			version = port.DefaultAgentReviewPolicyVersion
		}
		now := time.Now().UTC()
		if found {
			result, err := session.Exec(`
UPDATE t_agent_review_policy
SET version = ?, review_required = TRUE, review_ttl_seconds = ?,
    max_candidate_runes = ?, similarity_threshold = ?, daily_limit = ?,
    per_article_limit = ?, per_action_limit = ?, allowed_actions = ?::jsonb,
    sensitive_patterns = ?::jsonb, updated_at = ?
WHERE id = ? AND version = ?`,
				version, int64(policy.ReviewTTL/time.Second), policy.MaxCandidateRunes,
				policy.SimilarityThreshold, policy.DailyLimit, policy.PerArticleLimit,
				policy.PerActionLimit, string(allowedActions), string(sensitivePatterns),
				now, policy.ID, existing.Version)
			if err != nil {
				return err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected != 1 {
				return apperrors.Conflict("agent.review_policy.save", "review policy was changed concurrently")
			}
			return nil
		}
		_, err = session.Exec(`
INSERT INTO t_agent_review_policy
    (id, version, review_required, review_ttl_seconds, max_candidate_runes,
     similarity_threshold, daily_limit, per_article_limit, per_action_limit,
     allowed_actions, sensitive_patterns, updated_at)
VALUES (?, ?, TRUE, ?, ?, ?, ?, ?, ?, ?::jsonb, ?::jsonb, ?)`,
			policy.ID, version, int64(policy.ReviewTTL/time.Second), policy.MaxCandidateRunes,
			policy.SimilarityThreshold, policy.DailyLimit, policy.PerArticleLimit,
			policy.PerActionLimit, string(allowedActions), string(sensitivePatterns), now)
		return err
	})
}

func (r agentReviewPolicyRow) policy() (port.AgentReviewPolicy, error) {
	if r.ReviewTTLSeconds <= 0 || r.ReviewTTLSeconds > maxAgentReviewTTLSeconds {
		return port.AgentReviewPolicy{}, errors.New("review policy ttl is outside the supported range")
	}
	var actions []string
	if err := json.Unmarshal([]byte(r.AllowedActions), &actions); err != nil {
		return port.AgentReviewPolicy{}, err
	}
	var patterns []string
	if err := json.Unmarshal([]byte(r.SensitivePatterns), &patterns); err != nil {
		return port.AgentReviewPolicy{}, err
	}
	allowed := make([]port.AgentBehaviorAction, 0, len(actions))
	for _, action := range actions {
		allowed = append(allowed, port.AgentBehaviorAction(action))
	}
	return port.NormalizeAgentReviewPolicy(port.AgentReviewPolicy{
		ID:                  strings.TrimSpace(r.ID),
		Version:             r.Version,
		ReviewRequired:      r.ReviewRequired,
		ReviewTTL:           time.Duration(r.ReviewTTLSeconds) * time.Second,
		MaxCandidateRunes:   r.MaxCandidateRunes,
		SimilarityThreshold: r.SimilarityThreshold,
		DailyLimit:          r.DailyLimit,
		PerArticleLimit:     r.PerArticleLimit,
		PerActionLimit:      r.PerActionLimit,
		AllowedActions:      allowed,
		SensitivePatterns:   patterns,
		UpdatedAt:           r.UpdatedAt,
	})
}
