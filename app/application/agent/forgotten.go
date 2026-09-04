package agent

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"sort"
	"strings"
	"time"
)

const (
	DefaultForgottenAfter         = 90 * 24 * time.Hour
	DefaultForgottenMaxCandidates = 20
)

// ForgottenArticleSelector is deliberately deterministic. It identifies
// public, non-promoted articles that have not changed for a configured period;
// a model may write a candidate for the result, but it never decides whether
// an article is eligible or publishes anything.
type ForgottenArticleSelector struct {
	staleAfter    time.Duration
	maxCandidates int
}

type ForgottenArticleSelectorConfig struct {
	StaleAfter    time.Duration
	MaxCandidates int
}

func NewForgottenArticleSelector(config ForgottenArticleSelectorConfig) (*ForgottenArticleSelector, error) {
	if config.StaleAfter <= 0 {
		config.StaleAfter = DefaultForgottenAfter
	}
	if config.StaleAfter > 365*24*time.Hour {
		return nil, errors.Invalid("agent.behavior.forgotten", "stale period is too long")
	}
	if config.MaxCandidates <= 0 {
		config.MaxCandidates = DefaultForgottenMaxCandidates
	}
	if config.MaxCandidates > 500 {
		return nil, errors.Invalid("agent.behavior.forgotten", "candidate limit is too large")
	}
	return &ForgottenArticleSelector{staleAfter: config.StaleAfter, maxCandidates: config.MaxCandidates}, nil
}

// Select returns a new slice, ordered oldest-first and then by article ID.
// UpdateTime is preferred because edits revive an article; old rows without a
// valid update time fall back to CreateTime. Future timestamps are not treated
// as forgotten.
func (s *ForgottenArticleSelector) Select(now time.Time, sources []port.ArticleIndexSource) []port.ArticleIndexSource {
	if s == nil || len(sources) == 0 {
		return nil
	}
	cutoff := now.UTC().Add(-s.staleAfter)
	type candidate struct {
		source     port.ArticleIndexSource
		activityAt time.Time
	}
	candidates := make([]candidate, 0, len(sources))
	seen := make(map[int]struct{}, len(sources))
	for _, source := range sources {
		article := source.Article
		if article.Id <= 0 || !port.IsPublicArticle(article.Status, article.IsDelete) || article.IsTop != 0 || article.IsFeatured != 0 {
			continue
		}
		if _, exists := seen[article.Id]; exists {
			continue
		}
		seen[article.Id] = struct{}{}
		activityAt := article.UpdateTime
		if activityAt.IsZero() {
			activityAt = article.CreateTime
		}
		if activityAt.IsZero() || activityAt.After(cutoff) {
			continue
		}
		candidates = append(candidates, candidate{source: source, activityAt: activityAt.UTC()})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].activityAt.Equal(candidates[j].activityAt) {
			return candidates[i].source.Article.Id < candidates[j].source.Article.Id
		}
		return candidates[i].activityAt.Before(candidates[j].activityAt)
	})
	if len(candidates) > s.maxCandidates {
		candidates = candidates[:s.maxCandidates]
	}
	result := make([]port.ArticleIndexSource, 0, len(candidates))
	for _, item := range candidates {
		result = append(result, item.source)
	}
	return result
}

func (s *ForgottenArticleSelector) StaleAfter() time.Duration {
	if s == nil {
		return 0
	}
	return s.staleAfter
}

func (s *ForgottenArticleSelector) MaxCandidates() int {
	if s == nil {
		return 0
	}
	return s.maxCandidates
}

// ForgottenArticlePromptHint is kept small and provider-neutral. It is used
// by the candidate generator only as an instruction label, never as article
// content or an authorization signal.
func ForgottenArticlePromptHint(trigger port.AgentBehaviorTrigger) string {
	if strings.TrimSpace(string(trigger)) == string(port.AgentBehaviorTriggerForgotten) {
		return "这是一篇一段时间未更新的公开文章，请基于文章现有内容给出克制、具体的重新关注理由。"
	}
	return ""
}
