package repository

import (
	"benetnasch/internal/infrastructure/persistence/postgres/orm"
	"benetnasch/internal/infrastructure/persistence/postgres/query"
	"benetnasch/internal/interfaces/http/model"
	"log/slog"
)

type RoleRepo interface {
	ListResourceRoles() []*model.ResourceRoleDTO
	ListRolesByUserInfoId(userInfoId int) []string
	ListRoles(current, size int, vo *model.ConditionVO) []*model.RoleDTO
}

type MyRoleRepo struct{}

func (r *MyRoleRepo) ListResourceRoles() []*model.ResourceRoleDTO {
	engine := ormInit.GetEngine()
	var resourceRoles []*model.ResourceRoleDTO
	if err := engine.SQL(pgsql.ListResourceRoles).Find(&resourceRoles); err != nil {
		slog.Error("list resource roles failed", "error", err)
	}
	for _, resource := range resourceRoles {
		var roles []string
		if err := engine.SQL(pgsql.ResourceRoles, resource.Id).Find(&roles); err != nil {
			slog.Error("load resource roles failed", "error", err)
		}
		resource.RoleList = roles
	}
	return resourceRoles
}

func (r *MyRoleRepo) ListRolesByUserInfoId(userInfoId int) []string {
	var roles []string
	if err := ormInit.GetEngine().SQL(pgsql.ListRolesByUserInfoId, userInfoId).Find(&roles); err != nil {
		slog.Error("list user roles failed", "error", err)
	}
	return roles
}

func roleFilter(vo *model.ConditionVO) (string, []interface{}) {
	if vo.Keywords == "" {
		return "", nil
	}
	return " WHERE role_name LIKE ? ESCAPE '\\'", []interface{}{pgsql.ContainsPattern(vo.Keywords)}
}

func (r *MyRoleRepo) ListRoles(current, size int, vo *model.ConditionVO) []*model.RoleDTO {
	limit, offset := pgsql.Page(current, size)
	filter, filterArgs := roleFilter(vo)
	engine := ormInit.GetEngine()
	query := "SELECT r.id, role_name, r.create_time, r.is_disable FROM (SELECT id, role_name, create_time, is_disable FROM t_role" + filter + " LIMIT ? OFFSET ?) r ORDER BY r.id"
	args := append(filterArgs, limit, offset)
	var roles []*model.RoleDTO
	if err := engine.SQL(query, args...).Find(&roles); err != nil {
		slog.Error("list roles failed", "error", err)
	}

	for _, role := range roles {
		resourceQuery := "SELECT rr.resource_id FROM (SELECT id, role_name, create_time, is_disable FROM t_role" + filter + " LIMIT ? OFFSET ?) r LEFT JOIN t_role_resource rr ON r.id = rr.role_id WHERE r.id = ? AND rr.resource_id IS NOT NULL"
		resourceArgs := append(append([]interface{}{}, filterArgs...), limit, offset, role.Id)
		var resourceIDs []int
		if err := engine.SQL(resourceQuery, resourceArgs...).Find(&resourceIDs); err != nil {
			slog.Error("load role resources failed", "error", err)
		}

		menuQuery := "SELECT rm.menu_id FROM (SELECT id, role_name, create_time, is_disable FROM t_role" + filter + " LIMIT ? OFFSET ?) r LEFT JOIN t_role_menu rm ON r.id = rm.role_id WHERE r.id = ? AND rm.menu_id IS NOT NULL"
		menuArgs := append(append([]interface{}{}, filterArgs...), limit, offset, role.Id)
		var menuIDs []int
		if err := engine.SQL(menuQuery, menuArgs...).Find(&menuIDs); err != nil {
			slog.Error("load role menus failed", "error", err)
		}
		role.MenuIds = menuIDs
		role.ResourceIds = resourceIDs
	}
	return roles
}
