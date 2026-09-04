-- Add the explicit administrator resource for the configured-provider smoke
-- test. The endpoint never accepts or stores credentials from the request.
-- This migration is executed only by the explicit migrate command.
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
    SELECT '测试 AI Provider 连接', '/admin/ai/providers/test', 'POST', ai_resource_id, 0, CURRENT_TIMESTAMP, NULL
    WHERE NOT EXISTS (
        SELECT 1 FROM t_resource
        WHERE url = '/admin/ai/providers/test' AND request_method = 'POST'
    );

    INSERT INTO t_role_resource (role_id, resource_id)
    SELECT role.id, resource.id
    FROM t_role role
    JOIN t_resource resource
      ON resource.parent_id = ai_resource_id
     AND resource.url = '/admin/ai/providers/test'
     AND resource.request_method = 'POST'
    WHERE role.role_name = 'admin'
      AND role.is_disable = 0
      AND NOT EXISTS (
          SELECT 1 FROM t_role_resource existing
          WHERE existing.role_id = role.id AND existing.resource_id = resource.id
      );
END $$;
