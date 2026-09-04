-- Versioned review safety policy and its new-admin control-plane resources.
-- This migration is executed only by the explicit `migrate` command.
CREATE TABLE IF NOT EXISTS t_agent_review_policy (
    id VARCHAR(64) PRIMARY KEY,
    version BIGINT NOT NULL,
    review_required BOOLEAN NOT NULL DEFAULT TRUE,
    review_ttl_seconds BIGINT NOT NULL,
    max_candidate_runes INTEGER NOT NULL,
    similarity_threshold DOUBLE PRECISION NOT NULL,
    daily_limit INTEGER NOT NULL,
    per_article_limit INTEGER NOT NULL,
    per_action_limit INTEGER NOT NULL,
    allowed_actions JSONB NOT NULL DEFAULT '[]'::jsonb,
    sensitive_patterns JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_agent_review_policy_version_check CHECK (version > 0),
    CONSTRAINT t_agent_review_policy_required_check CHECK (review_required = TRUE),
    CONSTRAINT t_agent_review_policy_ttl_check CHECK (review_ttl_seconds > 0),
    CONSTRAINT t_agent_review_policy_runes_check CHECK (max_candidate_runes > 0),
    CONSTRAINT t_agent_review_policy_similarity_check CHECK (similarity_threshold > 0 AND similarity_threshold <= 1),
    CONSTRAINT t_agent_review_policy_daily_check CHECK (daily_limit > 0),
    CONSTRAINT t_agent_review_policy_article_check CHECK (per_article_limit > 0),
    CONSTRAINT t_agent_review_policy_action_check CHECK (per_action_limit > 0),
    CONSTRAINT t_agent_review_policy_actions_object_check CHECK (jsonb_typeof(allowed_actions) = 'array'),
    CONSTRAINT t_agent_review_policy_patterns_object_check CHECK (jsonb_typeof(sensitive_patterns) = 'array')
);

DO $$
DECLARE
    ai_menu_id INTEGER;
    ai_resource_id INTEGER;
BEGIN
    SELECT id INTO ai_menu_id
    FROM t_menu
    WHERE path = '/ai-submenu'
    ORDER BY id
    LIMIT 1;
    IF ai_menu_id IS NULL THEN
        RAISE EXCEPTION 'unable to resolve AI menu parent; apply 0015_agent_admin_rbac first';
    END IF;

    INSERT INTO t_menu
        (name, path, component, icon, create_time, update_time, order_num, parent_id, is_hidden)
    SELECT '审核策略', '/ai-review-policy', '/ai/ReviewPolicy.vue', 'el-icon-myguanyuwo', CURRENT_TIMESTAMP, NULL, 3, ai_menu_id, 0
    WHERE NOT EXISTS (SELECT 1 FROM t_menu WHERE path = '/ai-review-policy');

    SELECT id INTO ai_resource_id
    FROM t_resource
    WHERE resource_name = 'AI 模块' AND url IS NULL AND request_method IS NULL
    ORDER BY id
    LIMIT 1;
    IF ai_resource_id IS NULL THEN
        RAISE EXCEPTION 'unable to resolve AI resource parent; apply 0015_agent_admin_rbac first';
    END IF;

    INSERT INTO t_resource
        (resource_name, url, request_method, parent_id, is_anonymous, create_time, update_time)
    SELECT route.resource_name, route.url, route.request_method, ai_resource_id, 0, CURRENT_TIMESTAMP, NULL
    FROM (VALUES
        ('读取 Agent 审核策略', '/admin/ai/review-policy', 'GET'),
        ('修改 Agent 审核策略', '/admin/ai/review-policy', 'PATCH')
    ) AS route(resource_name, url, request_method)
    WHERE NOT EXISTS (
        SELECT 1 FROM t_resource existing
        WHERE existing.url = route.url AND existing.request_method = route.request_method
    );

    INSERT INTO t_role_menu (role_id, menu_id)
    SELECT role.id, menu.id
    FROM t_role role
    JOIN t_menu menu ON menu.path = '/ai-review-policy'
    WHERE role.role_name = 'admin'
      AND role.is_disable = 0
      AND NOT EXISTS (
          SELECT 1 FROM t_role_menu existing
          WHERE existing.role_id = role.id AND existing.menu_id = menu.id
      );

    INSERT INTO t_role_resource (role_id, resource_id)
    SELECT role.id, resource.id
    FROM t_role role
    JOIN t_resource resource ON resource.parent_id = ai_resource_id
                            AND resource.url = '/admin/ai/review-policy'
    WHERE role.role_name = 'admin'
      AND role.is_disable = 0
      AND NOT EXISTS (
          SELECT 1 FROM t_role_resource existing
          WHERE existing.role_id = role.id AND existing.resource_id = resource.id
      );
END $$;
