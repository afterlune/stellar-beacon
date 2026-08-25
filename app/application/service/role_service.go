package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"xorm.io/builder"
	"xorm.io/xorm"
)

type RoleService interface {
	ListUserRoles() model.ResultVO
	ListRoles(c *gin.Context) model.ResultVO
	SaveOrUpdateRole(c *gin.Context) model.ResultVO
	DeleteRoles(c *gin.Context) model.ResultVO
}

type MyRoleService struct{}

func (r *MyRoleService) ListUserRoles() model.ResultVO {
	engine := ormInit.GetEngine()
	var roles []entity.TRole
	err := engine.Select("id, role_name").Find(&roles)
	if err != nil {
		zlog.Error(err.Error())
	}
	var userRoleDTOs []model.UserRoleDTO
	marshal, err := json.Marshal(roles)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	err = json.Unmarshal(marshal, &userRoleDTOs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOkWithData(userRoleDTOs)
}

func (r *MyRoleService) ListRoles(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
	}
	var count int64
	engine := ormInit.GetEngine()
	if vo.Keywords != "" {
		count, err = engine.Where(builder.Like{"role_name", vo.Keywords}).Count(&entity.TRole{})
	} else {
		count, err = engine.Count(&entity.TRole{})
	}
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	data := roleRepo.ListRoles(vo.Current, vo.Size, &vo)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (r *MyRoleService) SaveOrUpdateRole(c *gin.Context) model.ResultVO {
	var vo model.RoleVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	engine := ormInit.GetEngine()
	var roleCheck entity.TRole
	_, err = engine.Select("id").Where("role_name = ?", vo.RoleName).Get(&roleCheck)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if roleCheck.Id != 0 && roleCheck.Id != vo.Id {
		return model.ResultFailWithMessage("该角色存在")
	}
	role := entity.TRole{
		Id:        vo.Id,
		RoleName:  vo.RoleName,
		IsDisable: shared.FALSE,
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		if roleCheck.Id != 0 {
			if _, err = session.ID(roleCheck.Id).Update(&role); err != nil {
				return err
			}
		} else if _, err = session.Insert(&role); err != nil {
			return err
		}
		if len(vo.ResourceIds) != 0 {
			if vo.Id != 0 {
				if _, err = session.Where("role_id = ?", vo.Id).Delete(&entity.TRoleResource{}); err != nil {
					return err
				}
			}
			roleResources := make([]entity.TRoleResource, 0, len(vo.ResourceIds))
			for _, v := range vo.ResourceIds {
				roleResources = append(roleResources, entity.TRoleResource{RoleId: role.Id, ResourceId: v})
			}
			if _, err = session.Insert(&roleResources); err != nil {
				return err
			}
		}
		if len(vo.MenuIds) != 0 {
			if vo.Id != 0 {
				if _, err = session.Where("role_id = ?", vo.Id).Delete(&entity.TRoleMenu{}); err != nil {
					return err
				}
			}
			roleMenus := make([]entity.TRoleMenu, 0, len(vo.MenuIds))
			for _, v := range vo.MenuIds {
				roleMenus = append(roleMenus, entity.TRoleMenu{RoleId: role.Id, MenuId: v})
			}
			if _, err = session.Insert(&roleMenus); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (r *MyRoleService) DeleteRoles(c *gin.Context) model.ResultVO {
	var iDs []int
	err := c.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
	}
	engine := ormInit.GetEngine()
	var count int64
	count, err = engine.In("role_id", iDs).Count(&entity.TUserRole{})
	if err != nil {
		zlog.Error(err.Error())
	}
	if count > 0 {
		return model.ResultFailWithMessage("该角色下存在用户")
	}
	_, err = engine.In("id", iDs).Delete(&entity.TRole{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}
