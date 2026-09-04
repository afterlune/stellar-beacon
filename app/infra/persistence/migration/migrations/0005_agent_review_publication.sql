-- Publication outcome and lifecycle audit fields for approved agent reviews.
-- This migration is executed only by the explicit `migrate` command.
ALTER TABLE t_ai_review
    ADD COLUMN IF NOT EXISTS publish_status VARCHAR(32) NOT NULL DEFAULT 'not_attempted',
    ADD COLUMN IF NOT EXISTS publish_error VARCHAR(255),
    ADD COLUMN IF NOT EXISTS published_content_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS publication_key VARCHAR(255),
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_t_ai_review_publication_key
    ON t_ai_review (publication_key)
    WHERE publication_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_t_ai_review_publish_status
    ON t_ai_review (publish_status, updated_at DESC);

ALTER TABLE t_ai_review_action
    DROP CONSTRAINT IF EXISTS t_ai_review_action_type_check;

ALTER TABLE t_ai_review_action
    ADD CONSTRAINT t_ai_review_action_type_check
    CHECK (action IN ('created', 'accepted', 'partially_accepted', 'rejected', 'regenerated', 'expired', 'published', 'publish_failed'));
