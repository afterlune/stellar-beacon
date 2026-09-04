-- Durable control state for explicit article index backfills.
-- This migration is executed only by the explicit `migrate` command.
CREATE TABLE IF NOT EXISTS t_ai_index_backfill (
    id VARCHAR(128) PRIMARY KEY,
    index_uid VARCHAR(128) NOT NULL,
    index_version VARCHAR(64) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    model VARCHAR(255) NOT NULL,
    model_version VARCHAR(128) NOT NULL,
    dimension INTEGER NOT NULL,
    embedding_batch_size INTEGER NOT NULL,
    page_size INTEGER NOT NULL,
    status VARCHAR(16) NOT NULL,
    cursor BIGINT NOT NULL DEFAULT 0,
    processed_articles BIGINT NOT NULL DEFAULT 0,
    indexed_chunks BIGINT NOT NULL DEFAULT 0,
    pause_requested BOOLEAN NOT NULL DEFAULT FALSE,
    lease_owner VARCHAR(128),
    lease_until TIMESTAMPTZ,
    last_error TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_ai_index_backfill_status_check CHECK (status IN ('pending', 'running', 'paused', 'completed', 'failed')),
    CONSTRAINT t_ai_index_backfill_contract_check CHECK (dimension > 0 AND embedding_batch_size > 0 AND page_size > 0),
    CONSTRAINT t_ai_index_backfill_progress_check CHECK (cursor >= 0 AND processed_articles >= 0 AND indexed_chunks >= 0)
);

CREATE INDEX IF NOT EXISTS idx_t_ai_index_backfill_status
    ON t_ai_index_backfill (status, updated_at);

CREATE INDEX IF NOT EXISTS idx_t_ai_index_backfill_lease
    ON t_ai_index_backfill (status, lease_until);
