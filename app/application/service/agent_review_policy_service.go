package service

import (
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

type AgentReviewPolicyService interface {
	Get(port.Request) port.ResultVO
	Update(port.Request) port.ResultVO
}

type MyAgentReviewPolicyService struct {
	policies port.AgentReviewPolicyRepository
	policyID string
}

func NewAgentReviewPolicyService(policies port.AgentReviewPolicyRepository, policyID string) *MyAgentReviewPolicyService {
	return &MyAgentReviewPolicyService{policies: policies, policyID: strings.TrimSpace(policyID)}
}

func (s *MyAgentReviewPolicyService) Get(c port.Request) port.ResultVO {
	policy, err := s.get(c)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(agentReviewPolicyDTO(policy))
}

func (s *MyAgentReviewPolicyService) Update(c port.Request) port.ResultVO {
	if s == nil || s.policies == nil {
		return port.ResultFromError(apperrors.Unavailable("agent.review_policy.service", nil))
	}
	var request port.AgentReviewPolicyUpdateVO
	if err := c.BindJSON(&request); err != nil {
		return port.ResultFromError(apperrors.Invalid("agent.review_policy.request", "request body is invalid"))
	}
	policy, err := s.get(c)
	if err != nil {
		return port.ResultFromError(err)
	}
	if err := applyAgentReviewPolicyUpdate(&policy, request); err != nil {
		return port.ResultFromError(err)
	}
	if err := s.policies.Save(c.Context(), policy); err != nil {
		return port.ResultFromError(err)
	}
	updated, err := s.get(c)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(agentReviewPolicyDTO(updated))
}

func (s *MyAgentReviewPolicyService) get(c port.Request) (port.AgentReviewPolicy, error) {
	if s == nil || s.policies == nil {
		return port.AgentReviewPolicy{}, apperrors.Unavailable("agent.review_policy.service", nil)
	}
	policyID := s.policyID
	if policyID == "" {
		policyID = port.DefaultAgentReviewPolicyID
	}
	return s.policies.Get(c.Context(), policyID)
}

func applyAgentReviewPolicyUpdate(policy *port.AgentReviewPolicy, request port.AgentReviewPolicyUpdateVO) error {
	if policy == nil {
		return apperrors.Invalid("agent.review_policy.update", "policy is required")
	}
	if request.Version != nil {
		if *request.Version <= 0 {
			return apperrors.Invalid("agent.review_policy.version", "version must be positive")
		}
		policy.Version = *request.Version
	}
	if request.ReviewTTLSeconds != nil {
		const maxReviewTTLSeconds = int64((30 * 24 * time.Hour) / time.Second)
		if *request.ReviewTTLSeconds <= 0 || *request.ReviewTTLSeconds > maxReviewTTLSeconds {
			return apperrors.Invalid("agent.review_policy.ttl", "review ttl is invalid")
		}
		policy.ReviewTTL = time.Duration(*request.ReviewTTLSeconds) * time.Second
	}
	if request.MaxCandidateRunes != nil {
		policy.MaxCandidateRunes = *request.MaxCandidateRunes
	}
	if request.SimilarityThreshold != nil {
		policy.SimilarityThreshold = *request.SimilarityThreshold
	}
	if request.DailyLimit != nil {
		policy.DailyLimit = *request.DailyLimit
	}
	if request.PerArticleLimit != nil {
		policy.PerArticleLimit = *request.PerArticleLimit
	}
	if request.PerActionLimit != nil {
		policy.PerActionLimit = *request.PerActionLimit
	}
	if request.AllowedActions != nil {
		policy.AllowedActions = make([]port.AgentBehaviorAction, 0, len(*request.AllowedActions))
		for _, action := range *request.AllowedActions {
			policy.AllowedActions = append(policy.AllowedActions, port.AgentBehaviorAction(action))
		}
	}
	if request.SensitivePatterns != nil {
		policy.SensitivePatterns = append([]string(nil), (*request.SensitivePatterns)...)
	}
	policy.ReviewRequired = true
	if _, err := port.NormalizeAgentReviewPolicy(*policy); err != nil {
		return apperrors.Invalid("agent.review_policy.validate", err.Error())
	}
	return nil
}

func agentReviewPolicyDTO(policy port.AgentReviewPolicy) port.AgentReviewPolicyDTO {
	return port.AgentReviewPolicyDTO{
		ID:                  policy.ID,
		Version:             policy.Version,
		ReviewRequired:      policy.ReviewRequired,
		ReviewTTLSeconds:    int64(policy.ReviewTTL / time.Second),
		MaxCandidateRunes:   policy.MaxCandidateRunes,
		SimilarityThreshold: policy.SimilarityThreshold,
		DailyLimit:          policy.DailyLimit,
		PerArticleLimit:     policy.PerArticleLimit,
		PerActionLimit:      policy.PerActionLimit,
		AllowedActions:      policy.ActionNames(),
		SensitivePatterns:   append([]string(nil), policy.SensitivePatterns...),
		UpdatedAt:           policy.UpdatedAt,
	}
}
