-- Public, provider-neutral activity records used only for bounded Agent
-- vitals/emotion aggregation. This migration is explicit and is never run
-- during service startup.
CREATE TABLE IF NOT EXISTS t_agent_activity (
    id VARCHAR(64) PRIMARY KEY,
    kind VARCHAR(64) NOT NULL,
    emotion VARCHAR(32) NOT NULL DEFAULT 'neutral',
    content_type VARCHAR(64) NOT NULL DEFAULT '',
    content_id VARCHAR(128) NOT NULL DEFAULT '',
    life_stage VARCHAR(32) NOT NULL DEFAULT 'growing',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_agent_activity_emotion_check CHECK (emotion IN ('toxic', 'gentle', 'melancholic', 'neutral')),
    CONSTRAINT t_agent_activity_life_stage_check CHECK (life_stage IN ('newborn', 'growing', 'settled', 'forgotten'))
);

CREATE INDEX IF NOT EXISTS idx_t_agent_activity_occurred
    ON t_agent_activity (occurred_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_t_agent_activity_emotion
    ON t_agent_activity (emotion, occurred_at DESC);
