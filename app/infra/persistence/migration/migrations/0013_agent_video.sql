-- Public video metadata. The migration is explicit and is never executed by
-- normal server startup.
CREATE TABLE IF NOT EXISTS t_agent_video (
    id VARCHAR(128) PRIMARY KEY,
    title VARCHAR(160) NOT NULL,
    description VARCHAR(2000) NOT NULL DEFAULT '',
    source VARCHAR(16) NOT NULL,
    url TEXT NOT NULL,
    embed_url TEXT NOT NULL DEFAULT '',
    mime_type VARCHAR(128) NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    published BOOLEAN NOT NULL DEFAULT TRUE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_agent_video_source_check CHECK (source IN ('local', 'external')),
    CONSTRAINT t_agent_video_size_check CHECK (size_bytes >= 0 AND size_bytes <= 524288000),
    CONSTRAINT t_agent_video_external_embed_check CHECK (
        (source = 'local') OR (source = 'external' AND embed_url <> '')
    )
);

CREATE INDEX IF NOT EXISTS idx_t_agent_video_public
    ON t_agent_video (created_at DESC, id DESC)
    WHERE published = TRUE AND deleted = FALSE;
