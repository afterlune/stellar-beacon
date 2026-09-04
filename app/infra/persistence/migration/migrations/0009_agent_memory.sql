-- Versioned long-term memory assertions. Anonymous chat sessions are not
-- valid subjects; the application/repository boundary enforces user:* keys.
-- This migration is explicit and is never executed by service startup.
CREATE TABLE IF NOT EXISTS t_agent_memory_assertion (
    id VARCHAR(128) PRIMARY KEY,
    subject_key VARCHAR(128) NOT NULL,
    predicate VARCHAR(128) NOT NULL,
    object TEXT NOT NULL,
    source_type VARCHAR(64) NOT NULL,
    source_id VARCHAR(128) NOT NULL,
    version INTEGER NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    status VARCHAR(32) NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_agent_memory_confidence_check CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT t_agent_memory_version_check CHECK (version > 0),
    CONSTRAINT t_agent_memory_status_check CHECK (status IN ('active', 'stale', 'conflicted', 'retracted')),
    CONSTRAINT t_agent_memory_subject_check CHECK (subject_key LIKE 'user:%')
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_t_agent_memory_assertion_identity
    ON t_agent_memory_assertion (subject_key, predicate, source_type, source_id, version);

CREATE INDEX IF NOT EXISTS idx_t_agent_memory_lookup
    ON t_agent_memory_assertion (subject_key, status, predicate, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_t_agent_memory_expiry
    ON t_agent_memory_assertion (valid_until)
    WHERE valid_until IS NOT NULL AND status = 'active';
