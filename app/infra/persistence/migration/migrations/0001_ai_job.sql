-- Durable AI work queue for article indexing and later AI workers.
-- This migration is executed only by the explicit `migrate` command.
CREATE TABLE IF NOT EXISTS t_ai_job (
    id VARCHAR(64) PRIMARY KEY,
    kind VARCHAR(64) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    payload TEXT NOT NULL,
    status VARCHAR(16) NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    run_after TIMESTAMPTZ NOT NULL,
    lease_owner VARCHAR(128),
    lease_until TIMESTAMPTZ,
    last_error TEXT,
    result_run_id VARCHAR(128),
    result_payload TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_ai_job_status_check CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'dead')),
    CONSTRAINT t_ai_job_attempts_check CHECK (attempts >= 0 AND max_attempts > 0 AND attempts <= max_attempts)
);

CREATE INDEX IF NOT EXISTS idx_t_ai_job_claim
    ON t_ai_job (status, run_after, created_at);

CREATE INDEX IF NOT EXISTS idx_t_ai_job_lease
    ON t_ai_job (status, lease_until);
