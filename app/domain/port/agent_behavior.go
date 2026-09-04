package port

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// AIJobKindAgentArticleReading is a durable, side-effect-free reading task.
// The task may later produce a review candidate, but it never publishes
// content by itself.
const AIJobKindAgentArticleReading = "agent.behavior.read_article"

const (
	AgentBehaviorOperationComment = "agent.comment"
	AgentBehaviorOperationTalk    = "agent.talk"
	AgentBehaviorOperationWake    = "agent.wake"
)

type AgentBehaviorAction string

const (
	AgentBehaviorActionComment AgentBehaviorAction = "comment"
	AgentBehaviorActionTalk    AgentBehaviorAction = "talk"
	AgentBehaviorActionWake    AgentBehaviorAction = "wake"
)

func NormalizeAgentBehaviorAction(value string) (AgentBehaviorAction, error) {
	action := AgentBehaviorAction(strings.ToLower(strings.TrimSpace(value)))
	switch action {
	case AgentBehaviorActionComment, AgentBehaviorActionTalk, AgentBehaviorActionWake:
		return action, nil
	default:
		return "", errors.New("agent behavior action is invalid")
	}
}

// AgentBehaviorTrigger explains why a reading task was selected. It is
// metadata only: both paths still produce a review candidate and neither can
// publish content directly.
type AgentBehaviorTrigger string

const (
	AgentBehaviorTriggerRecent    AgentBehaviorTrigger = "recent"
	AgentBehaviorTriggerForgotten AgentBehaviorTrigger = "forgotten"
)

func NormalizeAgentBehaviorTrigger(value string) (AgentBehaviorTrigger, error) {
	trigger := AgentBehaviorTrigger(strings.ToLower(strings.TrimSpace(value)))
	if trigger == "" {
		return AgentBehaviorTriggerRecent, nil
	}
	switch trigger {
	case AgentBehaviorTriggerRecent, AgentBehaviorTriggerForgotten:
		return trigger, nil
	default:
		return "", errors.New("agent behavior trigger is invalid")
	}
}

// AgentReadingTaskPayload is metadata only. The worker always reads the
// current article again, so a retry cannot publish stale article content.
type AgentReadingTaskPayload struct {
	SchemaVersion    int                  `json:"schemaVersion"`
	ArticleID        int                  `json:"articleId"`
	ArticleUpdatedAt time.Time            `json:"articleUpdatedAt"`
	Action           AgentBehaviorAction  `json:"action"`
	Trigger          AgentBehaviorTrigger `json:"trigger"`
	ProfileID        string               `json:"profileId"`
	PromptVersion    string               `json:"promptVersion"`
	OccurredAt       time.Time            `json:"occurredAt"`
}

func (p AgentReadingTaskPayload) Validate() error {
	if p.SchemaVersion <= 0 {
		return errors.New("agent reading task schema version must be positive")
	}
	if p.ArticleID <= 0 {
		return errors.New("agent reading task article id must be positive")
	}
	if _, err := NormalizeAgentBehaviorAction(string(p.Action)); err != nil {
		return err
	}
	if _, err := NormalizeAgentBehaviorTrigger(string(p.Trigger)); err != nil {
		return err
	}
	if strings.TrimSpace(p.ProfileID) == "" || len(p.ProfileID) > 128 {
		return errors.New("agent reading task profile id is invalid")
	}
	if strings.TrimSpace(p.PromptVersion) == "" || len(p.PromptVersion) > 128 {
		return errors.New("agent reading task prompt version is invalid")
	}
	if p.OccurredAt.IsZero() {
		return errors.New("agent reading task occurrence time is required")
	}
	return nil
}

func NewAgentReadingTaskPayload(articleID int, articleUpdatedAt time.Time, action AgentBehaviorAction, profileID, promptVersion string, occurredAt time.Time) ([]byte, error) {
	return NewAgentReadingTaskPayloadForTrigger(articleID, articleUpdatedAt, action, AgentBehaviorTriggerRecent, profileID, promptVersion, occurredAt)
}

func NewAgentReadingTaskPayloadForTrigger(articleID int, articleUpdatedAt time.Time, action AgentBehaviorAction, trigger AgentBehaviorTrigger, profileID, promptVersion string, occurredAt time.Time) ([]byte, error) {
	payload := AgentReadingTaskPayload{
		SchemaVersion:    1,
		ArticleID:        articleID,
		ArticleUpdatedAt: articleUpdatedAt.UTC(),
		Action:           action,
		Trigger:          trigger,
		ProfileID:        strings.TrimSpace(profileID),
		PromptVersion:    strings.TrimSpace(promptVersion),
		OccurredAt:       occurredAt.UTC(),
	}
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(payload)
}

type AgentBehaviorCandidateResult struct {
	Created  bool   `json:"created"`
	ReviewID string `json:"reviewId,omitempty"`
	RunID    string `json:"runId,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// AgentBehaviorGenerator creates review-only candidates from a reading task.
// Implementations must not expose a content publication method.
type AgentBehaviorGenerator interface {
	Generate(context.Context, AgentReadingTaskPayload) (AgentBehaviorCandidateResult, error)
}

// AgentArticleReader is the narrow article read contract needed by the
// behavior worker. It deliberately exposes the existing admin read method
// only as an application port; the worker still re-checks public visibility
// before sending any content to a model.
type AgentArticleReader interface {
	GetAdminArticle(context.Context, int) (article TArticle, categoryName string, tagNames []string, err error)
}
