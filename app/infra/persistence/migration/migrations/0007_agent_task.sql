-- Durable long-running Agent task state. This migration is explicit and is
-- never executed as part of service startup.
CREATE TABLE IF NOT EXISTS t_agent_task_run (
    id VARCHAR(128) PRIMARY KEY,
    request_id VARCHAR(128) NOT NULL DEFAULT '',
    session_id VARCHAR(128) NOT NULL DEFAULT '',
    goal TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    plan_revision INTEGER NOT NULL DEFAULT 0,
    replan_count INTEGER NOT NULL DEFAULT 0,
    question_count INTEGER NOT NULL DEFAULT 0,
    lease_owner VARCHAR(128),
    lease_until TIMESTAMPTZ,
    wake_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMPTZ,
    CONSTRAINT t_agent_task_run_status_check CHECK (status IN ('queued', 'planning', 'running', 'waiting_question', 'waiting_approval', 'completed', 'failed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_t_agent_task_run_claim
    ON t_agent_task_run (status, wake_at, created_at, id);

CREATE INDEX IF NOT EXISTS idx_t_agent_task_run_lease
    ON t_agent_task_run (lease_until)
    WHERE lease_until IS NOT NULL;

CREATE TABLE IF NOT EXISTS t_agent_task_plan_revision (
    run_id VARCHAR(128) NOT NULL REFERENCES t_agent_task_run(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL,
    plan TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (run_id, revision)
);

CREATE TABLE IF NOT EXISTS t_agent_task_step (
    run_id VARCHAR(128) NOT NULL REFERENCES t_agent_task_run(id) ON DELETE CASCADE,
    step_id VARCHAR(128) NOT NULL,
    revision INTEGER NOT NULL,
    ordinal INTEGER NOT NULL,
    action VARCHAR(128) NOT NULL,
    input TEXT NOT NULL DEFAULT '',
    depends_on TEXT NOT NULL DEFAULT '[]',
    completion TEXT NOT NULL DEFAULT '',
    risk VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 0,
    output TEXT NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    wake_at TIMESTAMPTZ,
    PRIMARY KEY (run_id, step_id, revision),
    CONSTRAINT t_agent_task_step_status_check CHECK (status IN ('pending', 'running', 'waiting', 'succeeded', 'failed', 'skipped', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_t_agent_task_step_run
    ON t_agent_task_step (run_id, revision, ordinal);

CREATE TABLE IF NOT EXISTS t_agent_task_question (
    id VARCHAR(128) PRIMARY KEY,
    run_id VARCHAR(128) NOT NULL REFERENCES t_agent_task_run(id) ON DELETE CASCADE,
    step_id VARCHAR(128) NOT NULL,
    prompt TEXT NOT NULL,
    options TEXT NOT NULL DEFAULT '[]',
    status VARCHAR(32) NOT NULL,
    answer TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    answered_at TIMESTAMPTZ,
    CONSTRAINT t_agent_task_question_status_check CHECK (status IN ('pending', 'answered', 'expired', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_t_agent_task_question_pending
    ON t_agent_task_question (run_id, status, created_at);

CREATE TABLE IF NOT EXISTS t_agent_task_effect (
    effect_key VARCHAR(255) PRIMARY KEY,
    run_id VARCHAR(128) NOT NULL REFERENCES t_agent_task_run(id) ON DELETE CASCADE,
    step_id VARCHAR(128) NOT NULL,
    tool VARCHAR(128) NOT NULL,
    result TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
