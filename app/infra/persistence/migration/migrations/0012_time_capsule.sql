-- Private, user-authored time capsules. This migration is explicit and is
-- never executed during service startup.
CREATE TABLE IF NOT EXISTS t_time_capsule (
    id VARCHAR(128) PRIMARY KEY,
    owner_user_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
    title VARCHAR(120) NOT NULL,
    content TEXT NOT NULL,
    deliver_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL,
    sealed_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    delivery_attempts INTEGER NOT NULL DEFAULT 0,
    last_delivery_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_time_capsule_status_check CHECK (status IN ('draft', 'sealed', 'due', 'delivered', 'delivery_failed')),
    CONSTRAINT t_time_capsule_attempts_check CHECK (delivery_attempts >= 0),
    CONSTRAINT t_time_capsule_sealed_contract_check CHECK (
        (status = 'draft' AND sealed_at IS NULL AND delivered_at IS NULL)
        OR (status IN ('sealed', 'due', 'delivery_failed') AND sealed_at IS NOT NULL AND delivered_at IS NULL)
        OR (status = 'delivered' AND sealed_at IS NOT NULL AND delivered_at IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_t_time_capsule_owner
    ON t_time_capsule (owner_user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_t_time_capsule_due
    ON t_time_capsule (status, deliver_at, id)
    WHERE status = 'sealed';
