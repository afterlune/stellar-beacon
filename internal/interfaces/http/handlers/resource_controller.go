package api

import (
	"net/http"
	"strconv"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ListResources
// @Summary         资源模块
// @Description    查看资源列表
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/permissions [GET]
func ListResources(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	resources, err := resourceService.ListResources(c.Request.Context(), query.Keywords)
	if err != nil {
		writeResourceError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(resourceDTOs(resources)))
}

// DeleteResource
// @Summary         资源模块
// @Description    删除资源
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/permissions/{resourceId} [DELETE]
func DeleteResource(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("resourceId"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := resourceService.DeleteResource(c.Request.Context(), id); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("该资源下存在角色"))
			return
		}
		writeResourceError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// SaveOrUpdateResource
// @Summary         资源模块
// @Description    新增或修改资源
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/permissions [POST]
func SaveOrUpdateResource(c *gin.Context) {
	var request model.ResourceVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	resource := entity.TResource{Id: request.Id, ResourceName: request.ResourceName, Url: request.Url, RequestMethod: request.RequestMethod, ParentId: request.ParentId, IsAnonymous: request.IsAnonymous}
	if err := resourceService.SaveOrUpdateResource(c.Request.Context(), resource); err != nil {
		writeResourceError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// ListResourceOption
// @Summary         资源模块
// @Description    查看角色资源选项
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/roles/resource-options [GET]
func ListResourceOption(c *gin.Context) {
	resources, err := resourceService.ListResourceOptions(c.Request.Context())
	if err != nil {
		writeResourceError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(resourceOptionDTOs(resources)))
}

func writeResourceError(c *gin.Context, err error) {
	c.JSON(http.StatusOK, model.ResultFromError(err))
}

func resourceDTOs(resources []entity.TResource) []model.ResourceDTO {
	parents := make([]entity.TResource, 0, len(resources))
	children := make(map[int][]entity.TResource)
	for _, resource := range resources {
		if resource.ParentId == 0 {
			parents = append(parents, resource)
		} else {
			children[resource.ParentId] = append(children[resource.ParentId], resource)
		}
	}
	result := make([]model.ResourceDTO, 0, len(parents))
	for _, parent := range parents {
		dto := resourceDTO(parent)
		for _, child := range children[parent.Id] {
			dto.Children = append(dto.Children, resourceDTO(child))
		}
		result = append(result, dto)
		delete(children, parent.Id)
	}
	for _, childList := range children {
		for _, child := range childList {
			result = append(result, resourceDTO(child))
		}
	}
	return result
}

func resourceDTO(resource entity.TResource) model.ResourceDTO {
	return model.ResourceDTO{
		Id: resource.Id, ResourceName: resource.ResourceName, Url: resource.Url,
		RequestMethod: resource.RequestMethod, IsAnonymous: resource.IsAnonymous,
		CreateTime: resource.CreateTime,
	}
}

func resourceOptionDTOs(resources []entity.TResource) []model.LabelOptionDTO {
	parents := make([]entity.TResource, 0, len(resources))
	children := make(map[int][]entity.TResource)
	for _, resource := range resources {
		if resource.ParentId == 0 {
			parents = append(parents, resource)
		} else {
			children[resource.ParentId] = append(children[resource.ParentId], resource)
		}
	}
	result := make([]model.LabelOptionDTO, 0, len(parents))
	for _, parent := range parents {
		option := model.LabelOptionDTO{Id: parent.Id, Label: parent.ResourceName}
		for _, child := range children[parent.Id] {
			option.Children = append(option.Children, model.LabelOptionDTO{Id: child.Id, Label: child.ResourceName})
		}
		result = append(result, option)
	}
	return result
}
