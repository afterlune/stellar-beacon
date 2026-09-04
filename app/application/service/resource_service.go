package service

import (
	"benetnasch/app/application/support"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"strconv"
)

type ResourceService interface {
	ListResources(c port.Request) port.ResultVO
	DeleteResource(c port.Request) port.ResultVO
	SaveOrUpdateResource(c port.Request) port.ResultVO
	ListResourceOption(ctx context.Context) port.ResultVO
	listResourceModule(resources []port.TResource) []port.TResource
	listResourceChildren(resources []port.TResource) map[int][]port.TResource
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

func (r *MyResourceService) ListResources(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	resources, err := r.resourceRepository().List(c.Context(), vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(r.resourceDTOs(resources))
}

func (r *MyResourceService) DeleteResource(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("resourceId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := r.resourceRepository().Delete(c.Context(), id); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return port.ResultFailWithMessage("该资源下存在角色")
		}
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (r *MyResourceService) SaveOrUpdateResource(c port.Request) port.ResultVO {
	var vo port.ResourceVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	resource := port.TResource{Id: vo.Id, ResourceName: vo.ResourceName, Url: vo.Url, RequestMethod: vo.RequestMethod, ParentId: vo.ParentId, IsAnonymous: vo.IsAnonymous}
	if err := r.resourceRepository().SaveOrUpdate(c.Context(), resource); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (r *MyResourceService) ListResourceOption(ctx context.Context) port.ResultVO {
	resources, err := r.resourceRepository().ListOptions(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	parents := r.listResourceModule(resources)
	children := r.listResourceChildren(resources)
	options := make([]port.LabelOptionDTO, 0, len(parents))
	for _, parent := range parents {
		items := children[parent.Id]
		childrenDTO := make([]port.LabelOptionDTO, 0, len(items))
		for _, child := range items {
			childrenDTO = append(childrenDTO, port.LabelOptionDTO{Id: child.Id, Label: child.ResourceName})
		}
		options = append(options, port.LabelOptionDTO{Id: parent.Id, Label: parent.ResourceName, Children: childrenDTO})
	}
	return port.ResultOkWithData(options)
}

func (r *MyResourceService) listResourceModule(resources []port.TResource) []port.TResource {
	result := make([]port.TResource, 0)
	for _, resource := range resources {
		if resource.ParentId == 0 {
			result = append(result, resource)
		}
	}
	return result
}

func (r *MyResourceService) listResourceChildren(resources []port.TResource) map[int][]port.TResource {
	result := make(map[int][]port.TResource)
	for _, resource := range resources {
		if resource.ParentId != 0 {
			result[resource.ParentId] = append(result[resource.ParentId], resource)
		}
	}
	return result
}

func (r *MyResourceService) resourceDTOs(resources []port.TResource) []port.ResourceDTO {
	parents := r.listResourceModule(resources)
	children := r.listResourceChildren(resources)
	result := make([]port.ResourceDTO, 0, len(parents))
	for _, parent := range parents {
		var dto port.ResourceDTO
		support.StructCopy(parent, &dto)
		var childDTOs []port.ResourceDTO
		support.StructCopy(children[parent.Id], &childDTOs)
		dto.Children = childDTOs
		result = append(result, dto)
		delete(children, parent.Id)
	}
	for _, childList := range children {
		var dtos []port.ResourceDTO
		support.StructCopy(childList, &dtos)
		result = append(result, dtos...)
	}
	return result
}
