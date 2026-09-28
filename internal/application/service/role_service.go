package service

import (
	"context"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type RoleService interface {
	ListUserRoles(context.Context) ([]entity.TRole, error)
	ListRoles(context.Context, int, int, string) ([]port.RoleView, int64, error)
	SaveOrUpdateRole(context.Context, int, string, []int, []int) error
	DeleteRoles(context.Context, []int) error
}

type MyRoleService struct{ repo port.RoleRepository }

func NewRoleService(repo port.RoleRepository) *MyRoleService { return &MyRoleService{repo: repo} }

func (r *MyRoleService) roleRepository() port.RoleRepository {
	if r.repo != nil {
		return r.repo
	}
	return roleRepo
}

func (r *MyRoleService) ListUserRoles(ctx context.Context) ([]entity.TRole, error) {
	return r.roleRepository().ListUserRoles(ctx)
}

func (r *MyRoleService) ListRoles(ctx context.Context, current, size int, keywords string) ([]port.RoleView, int64, error) {
	count, err := r.roleRepository().Count(ctx, keywords)
	if err != nil || count == 0 {
		return []port.RoleView{}, count, err
	}
	roles, err := r.roleRepository().List(ctx, current, size, keywords)
	return roles, count, err
}

func (r *MyRoleService) SaveOrUpdateRole(ctx context.Context, id int, name string, resourceIDs, menuIDs []int) error {
	existing, err := r.roleRepository().FindByName(ctx, name)
	if err != nil {
		return err
	}
	if existing.Id != 0 && existing.Id != id {
		return apperrors.Conflict("role.save", "该角色存在")
	}
	role := entity.TRole{Id: id, RoleName: name, IsDisable: False}
	if err := r.roleRepository().SaveOrUpdate(ctx, role, resourceIDs, menuIDs); err != nil && apperrors.IsKind(err, apperrors.KindConflict) {
		return apperrors.Conflict("role.save", "该角色存在")
	} else {
		return err
	}
}

func (r *MyRoleService) DeleteRoles(ctx context.Context, ids []int) error {
	return r.roleRepository().Delete(ctx, ids)
}
