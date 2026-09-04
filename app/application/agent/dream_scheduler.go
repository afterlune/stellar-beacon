package agent

import (
	"context"
	"strconv"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

type DreamSchedulerDeps struct {
	Sources       port.ArticleIndexSourceRepository
	Jobs          port.AIJobRepository
	ProfileID     string
	PromptVersion string
	BatchSize     int
	Now           func() time.Time
}

// DreamScheduler enqueues one deterministic, review-only candidate task per
// public article. The UTC day is part of the idempotency key, which gives the
// feature a bounded daily cadence without adding another scheduler service.
type DreamScheduler struct {
	sources       port.ArticleIndexSourceRepository
	jobs          port.AIJobRepository
	profileID     string
	promptVersion string
	batchSize     int
	now           func() time.Time
}

func NewDreamScheduler(deps DreamSchedulerDeps) (*DreamScheduler, error) {
	if deps.Sources == nil || deps.Jobs == nil {
		return nil, apperrors.Invalid("agent.dream.scheduler", "article source and job repository are required")
	}
	deps.ProfileID = strings.TrimSpace(deps.ProfileID)
	deps.PromptVersion = strings.TrimSpace(deps.PromptVersion)
	if deps.ProfileID == "" || len(deps.ProfileID) > 128 || deps.PromptVersion == "" || len(deps.PromptVersion) > 128 {
		return nil, apperrors.Invalid("agent.dream.scheduler", "profile and prompt version are required")
	}
	if deps.BatchSize <= 0 {
		deps.BatchSize = 20
	}
	if deps.BatchSize > 100 {
		return nil, apperrors.Invalid("agent.dream.scheduler", "scan batch size is too large")
	}
	return &DreamScheduler{
		sources:       deps.Sources,
		jobs:          deps.Jobs,
		profileID:     deps.ProfileID,
		promptVersion: deps.PromptVersion,
		batchSize:     deps.BatchSize,
		now:           deps.Now,
	}, nil
}

func (s *DreamScheduler) Scan(ctx context.Context, afterArticleID int) (nextArticleID, enqueued int, err error) {
	if s == nil || s.sources == nil || s.jobs == nil {
		return 0, 0, apperrors.Unavailable("agent.dream.scan", nil)
	}
	if afterArticleID < 0 {
		return 0, 0, apperrors.Invalid("agent.dream.scan", "article cursor cannot be negative")
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	sources, err := s.sources.ListPublicArticleIndexSources(ctx, afterArticleID, s.batchSize)
	if err != nil {
		return 0, 0, err
	}
	for _, source := range sources {
		if source.Article.Id > nextArticleID {
			nextArticleID = source.Article.Id
		}
		if !port.IsPublicArticle(source.Article.Status, source.Article.IsDelete) || source.Article.Id <= 0 {
			continue
		}
		key := DreamTaskIdempotencyKey(now, source.Article.Id, s.profileID, s.promptVersion)
		payload, payloadErr := port.NewDreamTaskPayload([]int{source.Article.Id}, s.profileID, s.promptVersion, key, now)
		if payloadErr != nil {
			return nextArticleID, enqueued, apperrors.Invalid("agent.dream.scan.payload", payloadErr.Error())
		}
		if enqueueErr := s.jobs.Enqueue(ctx, port.AIJob{
			ID:             DreamReviewID(key),
			Kind:           port.AIJobKindDreamCandidate,
			IdempotencyKey: key,
			Payload:        payload,
			Status:         port.AIJobPending,
			MaxAttempts:    port.DefaultAIJobMaxAttempts,
			RunAfter:       now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}); enqueueErr != nil {
			return nextArticleID, enqueued, enqueueErr
		}
		enqueued++
	}
	if len(sources) == 0 {
		return 0, 0, nil
	}
	return nextArticleID, enqueued, nil
}

func DreamTaskIdempotencyKey(now time.Time, articleID int, profileID, promptVersion string) string {
	return "agent-dream:v1:day:" + now.UTC().Format("2006-01-02") + ":article:" + strconv.Itoa(articleID) + ":profile:" + strings.TrimSpace(profileID) + ":prompt:" + strings.TrimSpace(promptVersion)
}
