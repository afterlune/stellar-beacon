-- Casbin resources for the durable Agent memory review surface.
-- This migration is executed only by the explicit `migrate` command.
-- No menu is seeded yet: the admin UI will be added together with a
-- dedicated memory review page in a later, separately reviewable change.
DO $$
DECLARE
    ai_resource_id INTEGER;
BEGIN
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
        ('读取 Agent 记忆断言', '/admin/ai/memory/assertions', 'GET'),
        ('读取 Agent 记忆历史', '/admin/ai/memory/assertions/*/history', 'GET'),
        ('撤回 Agent 记忆断言', '/admin/ai/memory/assertions/*', 'DELETE'),
        ('读取 Agent 记忆冲突', '/admin/ai/memory/conflicts', 'GET'),
        ('解决 Agent 记忆冲突', '/admin/ai/memory/conflicts/*/resolve', 'POST'),
        ('驳回 Agent 记忆冲突', '/admin/ai/memory/conflicts/*/reject', 'POST')
    ) AS route(resource_name, url, request_method)
    WHERE NOT EXISTS (
        SELECT 1 FROM t_resource existing
        WHERE existing.url = route.url AND existing.request_method = route.request_method
    );

    INSERT INTO t_role_resource (role_id, resource_id)
    SELECT role.id, resource.id
    FROM t_role role
    CROSS JOIN (VALUES
        ('/admin/ai/memory/assertions', 'GET'),
        ('/admin/ai/memory/assertions/*/history', 'GET'),
        ('/admin/ai/memory/assertions/*', 'DELETE'),
        ('/admin/ai/memory/conflicts', 'GET'),
        ('/admin/ai/memory/conflicts/*/resolve', 'POST'),
        ('/admin/ai/memory/conflicts/*/reject', 'POST')
    ) AS route(url, request_method)
    JOIN t_resource resource
      ON resource.parent_id = ai_resource_id
     AND resource.url = route.url
     AND resource.request_method = route.request_method
    WHERE role.role_name = 'admin'
      AND role.is_disable = 0
      AND NOT EXISTS (
          SELECT 1 FROM t_role_resource existing
          WHERE existing.role_id = role.id AND existing.resource_id = resource.id
      );
END $$;
