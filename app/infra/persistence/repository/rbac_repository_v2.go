package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/persistence/row"
	"context"
	"strings"

	"xorm.io/xorm"
)

var _ port.MenuRepository = (*MyMenuRepo)(nil)
var _ port.ResourceRepository = (*MyResourceRepo)(nil)
var _ port.RoleRepository = (*MyRoleRepository)(nil)

type MyMenuRepo struct{ engine *xorm.Engine }

func NewMenuRepo(engine *xorm.Engine) *MyMenuRepo { return &MyMenuRepo{engine: engine} }

func (r *MyMenuRepo) List(ctx context.Context, keywords string) ([]port.TMenu, error) {
	session, err := repoSession(r.engine, ctx, "menu.list")
	if err != nil {
		return nil, err
	}
	where, args := containsFilter("name", keywords)
	var menus []row.TMenu
	if err := session.SQL("SELECT * FROM t_menu"+where+" ORDER BY order_num, id", args...).Find(&menus); err != nil {
		return nil, apperrors.Unavailable("menu.list", err)
	}
	return row.FromMenus(menus), nil
}

func (r *MyMenuRepo) ListOptions(ctx context.Context) ([]port.TMenu, error) {
	session, err := repoSession(r.engine, ctx, "menu.options")
	if err != nil {
		return nil, err
	}
	var menus []row.TMenu
	if err := session.Select("id, name, parent_id, order_num, path, component, icon, is_hidden").OrderBy("order_num, id").Find(&menus); err != nil {
		return nil, apperrors.Unavailable("menu.options", err)
	}
	return row.FromMenus(menus), nil
}

func (r *MyMenuRepo) ListByUserInfoID(ctx context.Context, userInfoID int) ([]port.TMenu, error) {
	session, err := repoSession(r.engine, ctx, "menu.user")
	if err != nil {
		return nil, err
	}
	var menus []row.TMenu
	if err := session.SQL(pgsql.ListMenusByUserInfoId, userInfoID).Find(&menus); err != nil {
		return nil, apperrors.Unavailable("menu.user", err)
	}
	return row.FromMenus(menus), nil
}

func (r *MyMenuRepo) SaveOrUpdate(ctx context.Context, menu port.TMenu) error {
	return repoTx(r.engine, ctx, "menu.save", func(session *xorm.Session) error {
		menuRow := row.ToMenu(menu)
		if menu.Id == 0 {
			_, err := session.Insert(&menuRow)
			return err
		}
		_, err := session.ID(menu.Id).Update(&menuRow)
		return err
	})
}

func (r *MyMenuRepo) UpdateHidden(ctx context.Context, id, hidden int) error {
	return repoTx(r.engine, ctx, "menu.hidden", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("is_hidden").Update(&row.TMenu{Id: id, IsHidden: hidden})
		return err
	})
}

func (r *MyMenuRepo) Delete(ctx context.Context, id int) error {
	return repoTx(r.engine, ctx, "menu.delete", func(session *xorm.Session) error {
		roleCount, err := session.Where("menu_id = ?", id).Count(&row.TRoleMenu{})
		if err != nil {
			return err
		}
		if roleCount > 0 {
			return apperrors.Conflict("menu.delete", "menu has role associations")
		}
		var children []row.TMenu
		if err := session.Table("t_menu").Select("id").Where("parent_id = ?", id).Find(&children); err != nil {
			return err
		}
		childIDs := []int{id}
		for _, child := range children {
			childIDs = append(childIDs, child.Id)
		}
		_, err = session.In("id", childIDs).Delete(&row.TMenu{})
		return err
	})
}

type MyResourceRepo struct{ engine *xorm.Engine }

func NewResourceRepo(engine *xorm.Engine) *MyResourceRepo { return &MyResourceRepo{engine: engine} }

func (r *MyResourceRepo) List(ctx context.Context, keywords string) ([]port.TResource, error) {
	session, err := repoSession(r.engine, ctx, "resource.list")
	if err != nil {
		return nil, err
	}
	where, args := containsFilter("resource_name", keywords)
	var resources []row.TResource
	if err := session.SQL("SELECT * FROM t_resource"+where+" ORDER BY id", args...).Find(&resources); err != nil {
		return nil, apperrors.Unavailable("resource.list", err)
	}
	return row.FromResources(resources), nil
}

func (r *MyResourceRepo) ListOptions(ctx context.Context) ([]port.TResource, error) {
	session, err := repoSession(r.engine, ctx, "resource.options")
	if err != nil {
		return nil, err
	}
	var resources []row.TResource
	if err := session.Select("id, resource_name, parent_id").Where("is_anonymous = ?", 0).OrderBy("id").Find(&resources); err != nil {
		return nil, apperrors.Unavailable("resource.options", err)
	}
	return row.FromResources(resources), nil
}

func (r *MyResourceRepo) SaveOrUpdate(ctx context.Context, resource port.TResource) error {
	return repoTx(r.engine, ctx, "resource.save", func(session *xorm.Session) error {
		resourceRow := row.ToResource(resource)
		if resource.Id == 0 {
			_, err := session.Insert(&resourceRow)
			return err
		}
		_, err := session.ID(resource.Id).Update(&resourceRow)
		return err
	})
}

func (r *MyResourceRepo) Delete(ctx context.Context, id int) error {
	return repoTx(r.engine, ctx, "resource.delete", func(session *xorm.Session) error {
		count, err := session.Where("resource_id = ?", id).Count(&row.TRoleResource{})
		if err != nil {
			return err
		}
		if count > 0 {
			return apperrors.Conflict("resource.delete", "resource has role associations")
		}
		var children []row.TResource
		if err := session.Table("t_resource").Select("id").Where("parent_id = ?", id).Find(&children); err != nil {
			return err
		}
		childIDs := []int{id}
		for _, child := range children {
			childIDs = append(childIDs, child.Id)
		}
		_, err = session.In("id", childIDs).Delete(&row.TResource{})
		return err
	})
}

type MyRoleRepository struct{ engine *xorm.Engine }

func NewRoleRepository(engine *xorm.Engine) *MyRoleRepository {
	return &MyRoleRepository{engine: engine}
}

func (r *MyRoleRepository) ListUserRoles(ctx context.Context) ([]port.TRole, error) {
	session, err := repoSession(r.engine, ctx, "role.user_options")
	if err != nil {
		return nil, err
	}
	var roles []row.TRole
	if err := session.Select("id, role_name, is_disable").OrderBy("id").Find(&roles); err != nil {
		return nil, apperrors.Unavailable("role.user_options", err)
	}
	return row.FromRoles(roles), nil
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

func (r *MyRoleRepository) FindByName(ctx context.Context, name string) (port.TRole, error) {
	session, err := repoSession(r.engine, ctx, "role.find_name")
	if err != nil {
		return port.TRole{}, err
	}
	var role row.TRole
	found, err := session.Where("role_name = ?", name).Get(&role)
	if err != nil {
		return port.TRole{}, apperrors.Unavailable("role.find_name", err)
	}
	if !found {
		return port.TRole{}, nil
	}
	return row.FromRole(role), nil
}

func (r *MyRoleRepository) SaveOrUpdate(ctx context.Context, role port.TRole, resourceIDs, menuIDs []int) error {
	return repoTx(r.engine, ctx, "role.save", func(session *xorm.Session) error {
		roleRow := row.ToRole(role)
		if role.Id == 0 {
			if _, err := session.Insert(&roleRow); err != nil {
				return err
			}
			role.Id = roleRow.Id
		} else if _, err := session.ID(role.Id).Update(&roleRow); err != nil {
			return err
		}
		if _, err := session.Where("role_id = ?", role.Id).Delete(&row.TRoleResource{}); err != nil {
			return err
		}
		if _, err := session.Where("role_id = ?", role.Id).Delete(&row.TRoleMenu{}); err != nil {
			return err
		}
		resources := make([]row.TRoleResource, 0, len(resourceIDs))
		for _, id := range resourceIDs {
			resources = append(resources, row.TRoleResource{RoleId: role.Id, ResourceId: id})
		}
		if len(resources) > 0 {
			if _, err := session.Insert(&resources); err != nil {
				return err
			}
		}
		menus := make([]row.TRoleMenu, 0, len(menuIDs))
		for _, id := range menuIDs {
			menus = append(menus, row.TRoleMenu{RoleId: role.Id, MenuId: id})
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
		count, err := session.In("role_id", ids).Count(&row.TUserRole{})
		if err != nil {
			return err
		}
		if count > 0 {
			return apperrors.Conflict("role.delete", "role has users")
		}
		_, err = session.In("id", ids).Delete(&row.TRole{})
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
