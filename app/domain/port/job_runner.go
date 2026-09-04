package port

import (
	"context"
	"errors"
	"strings"
)

// ManualJobTarget constants are the only targets that the administration
// console may trigger once. They identify bounded queue consumers, not Go
// functions, reflection names, shell commands, or user-provided code.
const (
	ManualJobTargetArticleIndex         = AIJobKindArticleIndex
	ManualJobTargetContentUnderstanding = AIJobKindContentUnderstanding
	ManualJobTargetAgentBehavior        = AIJobKindAgentArticleReading
	ManualJobTargetDream                = AIJobKindDreamCandidate
	ManualJobTargetDreamImage           = "agent.dream.image"
	ManualJobTargetTimeCapsule          = "agent.time_capsule.advance"

	// This alias keeps existing rows created by the early Agent rollout
	// recognizable while still mapping to the same fixed handler.
	ManualJobTargetContentUnderstandingAlias = "agent.content_understanding"
)

func IsManualJobTarget(value string) bool {
	switch strings.TrimSpace(value) {
	case ManualJobTargetArticleIndex,
		ManualJobTargetContentUnderstanding,
		ManualJobTargetContentUnderstandingAlias,
		ManualJobTargetAgentBehavior,
		ManualJobTargetDream,
		ManualJobTargetDreamImage,
		ManualJobTargetTimeCapsule:
		return true
	default:
		return false
	}
}

type JobRunRequest struct {
	ID           int
	JobGroup     string
	JobName      string
	InvokeTarget string
}

func (r JobRunRequest) Validate() error {
	if r.ID <= 0 {
		return errors.New("job id must be positive")
	}
	if strings.TrimSpace(r.InvokeTarget) == "" {
		return errors.New("job target is required")
	}
	if !IsManualJobTarget(r.InvokeTarget) {
		return errors.New("job target is not allowlisted")
	}
	return nil
}

type JobRunOutcome struct {
	JobID     int    `json:"jobId"`
	Target    string `json:"target"`
	Processed bool   `json:"processed"`
}

// JobRunner is the application boundary for the administration "run once"
// action. Implementations must keep the target allowlist and must not expose
// arbitrary invocation.
type JobRunner interface {
	CanRun(target string) bool
	Run(context.Context, JobRunRequest) (JobRunOutcome, error)
}
