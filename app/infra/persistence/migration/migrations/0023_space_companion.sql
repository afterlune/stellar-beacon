-- Digital-space resident identity and append-only public publications.
-- This migration is explicit and is never executed during normal startup.
-- It is intentionally independent from human users, passwords and private
-- time capsules. The service-token boundary may only use the seeded moonfei
-- principal for the first Companion integration.
CREATE TABLE IF NOT EXISTS t_agent_principal (
    id VARCHAR(128) PRIMARY KEY,
    principal_type VARCHAR(32) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_agent_principal_type_check CHECK (principal_type = 'agent')
);

INSERT INTO t_agent_principal (id, principal_type, display_name, scopes)
VALUES ('moonfei', 'agent', '月社妃', '["space:read", "space:publish"]'::jsonb)
ON CONFLICT (id) DO UPDATE
SET principal_type = EXCLUDED.principal_type,
    display_name = EXCLUDED.display_name,
    scopes = EXCLUDED.scopes,
    updated_at = CURRENT_TIMESTAMP;

CREATE TABLE IF NOT EXISTS t_space_publication (
    id VARCHAR(160) PRIMARY KEY,
    agent_id VARCHAR(128) NOT NULL REFERENCES t_agent_principal(id),
    content_type VARCHAR(16) NOT NULL,
    title VARCHAR(160) NOT NULL,
    body TEXT NOT NULL,
    media_url TEXT NOT NULL DEFAULT '',
    source_session_id VARCHAR(160) NOT NULL,
    source_run_id VARCHAR(160) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    request_digest CHAR(64) NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_space_publication_type_check CHECK (content_type IN ('status', 'dream', 'radio'))
);

CREATE INDEX IF NOT EXISTS idx_t_space_publication_public
    ON t_space_publication (content_type, published_at DESC, id DESC);
