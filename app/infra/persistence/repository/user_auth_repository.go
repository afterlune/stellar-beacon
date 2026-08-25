package repository

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"context"

	"xorm.io/xorm"
)

var _ port.AuthRepository = (*MyUserAuthRepo)(nil)

type MyUserAuthRepo struct {
	engine *xorm.Engine
}

func NewUserAuthRepo(engine *xorm.Engine) *MyUserAuthRepo {
	return &MyUserAuthRepo{engine: engine}
}

func (u *MyUserAuthRepo) authSession(ctx context.Context) (*xorm.Session, error) {
	return repoSession(u.engine, ctx, "auth")
}

func userFilters(filter port.UserFilter, alias string) (string, []interface{}) {
	query := ""
	args := make([]interface{}, 0, 2)
	if filter.LoginType != 0 {
		query += " WHERE " + alias + ".id IN (SELECT user_info_id FROM t_user_auth WHERE login_type = ?)"
		args = append(args, filter.LoginType)
	}
	if filter.Keywords != "" {
		if query == "" {
			query = " WHERE "
		} else {
			query += " AND "
		}
		query += alias + ".nickname LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(filter.Keywords))
	}
	return query, args
}

func (u *MyUserAuthRepo) ListUsers(ctx context.Context, filter port.UserFilter) ([]*port.UserAdmin, int64, error) {
	session, err := u.authSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	filters, args := userFilters(filter, "ui")
	var count int64
	countQuery := "SELECT count(1) FROM t_user_auth ua LEFT JOIN t_user_info ui ON ua.user_info_id = ui.id" + filters
	if _, err := session.SQL(countQuery, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "auth.count_users", err)
	}
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	query := "SELECT ua.id, ua.user_info_id, avatar, nickname, login_type, ip_address, ip_source, ua.create_time, last_login_time, ui.is_disable FROM (SELECT id, avatar, nickname, is_disable FROM t_user_info ui" + filters + " LIMIT ? OFFSET ?) ui LEFT JOIN t_user_auth ua ON ua.user_info_id = ui.id"
	args = append(args, limit, offset)
	var users []*port.UserAdmin
	if err := session.SQL(query, args...).Find(&users); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "auth.list_users", err)
	}
	for _, user := range users {
		roleFilters, roleArgs := userFilters(filter, "ui")
		roleQuery := "SELECT r.id, role_name FROM (SELECT id, avatar, nickname, is_disable FROM t_user_info ui" + roleFilters + " LIMIT ? OFFSET ?) ui LEFT JOIN t_user_auth ua ON ua.user_info_id = ui.id LEFT JOIN t_user_role ur ON ui.id = ur.user_id LEFT JOIN t_role r ON ur.role_id = r.id WHERE ua.id = ?"
		roleArgs = append(roleArgs, limit, offset, user.Id)
		var roles []port.UserRole
		if err := session.SQL(roleQuery, roleArgs...).Find(&roles); err != nil {
			return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "auth.list_user_roles", err)
		}
		user.Roles = roles
	}
	return users, count, nil
}

func (u *MyUserAuthRepo) FindByUsername(ctx context.Context, username string) (port.AuthUser, error) {
	session, err := u.authSession(ctx)
	if err != nil {
		return port.AuthUser{}, err
	}
	var auth entity.TUserAuth
	found, err := session.Where("username = ?", username).Get(&auth)
	if err != nil {
		return port.AuthUser{}, apperrors.Wrap(apperrors.KindUnavailable, "auth.find_username", err)
	}
	if !found {
		return port.AuthUser{}, apperrors.NotFound("auth.find_username")
	}
	var info entity.TUserInfo
	found, err = session.ID(auth.UserInfoId).Get(&info)
	if err != nil {
		return port.AuthUser{}, apperrors.Wrap(apperrors.KindUnavailable, "auth.find_user_info", err)
	}
	if !found {
		return port.AuthUser{}, apperrors.NotFound("auth.find_user_info")
	}
	var roles []string
	if err := session.SQL("SELECT role_name FROM t_role WHERE id IN (SELECT role_id FROM t_user_role WHERE user_id = ?)", info.Id).Find(&roles); err != nil {
		return port.AuthUser{}, apperrors.Wrap(apperrors.KindUnavailable, "auth.find_roles", err)
	}
	return port.AuthUser{Auth: auth, Info: info, Roles: roles}, nil
}

func (u *MyUserAuthRepo) FindByID(ctx context.Context, id int) (entity.TUserAuth, error) {
	session, err := u.authSession(ctx)
	if err != nil {
		return entity.TUserAuth{}, err
	}
	var auth entity.TUserAuth
	found, err := session.ID(id).Get(&auth)
	if err != nil {
		return entity.TUserAuth{}, apperrors.Wrap(apperrors.KindUnavailable, "auth.find_id", err)
	}
	if !found {
		return entity.TUserAuth{}, apperrors.NotFound("auth.find_id")
	}
	return auth, nil
}

func (u *MyUserAuthRepo) CreateUser(ctx context.Context, info entity.TUserInfo, auth entity.TUserAuth, roleID int) error {
	return ormInit.WithEngineTx(u.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.Insert(&info); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "auth.create_info", err)
		}
		if _, err := session.Insert(&entity.TUserRole{UserId: info.Id, RoleId: roleID}); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "auth.create_role", err)
		}
		auth.UserInfoId = info.Id
		if _, err := session.Insert(&auth); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "auth.create_auth", err)
		}
		return nil
	})
}

func (u *MyUserAuthRepo) UpdatePassword(ctx context.Context, username, password string) error {
	return ormInit.WithEngineTx(u.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.Where("username = ?", username).Cols("password").Update(&entity.TUserAuth{Password: password}); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "auth.update_password", err)
		}
		return nil
	})
}

func (u *MyUserAuthRepo) UpdatePasswordByID(ctx context.Context, id int, password string) error {
	return ormInit.WithEngineTx(u.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.ID(id).Cols("password").Update(&entity.TUserAuth{Password: password}); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "auth.update_admin_password", err)
		}
		return nil
	})
}

func (u *MyUserAuthRepo) UpdateLoginMetadata(ctx context.Context, user entity.TUserAuth) error {
	return ormInit.WithEngineTx(u.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.ID(user.Id).MustCols("ip_source", "ip_address", "last_login_time").Update(&user); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "auth.update_login_metadata", err)
		}
		return nil
	})
}
