package agent

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type BehaviorSchedulerDeps struct {
	Sources           port.ArticleIndexSourceRepository
	Jobs              port.AIJobRepository
	Policy            *BehaviorPolicy
	ForgottenSelector *ForgottenArticleSelector
	BatchSize         int
	Now               func() time.Time
}

// BehaviorScheduler turns a public article snapshot into durable reading
// tasks. It has no model or publication dependency, so scanning is safe to
// run repeatedly and can remain enabled independently of candidate generation.
type BehaviorScheduler struct {
	sources           port.ArticleIndexSourceRepository
	jobs              port.AIJobRepository
	policy            *BehaviorPolicy
	forgottenSelector *ForgottenArticleSelector
	batchSize         int
	now               func() time.Time
}

func NewBehaviorScheduler(deps BehaviorSchedulerDeps) (*BehaviorScheduler, error) {
	if deps.Sources == nil || deps.Jobs == nil || deps.Policy == nil {
		return nil, errors.Invalid("agent.behavior.scheduler", "article source, job repository and policy are required")
	}
	if deps.BatchSize <= 0 {
		deps.BatchSize = 20
	}
	if deps.BatchSize > 500 {
		return nil, errors.Invalid("agent.behavior.scheduler", "scan batch size is too large")
	}
	return &BehaviorScheduler{
		sources:           deps.Sources,
		jobs:              deps.Jobs,
		policy:            deps.Policy,
		forgottenSelector: deps.ForgottenSelector,
		batchSize:         deps.BatchSize,
		now:               deps.Now,
	}, nil
}

// Scan enqueues one reading task per allowed action. The returned cursor is
// the last article ID in the page; zero means the caller should wrap around
// on the next scan. Enqueue is idempotent in the durable repository.
func (s *BehaviorScheduler) Scan(ctx context.Context, afterArticleID int) (nextArticleID, enqueued int, err error) {
	if s == nil || s.sources == nil || s.jobs == nil || s.policy == nil {
		return 0, 0, errors.Unavailable("agent.behavior.scan", nil)
	}
	if afterArticleID < 0 {
		return 0, 0, errors.Invalid("agent.behavior.scan", "article cursor cannot be negative")
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	sources, err := s.sources.ListPublicArticleIndexSources(ctx, afterArticleID, s.batchSize)
	if err != nil {
		return 0, 0, err
	}
	return s.enqueue(ctx, sources, sources, now, port.AgentBehaviorTriggerRecent)
}

// ScanForgotten selects an independent, bounded page of old public articles.
// Its jobs use a separate trigger and idempotency key, so a recent-article
// task cannot accidentally consume the forgotten-content policy.
func (s *BehaviorScheduler) ScanForgotten(ctx context.Context, afterArticleID int) (nextArticleID, enqueued int, err error) {
	if s == nil || s.sources == nil || s.jobs == nil || s.policy == nil || s.forgottenSelector == nil {
		return 0, 0, errors.Unavailable("agent.behavior.scan_forgotten", nil)
	}
	if afterArticleID < 0 {
		return 0, 0, errors.Invalid("agent.behavior.scan_forgotten", "article cursor cannot be negative")
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	sources, err := s.sources.ListPublicArticleIndexSources(ctx, afterArticleID, s.batchSize)
	if err != nil {
		return 0, 0, err
	}
	selected := s.forgottenSelector.Select(now, sources)
	return s.enqueue(ctx, selected, sources, now, port.AgentBehaviorTriggerForgotten)
}

func (s *BehaviorScheduler) enqueue(ctx context.Context, sources, cursorSources []port.ArticleIndexSource, now time.Time, trigger port.AgentBehaviorTrigger) (nextArticleID, enqueued int, err error) {
	for _, source := range sources {
		if !port.IsPublicArticle(source.Article.Status, source.Article.IsDelete) || source.Article.Id <= 0 {
			continue
		}
		nextArticleID = source.Article.Id
		for _, action := range s.policy.Actions() {
			payload, payloadErr := port.NewAgentReadingTaskPayloadForTrigger(
				source.Article.Id,
				source.Article.UpdateTime,
				action,
				trigger,
				s.policy.ProfileID(),
				s.policy.PromptVersion(),
				now,
			)
			if payloadErr != nil {
				return nextArticleID, enqueued, errors.Invalid("agent.behavior.scan.payload", payloadErr.Error())
			}
			job := port.AIJob{
				ID:             "agent-read-" + strconv.Itoa(source.Article.Id) + "-" + string(trigger) + "-" + string(action),
				Kind:           port.AIJobKindAgentArticleReading,
				IdempotencyKey: behaviorReadingIdempotencyKeyForTrigger(source.Article.Id, action, trigger, s.policy.ProfileID(), s.policy.PromptVersion()),
				Payload:        payload,
				Status:         port.AIJobPending,
				MaxAttempts:    port.DefaultAIJobMaxAttempts,
				RunAfter:       now,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := s.jobs.Enqueue(ctx, job); err != nil {
				return nextArticleID, enqueued, err
			}
			enqueued++
		}
	}
	if len(cursorSources) == 0 {
		return 0, 0, nil
	}
	if nextArticleID == 0 {
		for _, source := range cursorSources {
			if source.Article.Id > nextArticleID {
				nextArticleID = source.Article.Id
			}
		}
	}
	return nextArticleID, enqueued, nil
}

func behaviorReadingIdempotencyKey(articleID int, action port.AgentBehaviorAction, profileID, promptVersion string) string {
	return behaviorReadingIdempotencyKeyForTrigger(articleID, action, port.AgentBehaviorTriggerRecent, profileID, promptVersion)
}

func behaviorReadingIdempotencyKeyForTrigger(articleID int, action port.AgentBehaviorAction, trigger port.AgentBehaviorTrigger, profileID, promptVersion string) string {
	return "agent-reading:v1:article:" + strconv.Itoa(articleID) + ":trigger:" + string(trigger) + ":action:" + string(action) + ":profile:" + strings.TrimSpace(profileID) + ":prompt:" + strings.TrimSpace(promptVersion)
}

func DecodeAgentReadingTask(payload []byte) (port.AgentReadingTaskPayload, error) {
	var task port.AgentReadingTaskPayload
	if err := json.Unmarshal(payload, &task); err != nil {
		return port.AgentReadingTaskPayload{}, err
	}
	if err := task.Validate(); err != nil {
		return port.AgentReadingTaskPayload{}, err
	}
	return task, nil
}
