package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"xorm.io/builder"
)

type ResourceService interface {
	ListResources(c *gin.Context) model.ResultVO
	DeleteResource(c *gin.Context) model.ResultVO
	SaveOrUpdateResource(c *gin.Context) model.ResultVO
	ListResourceOption() model.ResultVO
	listResourceModule(resources []entity.TResource) (res []entity.TResource)
	listResourceChildren(resources []entity.TResource) map[int][]entity.TResource
}

type MyResourceService struct{}

func (r *MyResourceService) ListResources(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
	}
	engine := ormInit.GetEngine()
	var resources []entity.TResource
	if vo.Keywords != "" {
		err = engine.Prepare().Where(builder.Like{"resource_name", vo.Keywords}).Find(&resources)
	} else {
		err = engine.Prepare().Find(&resources)
	}
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	parents := r.listResourceModule(resources)
	childrenMap := r.listResourceChildren(resources)
	var resourceDTOs []model.ResourceDTO
	for _, v := range parents {
		var resourceDTO model.ResourceDTO
		shared.StructCopy(v, &resourceDTO)
		var child []model.ResourceDTO
		shared.StructCopy(childrenMap[v.Id], &child)
		resourceDTO.Children = child
		delete(childrenMap, v.Id)
		resourceDTOs = append(resourceDTOs, resourceDTO)
	}
	if len(childrenMap) != 0 {
		var childrenList []entity.TResource
		for _, v := range childrenMap {
			childrenList = append(childrenList, v...)
		}
		var childrenDTOs []model.ResourceDTO
		for _, v := range childrenList {
			var dto model.ResourceDTO
			shared.StructCopy(v, &dto)
			childrenDTOs = append(childrenDTOs, dto)
		}
		resourceDTOs = append(resourceDTOs, childrenDTOs...)
	}
	return model.ResultOkWithData(resourceDTOs)
}

func (r *MyResourceService) DeleteResource(c *gin.Context) model.ResultVO {
	id, _ := strconv.Atoi(c.Param("resourceId"))
	engine := ormInit.GetEngine()
	count, err := engine.Prepare().Where(fmt.Sprintf("resource_id = %d", id)).Count(&entity.TRoleResource{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if count > 0 {
		return model.ResultFailWithMessage("该资源下存在角色")
	}
	var iDs []int
	err = engine.Prepare().Select("id").Where(fmt.Sprintf("parent_id = %d", id)).Find(&iDs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	iDs = append(iDs, id)
	_, err = engine.Prepare().In("id", iDs).Delete(&entity.TResource{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (r *MyResourceService) SaveOrUpdateResource(c *gin.Context) model.ResultVO {
	var vo model.ResourceVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var resource entity.TResource
	shared.StructCopy(vo, &resource)
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	if resource.Id != 0 {
		_, err = session.Prepare().ID(resource.Id).Update(&resource)
	} else {
		_, err = session.Insert(&resource)
	}
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFail()
	}
	session.Commit()
	return model.ResultOk()
}

func (r *MyResourceService) ListResourceOption() model.ResultVO {
	engine := ormInit.GetEngine()
	var resources []entity.TResource
	err := engine.Select("id, resource_name, parent_id").Where(fmt.Sprintf("is_anonymous = %d", shared.FALSE)).Find(&resources)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	parents := r.listResourceModule(resources)
	childrenMap := r.listResourceChildren(resources)
	var labelOptionDTOs []model.LabelOptionDTO
	for _, v := range parents {
		var dtos []model.LabelOptionDTO
		children := childrenMap[v.Id]
		if len(childrenMap) != 0 {
			for _, va := range children {
				dtos = append(dtos, model.LabelOptionDTO{
					Id:    va.Id,
					Label: va.ResourceName,
				})
			}
		}
		labelOptionDTOs = append(labelOptionDTOs, model.LabelOptionDTO{
			Id:       v.Id,
			Label:    v.ResourceName,
			Children: dtos,
		})
	}
	return model.ResultOkWithData(labelOptionDTOs)
}

func (r *MyResourceService) listResourceModule(resources []entity.TResource) (res []entity.TResource) {
	for _, v := range resources {
		if v.ParentId == 0 {
			res = append(res, v)
		}
	}
	return res
}

func (r *MyResourceService) listResourceChildren(resources []entity.TResource) map[int][]entity.TResource {
	cm := make(map[int][]entity.TResource)
	for _, v := range resources {
		if v.ParentId != 0 {
			cm[v.ParentId] = append(cm[v.ParentId], v)
		}
	}
	return cm
}
