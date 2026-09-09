package service

import (
	"benetnasch/internal/application/support"
	"benetnasch/internal/domain/entity"
	apperrors "benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/interfaces/http/model"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ResourceService interface {
	ListResources(c *gin.Context) model.ResultVO
	DeleteResource(c *gin.Context) model.ResultVO
	SaveOrUpdateResource(c *gin.Context) model.ResultVO
	ListResourceOption() model.ResultVO
	listResourceModule(resources []entity.TResource) []entity.TResource
	listResourceChildren(resources []entity.TResource) map[int][]entity.TResource
}

type MyResourceService struct{ repo port.ResourceRepository }

func NewResourceService(repo port.ResourceRepository) *MyResourceService {
	return &MyResourceService{repo: repo}
}

func (r *MyResourceService) resourceRepository() port.ResourceRepository {
	if r.repo != nil {
		return r.repo
	}
	return resourceRepo
}

func (r *MyResourceService) ListResources(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	resources, err := r.resourceRepository().List(c.Request.Context(), vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(r.resourceDTOs(resources))
}

func (r *MyResourceService) DeleteResource(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("resourceId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := r.resourceRepository().Delete(c.Request.Context(), id); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return model.ResultFailWithMessage("该资源下存在角色")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (r *MyResourceService) SaveOrUpdateResource(c *gin.Context) model.ResultVO {
	var vo model.ResourceVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	resource := entity.TResource{Id: vo.Id, ResourceName: vo.ResourceName, Url: vo.Url, RequestMethod: vo.RequestMethod, ParentId: vo.ParentId, IsAnonymous: vo.IsAnonymous}
	if err := r.resourceRepository().SaveOrUpdate(c.Request.Context(), resource); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (r *MyResourceService) ListResourceOption() model.ResultVO {
	resources, err := r.resourceRepository().ListOptions(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	parents := r.listResourceModule(resources)
	children := r.listResourceChildren(resources)
	options := make([]model.LabelOptionDTO, 0, len(parents))
	for _, parent := range parents {
		items := children[parent.Id]
		childrenDTO := make([]model.LabelOptionDTO, 0, len(items))
		for _, child := range items {
			childrenDTO = append(childrenDTO, model.LabelOptionDTO{Id: child.Id, Label: child.ResourceName})
		}
		options = append(options, model.LabelOptionDTO{Id: parent.Id, Label: parent.ResourceName, Children: childrenDTO})
	}
	return model.ResultOkWithData(options)
}

func (r *MyResourceService) listResourceModule(resources []entity.TResource) []entity.TResource {
	result := make([]entity.TResource, 0)
	for _, resource := range resources {
		if resource.ParentId == 0 {
			result = append(result, resource)
		}
	}
	return result
}

func (r *MyResourceService) listResourceChildren(resources []entity.TResource) map[int][]entity.TResource {
	result := make(map[int][]entity.TResource)
	for _, resource := range resources {
		if resource.ParentId != 0 {
			result[resource.ParentId] = append(result[resource.ParentId], resource)
		}
	}
	return result
}

func (r *MyResourceService) resourceDTOs(resources []entity.TResource) []model.ResourceDTO {
	parents := r.listResourceModule(resources)
	children := r.listResourceChildren(resources)
	result := make([]model.ResourceDTO, 0, len(parents))
	for _, parent := range parents {
		var dto model.ResourceDTO
		support.StructCopy(parent, &dto)
		var childDTOs []model.ResourceDTO
		support.StructCopy(children[parent.Id], &childDTOs)
		dto.Children = childDTOs
		result = append(result, dto)
		delete(children, parent.Id)
	}
	for _, childList := range children {
		var dtos []model.ResourceDTO
		support.StructCopy(childList, &dtos)
		result = append(result, dtos...)
	}
	return result
}
