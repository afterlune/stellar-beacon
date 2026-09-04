-- Immutable memory history and explicit conflict review records. This
-- migration is intentionally additive and is only executed by the explicit
-- migrate command; it never runs during normal server startup.

CREATE TABLE IF NOT EXISTS t_agent_memory_assertion_revision (
    assertion_id VARCHAR(128) NOT NULL,
    revision_no INTEGER NOT NULL,
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
    reason VARCHAR(256) NOT NULL DEFAULT '',
    changed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (assertion_id, revision_no),
    CONSTRAINT t_agent_memory_revision_assertion_fk
        FOREIGN KEY (assertion_id) REFERENCES t_agent_memory_assertion(id),
    CONSTRAINT t_agent_memory_revision_no_check CHECK (revision_no > 0),
    CONSTRAINT t_agent_memory_revision_version_check CHECK (version > 0),
    CONSTRAINT t_agent_memory_revision_confidence_check CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT t_agent_memory_revision_status_check CHECK (status IN ('active', 'stale', 'conflicted', 'retracted')),
    CONSTRAINT t_agent_memory_revision_subject_check CHECK (subject_key LIKE 'user:%')
);

CREATE INDEX IF NOT EXISTS idx_t_agent_memory_revision_lookup
    ON t_agent_memory_assertion_revision (assertion_id, revision_no DESC);

CREATE TABLE IF NOT EXISTS t_agent_memory_conflict (
    id VARCHAR(128) PRIMARY KEY,
    subject_key VARCHAR(128) NOT NULL,
    predicate VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    winner_assertion_id VARCHAR(128),
    resolution VARCHAR(256) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_agent_memory_conflict_status_check CHECK (status IN ('open', 'resolved', 'rejected')),
    CONSTRAINT t_agent_memory_conflict_subject_check CHECK (subject_key LIKE 'user:%'),
    CONSTRAINT t_agent_memory_conflict_winner_fk
        FOREIGN KEY (winner_assertion_id) REFERENCES t_agent_memory_assertion(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_t_agent_memory_open_conflict
    ON t_agent_memory_conflict (subject_key, predicate)
    WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_t_agent_memory_conflict_lookup
    ON t_agent_memory_conflict (subject_key, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS t_agent_memory_conflict_member (
    conflict_id VARCHAR(128) NOT NULL,
    assertion_id VARCHAR(128) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'candidate',
    added_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (conflict_id, assertion_id),
    CONSTRAINT t_agent_memory_conflict_member_conflict_fk
        FOREIGN KEY (conflict_id) REFERENCES t_agent_memory_conflict(id) ON DELETE CASCADE,
    CONSTRAINT t_agent_memory_conflict_member_assertion_fk
        FOREIGN KEY (assertion_id) REFERENCES t_agent_memory_assertion(id),
    CONSTRAINT t_agent_memory_conflict_member_role_check CHECK (role IN ('candidate', 'winner', 'rejected'))
);

CREATE INDEX IF NOT EXISTS idx_t_agent_memory_conflict_member_assertion
    ON t_agent_memory_conflict_member (assertion_id, conflict_id);
