package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

var _ port.AgentTaskRepository = (*MyAgentTaskRepository)(nil)

type MyAgentTaskRepository struct {
	engine *xorm.Engine
}

func NewAgentTaskRepository(engine *xorm.Engine) *MyAgentTaskRepository {
	return &MyAgentTaskRepository{engine: engine}
}

type agentTaskRunRow struct {
	ID            string    `xorm:"id"`
	RequestID     string    `xorm:"request_id"`
	SessionID     string    `xorm:"session_id"`
	Goal          string    `xorm:"goal"`
	Status        string    `xorm:"status"`
	PlanRevision  int       `xorm:"plan_revision"`
	ReplanCount   int       `xorm:"replan_count"`
	QuestionCount int       `xorm:"question_count"`
	LeaseOwner    string    `xorm:"lease_owner"`
	LeaseUntil    time.Time `xorm:"lease_until"`
	WakeAt        time.Time `xorm:"wake_at"`
	LastError     string    `xorm:"last_error"`
	CreatedAt     time.Time `xorm:"created_at"`
	UpdatedAt     time.Time `xorm:"updated_at"`
	FinishedAt    time.Time `xorm:"finished_at"`
}

const agentTaskRunColumns = `id, request_id, session_id, goal, status, plan_revision,
replan_count, question_count, COALESCE(lease_owner, '') AS lease_owner,
COALESCE(lease_until, 'epoch'::timestamptz) AS lease_until, wake_at,
COALESCE(last_error, '') AS last_error, created_at, updated_at,
COALESCE(finished_at, 'epoch'::timestamptz) AS finished_at`

func (r agentTaskRunRow) taskRun() port.AgentTaskRun {
	return port.AgentTaskRun{
		ID:            r.ID,
		RequestID:     r.RequestID,
		SessionID:     r.SessionID,
		Goal:          r.Goal,
		Status:        port.AgentTaskStatus(r.Status),
		PlanRevision:  r.PlanRevision,
		ReplanCount:   r.ReplanCount,
		QuestionCount: r.QuestionCount,
		LeaseOwner:    r.LeaseOwner,
		LeaseUntil:    postgresEpochToZero(r.LeaseUntil),
		WakeAt:        r.WakeAt.UTC(),
		LastError:     r.LastError,
		CreatedAt:     r.CreatedAt.UTC(),
		UpdatedAt:     r.UpdatedAt.UTC(),
		FinishedAt:    postgresEpochToZero(r.FinishedAt),
	}
}

func (r *MyAgentTaskRepository) Create(ctx context.Context, input port.AgentTaskRun) error {
	run, err := normalizeAgentTaskRun(input)
	if err != nil {
		return apperrors.Invalid("agent_task.create", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("agent_task.create.database", nil)
	}
	return repoTx(r.engine, ctx, "agent_task.create", func(session *xorm.Session) error {
		var existing agentTaskRunRow
		found, err := session.SQL("SELECT "+agentTaskRunColumns+" FROM t_agent_task_run WHERE id = ? FOR UPDATE", run.ID).Get(&existing)
		if err != nil {
			return err
		}
		if found {
			if existing.Goal != run.Goal || existing.RequestID != run.RequestID || existing.SessionID != run.SessionID {
				return apperrors.Conflict("agent_task.create", "task id belongs to another goal")
			}
			return nil
		}
		leaseUntil := nullableTime(run.LeaseUntil)
		finishedAt := nullableTime(run.FinishedAt)
		_, err = session.Exec(`
INSERT INTO t_agent_task_run
    (id, request_id, session_id, goal, status, plan_revision, replan_count, question_count,
     lease_owner, lease_until, wake_at, last_error, created_at, updated_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			run.ID, run.RequestID, run.SessionID, run.Goal, string(run.Status), run.PlanRevision,
			run.ReplanCount, run.QuestionCount, nullableString(run.LeaseOwner), leaseUntil,
			run.WakeAt, run.LastError, run.CreatedAt, run.UpdatedAt, finishedAt)
		return err
	})
}

func (r *MyAgentTaskRepository) Get(ctx context.Context, id string) (port.AgentTaskRun, error) {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 {
		return port.AgentTaskRun{}, apperrors.Invalid("agent_task.get", "task id is invalid")
	}
	if r == nil {
		return port.AgentTaskRun{}, apperrors.Unavailable("agent_task.get.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_task.get")
	if err != nil {
		return port.AgentTaskRun{}, err
	}
	defer session.Close()
	var row agentTaskRunRow
	found, err := session.SQL("SELECT "+agentTaskRunColumns+" FROM t_agent_task_run WHERE id = ?", id).Get(&row)
	if err != nil {
		return port.AgentTaskRun{}, apperrors.Unavailable("agent_task.get", err)
	}
	if !found {
		return port.AgentTaskRun{}, apperrors.NotFound("agent_task.get")
	}
	return row.taskRun(), nil
}

func (r *MyAgentTaskRepository) ClaimNext(ctx context.Context, workerID string, now time.Time, lease time.Duration) (port.AgentTaskRun, bool, error) {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" || len(workerID) > 128 {
		return port.AgentTaskRun{}, false, apperrors.Invalid("agent_task.claim", "worker id is invalid")
	}
	if lease <= 0 {
		return port.AgentTaskRun{}, false, apperrors.Invalid("agent_task.claim", "lease must be positive")
	}
	if r == nil {
		return port.AgentTaskRun{}, false, apperrors.Unavailable("agent_task.claim.database", nil)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	session, err := repoSession(r.engine, ctx, "agent_task.claim")
	if err != nil {
		return port.AgentTaskRun{}, false, err
	}
	defer session.Close()
	if err := session.Begin(); err != nil {
		return port.AgentTaskRun{}, false, apperrors.Unavailable("agent_task.claim.begin", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = session.Rollback()
		}
	}()
	var row agentTaskRunRow
	found, err := session.SQL(`SELECT `+agentTaskRunColumns+` FROM t_agent_task_run
WHERE status = ? AND wake_at <= ?
ORDER BY created_at ASC, id ASC
FOR UPDATE SKIP LOCKED LIMIT 1`, string(port.AgentTaskQueued), now).Get(&row)
	if err != nil {
		return port.AgentTaskRun{}, false, apperrors.Unavailable("agent_task.claim", err)
	}
	if !found {
		if err := session.Commit(); err != nil {
			return port.AgentTaskRun{}, false, apperrors.Unavailable("agent_task.claim.commit", err)
		}
		committed = true
		return port.AgentTaskRun{}, false, nil
	}
	leaseUntil := now.Add(lease)
	result, err := session.Exec(`
UPDATE t_agent_task_run
SET status = ?, lease_owner = ?, lease_until = ?, updated_at = ?
WHERE id = ? AND status = ?`,
		string(port.AgentTaskPlanning), workerID, leaseUntil, now, row.ID, string(port.AgentTaskQueued))
	if err != nil {
		return port.AgentTaskRun{}, false, apperrors.Unavailable("agent_task.claim.update", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		if err != nil {
			return port.AgentTaskRun{}, false, apperrors.Unavailable("agent_task.claim.rows", err)
		}
		return port.AgentTaskRun{}, false, apperrors.Conflict("agent_task.claim", "task was claimed by another worker")
	}
	if err := session.Commit(); err != nil {
		return port.AgentTaskRun{}, false, apperrors.Unavailable("agent_task.claim.commit", err)
	}
	committed = true
	row.Status = string(port.AgentTaskPlanning)
	row.LeaseOwner = workerID
	row.LeaseUntil = leaseUntil
	row.UpdatedAt = now
	return row.taskRun(), true, nil
}

func (r *MyAgentTaskRepository) RenewLease(ctx context.Context, id, workerID string, now time.Time, lease time.Duration) error {
	id = strings.TrimSpace(id)
	workerID = strings.TrimSpace(workerID)
	if id == "" || workerID == "" || len(id) > 128 || len(workerID) > 128 {
		return apperrors.Invalid("agent_task.renew", "task id and worker id are required")
	}
	if lease <= 0 {
		return apperrors.Invalid("agent_task.renew", "lease must be positive")
	}
	if r == nil {
		return apperrors.Unavailable("agent_task.renew.database", nil)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	session, err := repoSession(r.engine, ctx, "agent_task.renew")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_agent_task_run
SET lease_until = ?, updated_at = ?
WHERE id = ? AND lease_owner = ? AND lease_until IS NOT NULL AND lease_until > ?
  AND status IN (?, ?)`,
		now.Add(lease), now, id, workerID, now, string(port.AgentTaskPlanning), string(port.AgentTaskRunning))
	if err != nil {
		return apperrors.Unavailable("agent_task.renew", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("agent_task.renew.rows", err)
	}
	if affected != 1 {
		return apperrors.Conflict("agent_task.renew", "task lease is not owned or has expired")
	}
	return nil
}

func (r *MyAgentTaskRepository) RecoverStale(ctx context.Context, now time.Time) (int, error) {
	if r == nil {
		return 0, apperrors.Unavailable("agent_task.recover.database", nil)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	session, err := repoSession(r.engine, ctx, "agent_task.recover")
	if err != nil {
		return 0, err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_agent_task_run
SET status = ?, lease_owner = NULL, lease_until = NULL,
    last_error = CASE WHEN last_error = '' THEN ? ELSE last_error END, updated_at = ?
WHERE status IN (?, ?) AND lease_until IS NOT NULL AND lease_until <= ?`,
		string(port.AgentTaskQueued), "worker lease expired; task requeued", now,
		string(port.AgentTaskPlanning), string(port.AgentTaskRunning), now)
	if err != nil {
		return 0, apperrors.Unavailable("agent_task.recover", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Unavailable("agent_task.recover.rows", err)
	}
	return int(affected), nil
}

func (r *MyAgentTaskRepository) Transition(ctx context.Context, id string, from, to port.AgentTaskStatus, reason string) error {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 {
		return apperrors.Invalid("agent_task.transition", "task id is invalid")
	}
	if !validAgentTaskStatus(from) || !validAgentTaskStatus(to) || !port.CanTransitionAgentTask(from, to) {
		return apperrors.Invalid("agent_task.transition", "task state transition is invalid")
	}
	if len([]rune(reason)) > 4_000 {
		return apperrors.Invalid("agent_task.transition", "transition reason is too long")
	}
	if r == nil {
		return apperrors.Unavailable("agent_task.transition.database", nil)
	}
	now := time.Now().UTC()
	session, err := repoSession(r.engine, ctx, "agent_task.transition")
	if err != nil {
		return err
	}
	defer session.Close()
	var result sql.Result
	if to == port.AgentTaskWaitingQuestion || to == port.AgentTaskWaitingApproval || to.Terminal() {
		finishedAt := any(nil)
		if to.Terminal() {
			finishedAt = now
		}
		result, err = session.Exec(`
UPDATE t_agent_task_run
SET status = ?, last_error = ?, lease_owner = NULL, lease_until = NULL,
    finished_at = COALESCE(?, finished_at), updated_at = ?
WHERE id = ? AND status = ?`,
			string(to), strings.TrimSpace(reason), finishedAt, now, id, string(from))
	} else {
		result, err = session.Exec(`
UPDATE t_agent_task_run
SET status = ?, last_error = ?, updated_at = ?
WHERE id = ? AND status = ?`,
			string(to), strings.TrimSpace(reason), now, id, string(from))
	}
	if err != nil {
		return apperrors.Unavailable("agent_task.transition", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("agent_task.transition.rows", err)
	}
	if affected != 1 {
		return apperrors.Conflict("agent_task.transition", "task state changed concurrently")
	}
	return nil
}

func (r *MyAgentTaskRepository) SavePlanRevision(ctx context.Context, revision port.AgentPlanRevision, steps []port.AgentPlanStep) error {
	revision, err := normalizeAgentPlanRevision(revision)
	if err != nil {
		return apperrors.Invalid("agent_task.plan", err.Error())
	}
	normalizedSteps := make([]normalizedAgentPlanStep, 0, len(steps))
	for _, step := range steps {
		normalized, stepErr := normalizeAgentPlanStep(step, revision.RunID, revision.Revision)
		if stepErr != nil {
			return apperrors.Invalid("agent_task.plan_step", stepErr.Error())
		}
		normalizedSteps = append(normalizedSteps, normalized)
	}
	if r == nil {
		return apperrors.Unavailable("agent_task.plan.database", nil)
	}
	return repoTx(r.engine, ctx, "agent_task.plan", func(session *xorm.Session) error {
		if _, err := session.Exec(`
INSERT INTO t_agent_task_plan_revision (run_id, revision, plan, reason, created_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (run_id, revision) DO UPDATE SET plan = EXCLUDED.plan, reason = EXCLUDED.reason`,
			revision.RunID, revision.Revision, string(revision.Plan), revision.Reason, revision.CreatedAt); err != nil {
			return err
		}
		for _, step := range normalizedSteps {
			if _, err := session.Exec(`
INSERT INTO t_agent_task_step
    (run_id, step_id, revision, ordinal, action, input, depends_on, completion, risk,
     status, attempt, output, last_error, started_at, finished_at, wake_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (run_id, step_id, revision) DO UPDATE SET
    ordinal = EXCLUDED.ordinal, action = EXCLUDED.action, input = EXCLUDED.input,
    depends_on = EXCLUDED.depends_on, completion = EXCLUDED.completion, risk = EXCLUDED.risk,
    status = EXCLUDED.status, attempt = EXCLUDED.attempt, output = EXCLUDED.output,
    last_error = EXCLUDED.last_error, started_at = EXCLUDED.started_at,
    finished_at = EXCLUDED.finished_at, wake_at = EXCLUDED.wake_at`,
				step.RunID, step.StepID, step.Revision, step.Ordinal, step.Action, string(step.Input),
				step.DependsOn, string(step.Completion), step.Risk, string(step.Status), step.Attempt,
				string(step.Output), step.LastError, nullableTime(step.StartedAt), nullableTime(step.FinishedAt), nullableTime(step.WakeAt)); err != nil {
				return err
			}
		}
		return nil
	})
}

type agentPlanStepRow struct {
	RunID      string     `xorm:"run_id"`
	StepID     string     `xorm:"step_id"`
	Revision   int        `xorm:"revision"`
	Ordinal    int        `xorm:"ordinal"`
	Action     string     `xorm:"action"`
	Input      string     `xorm:"input"`
	DependsOn  string     `xorm:"depends_on"`
	Completion string     `xorm:"completion"`
	Risk       string     `xorm:"risk"`
	Status     string     `xorm:"status"`
	Attempt    int        `xorm:"attempt"`
	Output     string     `xorm:"output"`
	LastError  string     `xorm:"last_error"`
	StartedAt  *time.Time `xorm:"started_at"`
	FinishedAt *time.Time `xorm:"finished_at"`
	WakeAt     *time.Time `xorm:"wake_at"`
}

func (r *MyAgentTaskRepository) ListSteps(ctx context.Context, runID string, revision int) ([]port.AgentPlanStep, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" || len(runID) > 128 || revision <= 0 {
		return nil, apperrors.Invalid("agent_task.steps", "run id and positive revision are required")
	}
	if r == nil {
		return nil, apperrors.Unavailable("agent_task.steps.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_task.steps")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var rows []agentPlanStepRow
	if err := session.SQL(`SELECT run_id, step_id, revision, ordinal, action,
COALESCE(input, '') AS input, COALESCE(depends_on, '[]') AS depends_on,
COALESCE(completion, '') AS completion, COALESCE(risk, '') AS risk, status, attempt,
COALESCE(output, '') AS output, COALESCE(last_error, '') AS last_error,
started_at, finished_at, wake_at
FROM t_agent_task_step WHERE run_id = ? AND revision = ? ORDER BY ordinal ASC, step_id ASC`, runID, revision).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("agent_task.steps", err)
	}
	steps := make([]port.AgentPlanStep, 0, len(rows))
	for _, row := range rows {
		var dependsOn []string
		if err := json.Unmarshal([]byte(row.DependsOn), &dependsOn); err != nil {
			return nil, apperrors.Unavailable("agent_task.steps.depends_on", err)
		}
		steps = append(steps, port.AgentPlanStep{
			RunID: row.RunID, StepID: row.StepID, Revision: row.Revision, Ordinal: row.Ordinal,
			Action: row.Action, Input: []byte(row.Input), DependsOn: dependsOn, Completion: []byte(row.Completion),
			Risk: row.Risk, Status: port.AgentStepStatus(row.Status), Attempt: row.Attempt, Output: []byte(row.Output), LastError: row.LastError,
			StartedAt: timePointerValue(row.StartedAt), FinishedAt: timePointerValue(row.FinishedAt), WakeAt: timePointerValue(row.WakeAt),
		})
	}
	return steps, nil
}

func (r *MyAgentTaskRepository) SaveStep(ctx context.Context, input port.AgentPlanStep) error {
	step, err := normalizeAgentPlanStep(input, input.RunID, input.Revision)
	if err != nil {
		return apperrors.Invalid("agent_task.step", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("agent_task.step.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_task.step")
	if err != nil {
		return err
	}
	defer session.Close()
	_, err = session.Exec(`
INSERT INTO t_agent_task_step
    (run_id, step_id, revision, ordinal, action, input, depends_on, completion, risk,
     status, attempt, output, last_error, started_at, finished_at, wake_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (run_id, step_id, revision) DO UPDATE SET
    ordinal = EXCLUDED.ordinal, action = EXCLUDED.action, input = EXCLUDED.input,
    depends_on = EXCLUDED.depends_on, completion = EXCLUDED.completion, risk = EXCLUDED.risk,
    status = EXCLUDED.status, attempt = EXCLUDED.attempt, output = EXCLUDED.output,
    last_error = EXCLUDED.last_error, started_at = EXCLUDED.started_at,
    finished_at = EXCLUDED.finished_at, wake_at = EXCLUDED.wake_at`,
		step.RunID, step.StepID, step.Revision, step.Ordinal, step.Action, string(step.Input), step.DependsOn,
		string(step.Completion), step.Risk, string(step.Status), step.Attempt, string(step.Output), step.LastError,
		nullableTime(step.StartedAt), nullableTime(step.FinishedAt), nullableTime(step.WakeAt))
	if err != nil {
		return apperrors.Unavailable("agent_task.step", err)
	}
	return nil
}

type agentQuestionRow struct {
	ID         string     `xorm:"id"`
	RunID      string     `xorm:"run_id"`
	StepID     string     `xorm:"step_id"`
	Prompt     string     `xorm:"prompt"`
	Options    string     `xorm:"options"`
	Status     string     `xorm:"status"`
	Answer     string     `xorm:"answer"`
	CreatedAt  time.Time  `xorm:"created_at"`
	AnsweredAt *time.Time `xorm:"answered_at"`
}

func (r *MyAgentTaskRepository) CreateQuestion(ctx context.Context, input port.AgentQuestion) error {
	question, options, err := normalizeAgentQuestion(input)
	if err != nil {
		return apperrors.Invalid("agent_task.question.create", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("agent_task.question.create.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_task.question.create")
	if err != nil {
		return err
	}
	defer session.Close()
	_, err = session.Exec(`
INSERT INTO t_agent_task_question (id, run_id, step_id, prompt, options, status, answer, created_at, answered_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO NOTHING`,
		question.ID, question.RunID, question.StepID, question.Prompt, options, string(question.Status), string(question.Answer), question.CreatedAt, nullableTime(question.AnsweredAt))
	if err != nil {
		return apperrors.Unavailable("agent_task.question.create", err)
	}
	return nil
}

func (r *MyAgentTaskRepository) ListPendingQuestions(ctx context.Context, runID string) ([]port.AgentQuestion, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" || len(runID) > 128 {
		return nil, apperrors.Invalid("agent_task.question.list", "run id is invalid")
	}
	if r == nil {
		return nil, apperrors.Unavailable("agent_task.question.list.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_task.question.list")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var rows []agentQuestionRow
	if err := session.SQL(`SELECT id, run_id, step_id, prompt, COALESCE(options, '[]') AS options,
status, COALESCE(answer, '') AS answer, created_at, answered_at
FROM t_agent_task_question WHERE run_id = ? AND status = ? ORDER BY created_at ASC, id ASC`, runID, string(port.AgentQuestionPending)).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("agent_task.question.list", err)
	}
	questions := make([]port.AgentQuestion, 0, len(rows))
	for _, row := range rows {
		var options []string
		if err := json.Unmarshal([]byte(row.Options), &options); err != nil {
			return nil, apperrors.Unavailable("agent_task.question.options", err)
		}
		questions = append(questions, port.AgentQuestion{ID: row.ID, RunID: row.RunID, StepID: row.StepID, Prompt: row.Prompt, Options: options, Status: port.AgentQuestionStatus(row.Status), Answer: []byte(row.Answer), CreatedAt: row.CreatedAt.UTC(), AnsweredAt: timePointerValue(row.AnsweredAt)})
	}
	return questions, nil
}

func (r *MyAgentTaskRepository) AnswerQuestion(ctx context.Context, id, runID string, answer []byte, answeredAt time.Time) error {
	id = strings.TrimSpace(id)
	runID = strings.TrimSpace(runID)
	if id == "" || runID == "" || len(id) > 128 || len(runID) > 128 {
		return apperrors.Invalid("agent_task.question.answer", "question id and run id are required")
	}
	if len(answer) > 100_000 {
		return apperrors.Invalid("agent_task.question.answer", "answer is too large")
	}
	if answeredAt.IsZero() {
		answeredAt = time.Now().UTC()
	}
	if r == nil {
		return apperrors.Unavailable("agent_task.question.answer.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_task.question.answer")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_agent_task_question
SET status = ?, answer = ?, answered_at = ?
WHERE id = ? AND run_id = ? AND status = ?`,
		string(port.AgentQuestionAnswered), string(answer), answeredAt.UTC(), id, runID, string(port.AgentQuestionPending))
	if err != nil {
		return apperrors.Unavailable("agent_task.question.answer", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("agent_task.question.answer.rows", err)
	}
	if affected != 1 {
		return apperrors.Conflict("agent_task.question.answer", "question is no longer pending")
	}
	return nil
}

type agentEffectRow struct {
	EffectKey string    `xorm:"effect_key"`
	RunID     string    `xorm:"run_id"`
	StepID    string    `xorm:"step_id"`
	Tool      string    `xorm:"tool"`
	Result    string    `xorm:"result"`
	CreatedAt time.Time `xorm:"created_at"`
}

func (r agentEffectRow) effect() port.AgentEffect {
	return port.AgentEffect{EffectKey: r.EffectKey, RunID: r.RunID, StepID: r.StepID, Tool: r.Tool, Result: []byte(r.Result), CreatedAt: r.CreatedAt.UTC()}
}

func (r *MyAgentTaskRepository) GetEffect(ctx context.Context, effectKey string) (port.AgentEffect, bool, error) {
	effectKey = strings.TrimSpace(effectKey)
	if effectKey == "" || len(effectKey) > 255 {
		return port.AgentEffect{}, false, apperrors.Invalid("agent_task.effect.get", "effect key is invalid")
	}
	if r == nil {
		return port.AgentEffect{}, false, apperrors.Unavailable("agent_task.effect.get.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_task.effect.get")
	if err != nil {
		return port.AgentEffect{}, false, err
	}
	defer session.Close()
	var row agentEffectRow
	found, err := session.SQL(`SELECT effect_key, run_id, step_id, tool, COALESCE(result, '') AS result, created_at
FROM t_agent_task_effect WHERE effect_key = ?`, effectKey).Get(&row)
	if err != nil {
		return port.AgentEffect{}, false, apperrors.Unavailable("agent_task.effect.get", err)
	}
	if !found {
		return port.AgentEffect{}, false, nil
	}
	return row.effect(), true, nil
}

func (r *MyAgentTaskRepository) SaveEffect(ctx context.Context, input port.AgentEffect) error {
	effect, err := normalizeAgentEffect(input)
	if err != nil {
		return apperrors.Invalid("agent_task.effect.save", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("agent_task.effect.save.database", nil)
	}
	return repoTx(r.engine, ctx, "agent_task.effect.save", func(session *xorm.Session) error {
		var existing agentEffectRow
		found, err := session.SQL(`SELECT effect_key, run_id, step_id, tool, COALESCE(result, '') AS result, created_at
FROM t_agent_task_effect WHERE effect_key = ? FOR UPDATE`, effect.EffectKey).Get(&existing)
		if err != nil {
			return err
		}
		if found {
			if existing.RunID != effect.RunID || existing.StepID != effect.StepID || existing.Tool != effect.Tool {
				return apperrors.Conflict("agent_task.effect.save", "effect key belongs to another side effect")
			}
			return nil
		}
		_, err = session.Exec(`
INSERT INTO t_agent_task_effect (effect_key, run_id, step_id, tool, result, created_at)
VALUES (?, ?, ?, ?, ?, ?)`, effect.EffectKey, effect.RunID, effect.StepID, effect.Tool, string(effect.Result), effect.CreatedAt)
		return err
	})
}

func normalizeAgentTaskRun(input port.AgentTaskRun) (port.AgentTaskRun, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.Goal = strings.TrimSpace(input.Goal)
	input.LeaseOwner = strings.TrimSpace(input.LeaseOwner)
	input.LastError = strings.TrimSpace(input.LastError)
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if input.Status == "" {
		input.Status = port.AgentTaskQueued
	}
	if !validAgentTaskStatus(input.Status) {
		return port.AgentTaskRun{}, stderrors.New("task status is invalid")
	}
	if len(input.ID) > 128 || len(input.RequestID) > 128 || len(input.SessionID) > 128 || input.Goal == "" || len([]rune(input.Goal)) > 10_000 || len(input.LeaseOwner) > 128 || len([]rune(input.LastError)) > 4_000 {
		return port.AgentTaskRun{}, stderrors.New("task identity or content is invalid")
	}
	if input.PlanRevision < 0 || input.ReplanCount < 0 || input.QuestionCount < 0 {
		return port.AgentTaskRun{}, stderrors.New("task counters cannot be negative")
	}
	if input.LeaseOwner == "" {
		input.LeaseUntil = time.Time{}
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	input.CreatedAt = input.CreatedAt.UTC()
	input.UpdatedAt = input.UpdatedAt.UTC()
	if input.WakeAt.IsZero() {
		input.WakeAt = input.UpdatedAt
	}
	input.WakeAt = input.WakeAt.UTC()
	if !input.LeaseUntil.IsZero() {
		input.LeaseUntil = input.LeaseUntil.UTC()
	}
	if !input.FinishedAt.IsZero() {
		input.FinishedAt = input.FinishedAt.UTC()
	}
	return input, nil
}

type normalizedAgentPlanStep struct {
	RunID      string
	StepID     string
	Revision   int
	Ordinal    int
	Action     string
	Input      []byte
	DependsOn  string
	Completion []byte
	Risk       string
	Status     port.AgentStepStatus
	Attempt    int
	Output     []byte
	LastError  string
	StartedAt  time.Time
	FinishedAt time.Time
	WakeAt     time.Time
}

func normalizeAgentPlanRevision(input port.AgentPlanRevision) (port.AgentPlanRevision, error) {
	input.RunID = strings.TrimSpace(input.RunID)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.RunID == "" || len(input.RunID) > 128 || input.Revision <= 0 || len(input.Plan) == 0 || len(input.Plan) > 1_000_000 || len([]rune(input.Reason)) > 1_000 {
		return port.AgentPlanRevision{}, stderrors.New("plan revision is invalid")
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	input.CreatedAt = input.CreatedAt.UTC()
	return input, nil
}

func normalizeAgentPlanStep(input port.AgentPlanStep, runID string, revision int) (normalizedAgentPlanStep, error) {
	runID = strings.TrimSpace(runID)
	input.RunID = strings.TrimSpace(input.RunID)
	input.StepID = strings.TrimSpace(input.StepID)
	input.Action = strings.TrimSpace(input.Action)
	input.Risk = strings.TrimSpace(input.Risk)
	input.LastError = strings.TrimSpace(input.LastError)
	if input.RunID != runID || runID == "" || input.StepID == "" || len(input.StepID) > 128 || input.Revision != revision || input.Revision <= 0 || input.Ordinal < 0 || input.Action == "" || len(input.Action) > 128 || len(input.Risk) > 32 || input.Attempt < 0 || len(input.Input) > 1_000_000 || len(input.Completion) > 1_000_000 || len(input.Output) > 1_000_000 || len([]rune(input.LastError)) > 4_000 {
		return normalizedAgentPlanStep{}, stderrors.New("plan step is invalid")
	}
	if input.Status == "" {
		input.Status = port.AgentStepPending
	}
	if !validAgentStepStatus(input.Status) {
		return normalizedAgentPlanStep{}, stderrors.New("plan step status is invalid")
	}
	dependsOn, err := json.Marshal(input.DependsOn)
	if err != nil {
		return normalizedAgentPlanStep{}, err
	}
	return normalizedAgentPlanStep{
		RunID: input.RunID, StepID: input.StepID, Revision: input.Revision, Ordinal: input.Ordinal, Action: input.Action,
		Input: input.Input, DependsOn: string(dependsOn), Completion: input.Completion, Risk: input.Risk, Status: input.Status,
		Attempt: input.Attempt, Output: input.Output, LastError: input.LastError, StartedAt: input.StartedAt.UTC(), FinishedAt: input.FinishedAt.UTC(), WakeAt: input.WakeAt.UTC(),
	}, nil
}

func normalizeAgentQuestion(input port.AgentQuestion) (port.AgentQuestion, string, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.RunID = strings.TrimSpace(input.RunID)
	input.StepID = strings.TrimSpace(input.StepID)
	input.Prompt = strings.TrimSpace(input.Prompt)
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if input.Status == "" {
		input.Status = port.AgentQuestionPending
	}
	if input.Status != port.AgentQuestionPending || input.RunID == "" || input.StepID == "" || input.Prompt == "" || len(input.ID) > 128 || len(input.RunID) > 128 || len(input.StepID) > 128 || len([]rune(input.Prompt)) > 10_000 || len(input.Options) > 16 {
		return port.AgentQuestion{}, "", stderrors.New("question is invalid")
	}
	options := make([]string, 0, len(input.Options))
	seen := make(map[string]struct{}, len(input.Options))
	for _, option := range input.Options {
		option = strings.TrimSpace(option)
		if option == "" || len([]rune(option)) > 256 {
			return port.AgentQuestion{}, "", stderrors.New("question option is invalid")
		}
		if _, exists := seen[option]; exists {
			continue
		}
		seen[option] = struct{}{}
		options = append(options, option)
	}
	encodedOptions, err := json.Marshal(options)
	if err != nil {
		return port.AgentQuestion{}, "", err
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	input.CreatedAt = input.CreatedAt.UTC()
	return input, string(encodedOptions), nil
}

func normalizeAgentEffect(input port.AgentEffect) (port.AgentEffect, error) {
	input.EffectKey = strings.TrimSpace(input.EffectKey)
	input.RunID = strings.TrimSpace(input.RunID)
	input.StepID = strings.TrimSpace(input.StepID)
	input.Tool = strings.TrimSpace(input.Tool)
	if input.EffectKey == "" || input.RunID == "" || input.StepID == "" || input.Tool == "" || len(input.EffectKey) > 255 || len(input.RunID) > 128 || len(input.StepID) > 128 || len(input.Tool) > 128 || len(input.Result) > 1_000_000 {
		return port.AgentEffect{}, stderrors.New("effect is invalid")
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	input.CreatedAt = input.CreatedAt.UTC()
	return input, nil
}

func validAgentTaskStatus(status port.AgentTaskStatus) bool {
	switch status {
	case port.AgentTaskQueued, port.AgentTaskPlanning, port.AgentTaskRunning, port.AgentTaskWaitingQuestion, port.AgentTaskWaitingApproval, port.AgentTaskCompleted, port.AgentTaskFailed, port.AgentTaskCancelled:
		return true
	default:
		return false
	}
}

func validAgentStepStatus(status port.AgentStepStatus) bool {
	switch status {
	case port.AgentStepPending, port.AgentStepRunning, port.AgentStepWaiting, port.AgentStepSucceeded, port.AgentStepFailed, port.AgentStepSkipped, port.AgentStepCancelled:
		return true
	default:
		return false
	}
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}

func postgresEpochToZero(value time.Time) time.Time {
	if value.IsZero() || value.Equal(time.Unix(0, 0).UTC()) {
		return time.Time{}
	}
	return value.UTC()
}

func timePointerValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}
