-- Add the admin-next menu entry for the already-protected Agent memory review
-- API. This migration is executed only by the explicit `migrate` command.
-- The API resources themselves are seeded by 0018.
DO $$
DECLARE
    ai_menu_id INTEGER;
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
    SELECT 'Agent 记忆', '/ai-memory', '/ai/Memory.vue', 'el-icon-myguanyuwo', CURRENT_TIMESTAMP, NULL, 4, ai_menu_id, 0
    WHERE NOT EXISTS (SELECT 1 FROM t_menu WHERE path = '/ai-memory');

    INSERT INTO t_role_menu (role_id, menu_id)
    SELECT role.id, menu.id
    FROM t_role role
    JOIN t_menu menu ON menu.path = '/ai-memory'
    WHERE role.role_name = 'admin'
      AND role.is_disable = 0
      AND NOT EXISTS (
          SELECT 1 FROM t_role_menu existing
          WHERE existing.role_id = role.id AND existing.menu_id = menu.id
      );
END $$;
