-- Bind review actions to the generation/session snapshot and serialize
-- publication claims. This migration is explicit and is never executed by
-- service startup.
ALTER TABLE t_ai_review
    ADD COLUMN IF NOT EXISTS session_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS content_digest CHAR(64),
    ADD COLUMN IF NOT EXISTS bound_at TIMESTAMPTZ;

ALTER TABLE t_ai_review_action
    ADD COLUMN IF NOT EXISTS session_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS target_type VARCHAR(64),
    ADD COLUMN IF NOT EXISTS target_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS content_digest CHAR(64),
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_t_ai_review_session
    ON t_ai_review (session_id, created_at DESC)
    WHERE session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_t_ai_review_action_review
    ON t_ai_review_action (review_id, created_at DESC);

ALTER TABLE t_ai_review
    DROP CONSTRAINT IF EXISTS t_ai_review_publish_status_check;

ALTER TABLE t_ai_review
    ADD CONSTRAINT t_ai_review_publish_status_check
    CHECK (publish_status IN ('not_attempted', 'processing', 'succeeded', 'skipped', 'failed'));
