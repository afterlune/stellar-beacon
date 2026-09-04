-- Durable review records for unsaved AI Studio previews.
-- This migration is executed only by the explicit `migrate` command.
CREATE TABLE IF NOT EXISTS t_ai_review (
    id VARCHAR(64) PRIMARY KEY,
    target_type VARCHAR(64) NOT NULL,
    target_id VARCHAR(128) NOT NULL,
    operation VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    diff TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    run_id VARCHAR(128) NOT NULL,
    reviewer_id VARCHAR(64),
    reject_reason TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_ai_review_status_check CHECK (status IN ('pending', 'approved', 'partially_approved', 'rejected', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_t_ai_review_status
    ON t_ai_review (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_t_ai_review_target
    ON t_ai_review (target_type, target_id, created_at DESC);

CREATE TABLE IF NOT EXISTS t_ai_review_action (
    id VARCHAR(64) PRIMARY KEY,
    review_id VARCHAR(64) NOT NULL REFERENCES t_ai_review(id) ON DELETE CASCADE,
    action VARCHAR(32) NOT NULL,
    actor_id VARCHAR(64) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    run_id VARCHAR(128) NOT NULL DEFAULT '',
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_ai_review_action_type_check CHECK (action IN ('created', 'accepted', 'partially_accepted', 'rejected', 'regenerated'))
);

CREATE INDEX IF NOT EXISTS idx_t_ai_review_action_review
    ON t_ai_review_action (review_id, created_at ASC);
