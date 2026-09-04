-- Content lifecycle and embedding/PCA projection metadata. A deleted row is
-- retained only as a tombstone; its vector and PCA input are cleared by the
-- repository delete event. This migration is explicit and never runs at boot.
CREATE TABLE IF NOT EXISTS t_content_projection (
    article_id BIGINT PRIMARY KEY,
    life_stage VARCHAR(32) NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    embedding_model VARCHAR(255) NOT NULL DEFAULT '',
    embedding_version VARCHAR(128) NOT NULL DEFAULT '',
    embedding_dimension INTEGER NOT NULL DEFAULT 0,
    pca_input TEXT NOT NULL DEFAULT '[]',
    pca_input_version VARCHAR(64) NOT NULL DEFAULT '',
    projection_x DOUBLE PRECISION,
    projection_y DOUBLE PRECISION,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_content_projection_life_stage_check CHECK (life_stage IN ('newborn', 'growing', 'settled', 'forgotten')),
    CONSTRAINT t_content_projection_status_check CHECK (status IN ('pending', 'ready', 'deleted', 'failed')),
    CONSTRAINT t_content_projection_dimension_check CHECK (embedding_dimension >= 0),
    CONSTRAINT t_content_projection_delete_contract_check CHECK (
        (is_deleted = TRUE AND embedding_model = '' AND embedding_version = '' AND embedding_dimension = 0 AND pca_input = '[]' AND projection_x IS NULL AND projection_y IS NULL AND status = 'deleted')
        OR is_deleted = FALSE
    )
);

CREATE INDEX IF NOT EXISTS idx_t_content_projection_stage
    ON t_content_projection (life_stage, status, updated_at DESC)
    WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_t_content_projection_embedding
    ON t_content_projection (embedding_model, embedding_version, embedding_dimension)
    WHERE is_deleted = FALSE;
