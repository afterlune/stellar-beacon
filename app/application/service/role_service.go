package service

import (
	"benetnasch/app/application/support"
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"container/list"
	"context"

	"github.com/gin-gonic/gin"
)

type RoleService interface {
	ListUserRoles() model.ResultVO
	ListRoles(c *gin.Context) model.ResultVO
	SaveOrUpdateRole(c *gin.Context) model.ResultVO
	DeleteRoles(c *gin.Context) model.ResultVO
}

type MyRoleService struct{ repo port.RoleRepository }

func NewRoleService(repo port.RoleRepository) *MyRoleService { return &MyRoleService{repo: repo} }

func (r *MyRoleService) roleRepository() port.RoleRepository {
	if r.repo != nil {
		return r.repo
	}
	return roleRepo
}

func (r *MyRoleService) ListUserRoles() model.ResultVO {
	roles, err := r.roleRepository().ListUserRoles(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.UserRoleDTO
	support.StructCopy(roles, &dtos)
	return model.ResultOkWithData(dtos)
}

func (r *MyRoleService) ListRoles(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	count, err := r.roleRepository().Count(c.Request.Context(), vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	data, err := r.roleRepository().List(c.Request.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (r *MyRoleService) SaveOrUpdateRole(c *gin.Context) model.ResultVO {
	var vo model.RoleVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	existing, err := r.roleRepository().FindByName(c.Request.Context(), vo.RoleName)
	if err != nil {
		return model.ResultFromError(err)
	}
	if existing.Id != 0 && existing.Id != vo.Id {
		return model.ResultFailWithMessage("该角色存在")
	}
	role := entity.TRole{Id: vo.Id, RoleName: vo.RoleName, IsDisable: support.False}
	if err := r.roleRepository().SaveOrUpdate(c.Request.Context(), role, vo.ResourceIds, vo.MenuIds); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return model.ResultFailWithMessage("该角色存在")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (r *MyRoleService) DeleteRoles(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := r.roleRepository().Delete(c.Request.Context(), ids); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return model.ResultFailWithMessage("该角色下存在用户")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}
