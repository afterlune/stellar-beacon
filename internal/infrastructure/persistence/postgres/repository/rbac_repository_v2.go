package repository

import (
	"benetnasch/internal/domain/entity"
	apperrors "benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/infrastructure/persistence/postgres/query"
	"context"
	"strings"

	"xorm.io/xorm"
)

var _ port.MenuRepository = (*MyMenuRepo)(nil)
var _ port.ResourceRepository = (*MyResourceRepo)(nil)
var _ port.RoleRepository = (*MyRoleRepository)(nil)

type MyMenuRepo struct{ engine *xorm.Engine }

func NewMenuRepo(engine *xorm.Engine) *MyMenuRepo { return &MyMenuRepo{engine: engine} }

func (r *MyMenuRepo) List(ctx context.Context, keywords string) ([]entity.TMenu, error) {
	session, err := repoSession(r.engine, ctx, "menu.list")
	if err != nil {
		return nil, err
	}
	where, args := containsFilter("name", keywords)
	var menus []entity.TMenu
	if err := session.SQL("SELECT * FROM t_menu"+where+" ORDER BY order_num, id", args...).Find(&menus); err != nil {
		return nil, apperrors.Unavailable("menu.list", err)
	}
	return menus, nil
}

func (r *MyMenuRepo) ListOptions(ctx context.Context) ([]entity.TMenu, error) {
	session, err := repoSession(r.engine, ctx, "menu.options")
	if err != nil {
		return nil, err
	}
	var menus []entity.TMenu
	if err := session.Select("id, name, parent_id, order_num, path, component, icon, is_hidden").OrderBy("order_num, id").Find(&menus); err != nil {
		return nil, apperrors.Unavailable("menu.options", err)
	}
	return menus, nil
}

func (r *MyMenuRepo) ListByUserInfoID(ctx context.Context, userInfoID int) ([]entity.TMenu, error) {
	session, err := repoSession(r.engine, ctx, "menu.user")
	if err != nil {
		return nil, err
	}
	var menus []entity.TMenu
	if err := session.SQL(pgsql.ListMenusByUserInfoId, userInfoID).Find(&menus); err != nil {
		return nil, apperrors.Unavailable("menu.user", err)
	}
	return menus, nil
}

func (r *MyMenuRepo) SaveOrUpdate(ctx context.Context, menu entity.TMenu) error {
	return repoTx(r.engine, ctx, "menu.save", func(session *xorm.Session) error {
		if menu.Id == 0 {
			_, err := session.Insert(&menu)
			return err
		}
		_, err := session.ID(menu.Id).Update(&menu)
		return err
	})
}

func (r *MyMenuRepo) UpdateHidden(ctx context.Context, id, hidden int) error {
	return repoTx(r.engine, ctx, "menu.hidden", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("is_hidden").Update(&entity.TMenu{Id: id, IsHidden: hidden})
		return err
	})
}

func (r *MyMenuRepo) Delete(ctx context.Context, id int) error {
	return repoTx(r.engine, ctx, "menu.delete", func(session *xorm.Session) error {
		roleCount, err := session.Where("menu_id = ?", id).Count(&entity.TRoleMenu{})
		if err != nil {
			return err
		}
		if roleCount > 0 {
			return apperrors.Conflict("menu.delete", "menu has role associations")
		}
		var childIDs []int
		if err := session.Table("t_menu").Select("id").Where("parent_id = ?", id).Find(&childIDs); err != nil {
			return err
		}
		childIDs = append(childIDs, id)
		_, err = session.In("id", childIDs).Delete(&entity.TMenu{})
		return err
	})
}

type MyResourceRepo struct{ engine *xorm.Engine }

func NewResourceRepo(engine *xorm.Engine) *MyResourceRepo { return &MyResourceRepo{engine: engine} }

func (r *MyResourceRepo) List(ctx context.Context, keywords string) ([]entity.TResource, error) {
	session, err := repoSession(r.engine, ctx, "resource.list")
	if err != nil {
		return nil, err
	}
	where, args := containsFilter("resource_name", keywords)
	var resources []entity.TResource
	if err := session.SQL("SELECT * FROM t_resource"+where+" ORDER BY id", args...).Find(&resources); err != nil {
		return nil, apperrors.Unavailable("resource.list", err)
	}
	return resources, nil
}

func (r *MyResourceRepo) ListOptions(ctx context.Context) ([]entity.TResource, error) {
	session, err := repoSession(r.engine, ctx, "resource.options")
	if err != nil {
		return nil, err
	}
	var resources []entity.TResource
	if err := session.Select("id, resource_name, parent_id").Where("is_anonymous = ?", 0).OrderBy("id").Find(&resources); err != nil {
		return nil, apperrors.Unavailable("resource.options", err)
	}
	return resources, nil
}

func (r *MyResourceRepo) SaveOrUpdate(ctx context.Context, resource entity.TResource) error {
	return repoTx(r.engine, ctx, "resource.save", func(session *xorm.Session) error {
		if resource.Id == 0 {
			_, err := session.Insert(&resource)
			return err
		}
		_, err := session.ID(resource.Id).Update(&resource)
		return err
	})
}

func (r *MyResourceRepo) Delete(ctx context.Context, id int) error {
	return repoTx(r.engine, ctx, "resource.delete", func(session *xorm.Session) error {
		count, err := session.Where("resource_id = ?", id).Count(&entity.TRoleResource{})
		if err != nil {
			return err
		}
		if count > 0 {
			return apperrors.Conflict("resource.delete", "resource has role associations")
		}
		var childIDs []int
		if err := session.Table("t_resource").Select("id").Where("parent_id = ?", id).Find(&childIDs); err != nil {
			return err
		}
		childIDs = append(childIDs, id)
		_, err = session.In("id", childIDs).Delete(&entity.TResource{})
		return err
	})
}

type MyRoleRepository struct{ engine *xorm.Engine }

func NewRoleRepository(engine *xorm.Engine) *MyRoleRepository {
	return &MyRoleRepository{engine: engine}
}

func (r *MyRoleRepository) ListUserRoles(ctx context.Context) ([]entity.TRole, error) {
	session, err := repoSession(r.engine, ctx, "role.user_options")
	if err != nil {
		return nil, err
	}
	var roles []entity.TRole
	if err := session.Select("id, role_name, is_disable").OrderBy("id").Find(&roles); err != nil {
		return nil, apperrors.Unavailable("role.user_options", err)
	}
	return roles, nil
}

func (r *MyRoleRepository) Count(ctx context.Context, keywords string) (int64, error) {
	session, err := repoSession(r.engine, ctx, "role.count")
	if err != nil {
		return 0, err
	}
	where, args := containsFilter("role_name", keywords)
	var count int64
	if _, err := session.SQL("SELECT count(0) FROM t_role"+where, args...).Get(&count); err != nil {
		return 0, apperrors.Unavailable("role.count", err)
	}
	return count, nil
}

func (r *MyRoleRepository) List(ctx context.Context, current, size int, keywords string) ([]port.RoleView, error) {
	session, err := repoSession(r.engine, ctx, "role.list")
	if err != nil {
		return nil, err
	}
	where, filterArgs := containsFilter("role_name", keywords)
	limit, offset := pgsql.Page(current, size)
	args := append(append([]interface{}{}, filterArgs...), limit, offset)
	var roles []port.RoleView
	if err := session.SQL("SELECT id, role_name, create_time, is_disable FROM t_role"+where+" ORDER BY id LIMIT ? OFFSET ?", args...).Find(&roles); err != nil {
		return nil, apperrors.Unavailable("role.list", err)
	}
	for i := range roles {
		if err := session.SQL("SELECT rr.resource_id FROM t_role_resource rr WHERE rr.role_id = ?", roles[i].Id).Find(&roles[i].ResourceIds); err != nil {
			return nil, apperrors.Unavailable("role.resources", err)
		}
		if err := session.SQL("SELECT rm.menu_id FROM t_role_menu rm WHERE rm.role_id = ?", roles[i].Id).Find(&roles[i].MenuIds); err != nil {
			return nil, apperrors.Unavailable("role.menus", err)
		}
	}
	return roles, nil
}

func (r *MyRoleRepository) FindByName(ctx context.Context, name string) (entity.TRole, error) {
	session, err := repoSession(r.engine, ctx, "role.find_name")
	if err != nil {
		return entity.TRole{}, err
	}
	var role entity.TRole
	found, err := session.Where("role_name = ?", name).Get(&role)
	if err != nil {
		return entity.TRole{}, apperrors.Unavailable("role.find_name", err)
	}
	if !found {
		return entity.TRole{}, nil
	}
	return role, nil
}

func (r *MyRoleRepository) SaveOrUpdate(ctx context.Context, role entity.TRole, resourceIDs, menuIDs []int) error {
	return repoTx(r.engine, ctx, "role.save", func(session *xorm.Session) error {
		if role.Id == 0 {
			if _, err := session.Insert(&role); err != nil {
				return err
			}
		} else if _, err := session.ID(role.Id).Update(&role); err != nil {
			return err
		}
		if _, err := session.Where("role_id = ?", role.Id).Delete(&entity.TRoleResource{}); err != nil {
			return err
		}
		if _, err := session.Where("role_id = ?", role.Id).Delete(&entity.TRoleMenu{}); err != nil {
			return err
		}
		resources := make([]entity.TRoleResource, 0, len(resourceIDs))
		for _, id := range resourceIDs {
			resources = append(resources, entity.TRoleResource{RoleId: role.Id, ResourceId: id})
		}
		if len(resources) > 0 {
			if _, err := session.Insert(&resources); err != nil {
				return err
			}
		}
		menus := make([]entity.TRoleMenu, 0, len(menuIDs))
		for _, id := range menuIDs {
			menus = append(menus, entity.TRoleMenu{RoleId: role.Id, MenuId: id})
		}
		if len(menus) > 0 {
			_, err := session.Insert(&menus)
			return err
		}
		return nil
	})
}

func (r *MyRoleRepository) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "role.delete", func(session *xorm.Session) error {
		count, err := session.In("role_id", ids).Count(&entity.TUserRole{})
		if err != nil {
			return err
		}
		if count > 0 {
			return apperrors.Conflict("role.delete", "role has users")
		}
		_, err = session.In("id", ids).Delete(&entity.TRole{})
		return err
	})
}

func (r *MyRoleRepository) ListResourceRoles(ctx context.Context) ([]port.ResourceRoleView, error) {
	session, err := repoSession(r.engine, ctx, "role.resource_roles")
	if err != nil {
		return nil, err
	}
	var resources []port.ResourceRoleView
	if err := session.SQL(pgsql.ListResourceRoles).Find(&resources); err != nil {
		return nil, apperrors.Unavailable("role.resource_roles", err)
	}
	for i := range resources {
		if err := session.SQL(pgsql.ResourceRoles, resources[i].Id).Find(&resources[i].RoleList); err != nil {
			return nil, apperrors.Unavailable("role.resource_roles", err)
		}
	}
	return resources, nil
}

func (r *MyRoleRepository) ListRolesByUserInfoID(ctx context.Context, userInfoID int) ([]string, error) {
	session, err := repoSession(r.engine, ctx, "role.user_roles")
	if err != nil {
		return nil, err
	}
	var roles []string
	if err := session.SQL(pgsql.ListRolesByUserInfoId, userInfoID).Find(&roles); err != nil {
		return nil, apperrors.Unavailable("role.user_roles", err)
	}
	return roles, nil
}

func containsFilter(column, value string) (string, []interface{}) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return " WHERE " + column + " LIKE ? ESCAPE '\\'", []interface{}{pgsql.ContainsPattern(value)}
}
