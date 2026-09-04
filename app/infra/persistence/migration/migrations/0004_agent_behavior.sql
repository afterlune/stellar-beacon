-- Metadata and idempotency identity for review-only autonomous behavior.
-- This migration is executed only by the explicit `migrate` command.
-- No public content table is changed by the behavior worker.
ALTER TABLE t_ai_review
    ADD COLUMN IF NOT EXISTS agent_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS prompt_version VARCHAR(128),
    ADD COLUMN IF NOT EXISTS source_article_id BIGINT,
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_t_ai_review_idempotency
    ON t_ai_review (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_t_ai_review_source_article
    ON t_ai_review (source_article_id, operation, created_at DESC)
    WHERE source_article_id IS NOT NULL;
