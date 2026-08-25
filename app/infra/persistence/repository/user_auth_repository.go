package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
)

type UserAuthRepo interface {
	ListUsers(current, size int, vo *model.ConditionVO) []*model.UserAdminDTO
	CountUser(vo *model.ConditionVO) (count int64)
}

type MyUserAuthRepo struct{}

func userFilters(vo *model.ConditionVO, alias string) (string, []interface{}) {
	query := ""
	args := make([]interface{}, 0, 4)
	if vo.LonginType != 0 {
		query += " WHERE " + alias + ".id IN (SELECT user_info_id FROM t_user_auth WHERE login_type = ?)"
		args = append(args, vo.LonginType)
	}
	if vo.Keywords != "" {
		if query == "" {
			query = " WHERE "
		} else {
			query += " AND "
		}
		query += alias + ".nickname LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(vo.Keywords))
	}
	return query, args
}

func (u *MyUserAuthRepo) ListUsers(current, size int, vo *model.ConditionVO) []*model.UserAdminDTO {
	limit, offset := pgsql.Page(current, size)
	filters, args := userFilters(vo, "ui")
	query := "SELECT ua.id, ua.user_info_id, avatar, nickname, login_type, ip_address, ip_source, ua.create_time, last_login_time, ui.is_disable FROM (SELECT id, avatar, nickname, is_disable FROM t_user_info ui" + filters + " LIMIT ? OFFSET ?) ui LEFT JOIN t_user_auth ua ON ua.user_info_id = ui.id"
	args = append(args, limit, offset)
	var users []*model.UserAdminDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&users); err != nil {
		zlog.Error("list users: " + err.Error())
	}

	for _, user := range users {
		roleFilters, roleArgs := userFilters(vo, "ui")
		roleQuery := "SELECT r.id, role_name FROM (SELECT id, avatar, nickname, is_disable FROM t_user_info ui" + roleFilters + " LIMIT ? OFFSET ?) ui LEFT JOIN t_user_auth ua ON ua.user_info_id = ui.id LEFT JOIN t_user_role ur ON ui.id = ur.user_id LEFT JOIN t_role r ON ur.role_id = r.id WHERE ua.id = ?"
		roleArgs = append(roleArgs, limit, offset, user.Id)
		var roles []model.UserRoleDTO
		if err := ormInit.GetEngine().SQL(roleQuery, roleArgs...).Find(&roles); err != nil {
			zlog.Error("load user roles: " + err.Error())
		}
		user.Roles = roles
	}
	return users
}

func (u *MyUserAuthRepo) CountUser(vo *model.ConditionVO) (count int64) {
	filters, args := userFilters(vo, "ui")
	query := "SELECT count(1) FROM t_user_auth ua LEFT JOIN t_user_info ui ON ua.user_info_id = ui.id" + filters
	if _, err := ormInit.GetEngine().SQL(query, args...).Get(&count); err != nil {
		zlog.Error("count users: " + err.Error())
	}
	return count
}
