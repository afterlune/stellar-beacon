-- Idempotent menu and Casbin resource seeds for the new Agent control plane.
-- This migration is executed only by the explicit `migrate` command.
-- It does not grant access to any role other than the existing enabled admin role.
DO $$
DECLARE
    ai_menu_id INTEGER;
    ai_resource_id INTEGER;
BEGIN
    INSERT INTO t_menu
        (name, path, component, icon, create_time, update_time, order_num, parent_id, is_hidden)
    SELECT 'AI 管理', '/ai-submenu', 'Layout', 'el-icon-myguanyuwo', CURRENT_TIMESTAMP, NULL, 8, NULL, 0
    WHERE NOT EXISTS (SELECT 1 FROM t_menu WHERE path = '/ai-submenu');

    SELECT id INTO ai_menu_id
    FROM t_menu
    WHERE path = '/ai-submenu'
    ORDER BY id
    LIMIT 1;
    IF ai_menu_id IS NULL THEN
        RAISE EXCEPTION 'unable to resolve AI menu parent';
    END IF;

    INSERT INTO t_menu
        (name, path, component, icon, create_time, update_time, order_num, parent_id, is_hidden)
    SELECT 'AI Studio', '/ai-studio', '/ai/Studio.vue', 'el-icon-myguanyuwo', CURRENT_TIMESTAMP, NULL, 1, ai_menu_id, 0
    WHERE NOT EXISTS (SELECT 1 FROM t_menu WHERE path = '/ai-studio');

    INSERT INTO t_menu
        (name, path, component, icon, create_time, update_time, order_num, parent_id, is_hidden)
    SELECT 'Agent 人设', '/ai-profile', '/ai/Profile.vue', 'el-icon-myguanyuwo', CURRENT_TIMESTAMP, NULL, 2, ai_menu_id, 0
    WHERE NOT EXISTS (SELECT 1 FROM t_menu WHERE path = '/ai-profile');

    INSERT INTO t_resource
        (resource_name, url, request_method, parent_id, is_anonymous, create_time, update_time)
    SELECT 'AI 模块', NULL, NULL, NULL, 0, CURRENT_TIMESTAMP, NULL
    WHERE NOT EXISTS (
        SELECT 1 FROM t_resource
        WHERE resource_name = 'AI 模块' AND url IS NULL AND request_method IS NULL
    );

    SELECT id INTO ai_resource_id
    FROM t_resource
    WHERE resource_name = 'AI 模块' AND url IS NULL AND request_method IS NULL
    ORDER BY id
    LIMIT 1;
    IF ai_resource_id IS NULL THEN
        RAISE EXCEPTION 'unable to resolve AI resource parent';
    END IF;

    INSERT INTO t_resource
        (resource_name, url, request_method, parent_id, is_anonymous, create_time, update_time)
    SELECT route.resource_name, route.url, route.request_method, ai_resource_id, 0, CURRENT_TIMESTAMP, NULL
    FROM (VALUES
        ('AI 写作预览', '/admin/ai/writing/preview', 'POST'),
        ('查看 AI 审核队列', '/admin/ai/reviews', 'GET'),
        ('查看 AI 审核详情', '/admin/ai/reviews/*', 'GET'),
        ('批准 AI 生成物', '/admin/ai/reviews/*/approve', 'POST'),
        ('部分接受 AI 生成物', '/admin/ai/reviews/*/partial', 'POST'),
        ('拒绝 AI 生成物', '/admin/ai/reviews/*/reject', 'POST'),
        ('重新生成 AI 生成物', '/admin/ai/reviews/*/regenerate', 'POST'),
        ('过期 AI 生成物', '/admin/ai/reviews/*/expire', 'POST'),
        ('读取 Agent 人设', '/admin/ai/profile', 'GET'),
        ('修改 Agent 人设', '/admin/ai/profile', 'PATCH'),
        ('修改 Agent 紧急开关', '/admin/agent/emergency', 'PUT')
    ) AS route(resource_name, url, request_method)
    WHERE NOT EXISTS (
        SELECT 1 FROM t_resource existing
        WHERE existing.url = route.url AND existing.request_method = route.request_method
    );

    INSERT INTO t_role_menu (role_id, menu_id)
    SELECT role.id, menu.id
    FROM t_role role
    JOIN t_menu menu ON menu.path IN ('/ai-submenu', '/ai-studio', '/ai-profile')
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
    WHERE role.role_name = 'admin'
      AND role.is_disable = 0
      AND NOT EXISTS (
          SELECT 1 FROM t_role_resource existing
          WHERE existing.role_id = role.id AND existing.resource_id = resource.id
      );
END $$;
