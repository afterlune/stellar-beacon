package port

import (
	"context"
	"time"
)

// AgentTaskStatus is the durable state of a multi-step Agent task. It is
// intentionally separate from AIRunStatus, which describes one model call.
type AgentTaskStatus string

const (
	AgentTaskQueued          AgentTaskStatus = "queued"
	AgentTaskPlanning        AgentTaskStatus = "planning"
	AgentTaskRunning         AgentTaskStatus = "running"
	AgentTaskWaitingQuestion AgentTaskStatus = "waiting_question"
	AgentTaskWaitingApproval AgentTaskStatus = "waiting_approval"
	AgentTaskCompleted       AgentTaskStatus = "completed"
	AgentTaskFailed          AgentTaskStatus = "failed"
	AgentTaskCancelled       AgentTaskStatus = "cancelled"
)

func (s AgentTaskStatus) Terminal() bool {
	return s == AgentTaskCompleted || s == AgentTaskFailed || s == AgentTaskCancelled
}

// CanTransitionAgentTask is the single domain rule for task state changes.
// Persistence implementations must enforce it again with a conditional
// update so two workers cannot advance the same task concurrently.
func CanTransitionAgentTask(from, to AgentTaskStatus) bool {
	if from == to {
		return from == AgentTaskRunning
	}
	switch from {
	case AgentTaskQueued:
		return to == AgentTaskPlanning || to == AgentTaskCancelled
	case AgentTaskPlanning:
		return to == AgentTaskRunning || to == AgentTaskFailed || to == AgentTaskCancelled
	case AgentTaskRunning:
		return to == AgentTaskWaitingQuestion || to == AgentTaskWaitingApproval ||
			to == AgentTaskCompleted || to == AgentTaskFailed || to == AgentTaskCancelled
	case AgentTaskWaitingQuestion, AgentTaskWaitingApproval:
		return to == AgentTaskRunning || to == AgentTaskCancelled
	default:
		return false
	}
}

type AgentTaskRun struct {
	ID            string
	RequestID     string
	SessionID     string
	Goal          string
	Status        AgentTaskStatus
	PlanRevision  int
	ReplanCount   int
	QuestionCount int
	LeaseOwner    string
	LeaseUntil    time.Time
	WakeAt        time.Time
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	FinishedAt    time.Time
}

type AgentStepStatus string

const (
	AgentStepPending   AgentStepStatus = "pending"
	AgentStepRunning   AgentStepStatus = "running"
	AgentStepWaiting   AgentStepStatus = "waiting"
	AgentStepSucceeded AgentStepStatus = "succeeded"
	AgentStepFailed    AgentStepStatus = "failed"
	AgentStepSkipped   AgentStepStatus = "skipped"
	AgentStepCancelled AgentStepStatus = "cancelled"
)

type AgentPlanStep struct {
	RunID      string
	StepID     string
	Revision   int
	Ordinal    int
	Action     string
	Input      []byte
	DependsOn  []string
	Completion []byte
	Risk       string
	Status     AgentStepStatus
	Attempt    int
	Output     []byte
	LastError  string
	StartedAt  time.Time
	FinishedAt time.Time
	WakeAt     time.Time
}

type AgentPlanRevision struct {
	RunID     string
	Revision  int
	Plan      []byte
	Reason    string
	CreatedAt time.Time
}

type AgentQuestionStatus string

const (
	AgentQuestionPending   AgentQuestionStatus = "pending"
	AgentQuestionAnswered  AgentQuestionStatus = "answered"
	AgentQuestionExpired   AgentQuestionStatus = "expired"
	AgentQuestionCancelled AgentQuestionStatus = "cancelled"
)

type AgentQuestion struct {
	ID         string
	RunID      string
	StepID     string
	Prompt     string
	Options    []string
	Status     AgentQuestionStatus
	Answer     []byte
	CreatedAt  time.Time
	AnsweredAt time.Time
}

// AgentEffect is the durable idempotency record for a side effect. EffectKey
// must be unique in storage; a retry must observe the existing record instead
// of invoking the external operation again.
type AgentEffect struct {
	EffectKey string
	RunID     string
	StepID    string
	Tool      string
	Result    []byte
	CreatedAt time.Time
}

// AgentTaskRepository owns the durable state machine. Implementations must
// use conditional writes for ClaimNext, RenewLease, Transition and SaveEffect
// so a worker crash or duplicate request cannot create a second side effect.
type AgentTaskRepository interface {
	Create(context.Context, AgentTaskRun) error
	Get(context.Context, string) (AgentTaskRun, error)
	ClaimNext(context.Context, string, time.Time, time.Duration) (AgentTaskRun, bool, error)
	RenewLease(context.Context, string, string, time.Time, time.Duration) error
	RecoverStale(context.Context, time.Time) (int, error)
	Transition(context.Context, string, AgentTaskStatus, AgentTaskStatus, string) error
	SavePlanRevision(context.Context, AgentPlanRevision, []AgentPlanStep) error
	ListSteps(context.Context, string, int) ([]AgentPlanStep, error)
	SaveStep(context.Context, AgentPlanStep) error
	CreateQuestion(context.Context, AgentQuestion) error
	ListPendingQuestions(context.Context, string) ([]AgentQuestion, error)
	AnswerQuestion(context.Context, string, string, []byte, time.Time) error
	GetEffect(context.Context, string) (AgentEffect, bool, error)
	SaveEffect(context.Context, AgentEffect) error
}
