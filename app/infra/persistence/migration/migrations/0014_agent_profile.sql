-- Administrator-controlled versioned Agent persona.
-- This migration is executed only by the explicit `migrate` command.
-- Bootstrap uses the repository only when ai.agent.persistence_enabled=true.
CREATE TABLE IF NOT EXISTS t_agent_profile (
    id VARCHAR(128) PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    prompt_version VARCHAR(128) NOT NULL,
    system_prompt_ref VARCHAR(255) NOT NULL DEFAULT '',
    system_prompt TEXT NOT NULL,
    opening TEXT NOT NULL DEFAULT '',
    awake_prompt TEXT NOT NULL DEFAULT '',
    dusk_prompt TEXT NOT NULL DEFAULT '',
    night_prompt TEXT NOT NULL DEFAULT '',
    behavior_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_t_agent_profile_enabled
    ON t_agent_profile (enabled, updated_at DESC);
