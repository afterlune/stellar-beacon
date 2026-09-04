-- Review-gated dream candidates and their optional image projection.
-- This migration is executed only by the explicit `migrate` command.
CREATE TABLE IF NOT EXISTS t_dream_entry (
    id VARCHAR(64) PRIMARY KEY,
    review_id VARCHAR(64) NOT NULL UNIQUE REFERENCES t_ai_review(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    image_prompt TEXT NOT NULL DEFAULT '',
    source_article_ids TEXT NOT NULL DEFAULT '[]',
    status VARCHAR(32) NOT NULL,
    image_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    image_url TEXT NOT NULL DEFAULT '',
    image_error VARCHAR(255) NOT NULL DEFAULT '',
    image_lease_owner VARCHAR(128),
    image_lease_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_dream_entry_status_check CHECK (status IN ('pending_review', 'approved', 'rejected', 'expired')),
    CONSTRAINT t_dream_entry_image_status_check CHECK (image_status IN ('pending', 'processing', 'ready', 'placeholder', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_t_dream_entry_public
    ON t_dream_entry (status, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_t_dream_entry_images
    ON t_dream_entry (status, image_status, updated_at ASC, id ASC);
