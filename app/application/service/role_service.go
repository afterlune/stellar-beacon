package service

import (
	"benetnasch/app/application/support"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"container/list"
	"context"
	"strings"
	"unicode/utf8"
)

type RoleService interface {
	ListUserRoles(ctx context.Context) port.ResultVO
	ListRoles(c port.Request) port.ResultVO
	SaveOrUpdateRole(c port.Request) port.ResultVO
	DeleteRoles(c port.Request) port.ResultVO
}

type MyRoleService struct{ repo port.RoleRepository }

func NewRoleService(repo port.RoleRepository) *MyRoleService { return &MyRoleService{repo: repo} }

func (r *MyRoleService) roleRepository() port.RoleRepository {
	if r.repo != nil {
		return r.repo
	}
	return roleRepo
}

func (r *MyRoleService) ListUserRoles(ctx context.Context) port.ResultVO {
	roles, err := r.roleRepository().ListUserRoles(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	var dtos []port.UserRoleDTO
	support.StructCopy(roles, &dtos)
	return port.ResultOkWithData(dtos)
}

func (r *MyRoleService) ListRoles(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	count, err := r.roleRepository().Count(c.Context(), vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	data, err := r.roleRepository().List(c.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: data, Count: int(count)})
}

func (r *MyRoleService) SaveOrUpdateRole(c port.Request) port.ResultVO {
	var vo port.RoleVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	roleName := strings.TrimSpace(vo.RoleName)
	if roleName == "" || utf8.RuneCountInString(roleName) > 20 {
		return port.ResultFailWithMessage("角色名长度必须为1到20个字符")
	}
	existing, err := r.roleRepository().FindByName(c.Context(), roleName)
	if err != nil {
		return port.ResultFromError(err)
	}
	if existing.Id != 0 && existing.Id != vo.Id {
		return port.ResultFailWithMessage("该角色存在")
	}
	role := port.TRole{Id: vo.Id, RoleName: roleName, IsDisable: support.False}
	if err := r.roleRepository().SaveOrUpdate(c.Context(), role, vo.ResourceIds, vo.MenuIds); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return port.ResultFailWithMessage("该角色存在")
		}
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (r *MyRoleService) DeleteRoles(c port.Request) port.ResultVO {
	var ids []int
	if err := c.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := r.roleRepository().Delete(c.Context(), ids); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return port.ResultFailWithMessage("该角色下存在用户")
		}
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}
