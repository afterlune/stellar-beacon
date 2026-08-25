package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListResources
// @Summary		 资源模块
// @Description  查看资源列表
// @Success		 200	{object} model.ResultVO
// @Router       /admin/resources [GET]
func ListResources(c *gin.Context) {
	c.JSON(http.StatusOK, resourceService.ListResources(c))
}

// DeleteResource
// @Summary		 资源模块
// @Description  删除资源
// @Success		 200	{object} model.ResultVO
// @Router       /admin/resources/:resourceId [DELETE]
func DeleteResource(c *gin.Context) {
	c.JSON(http.StatusOK, resourceService.DeleteResource(c))
}

// SaveOrUpdateResource
// @Summary		 资源模块
// @Description  新增或修改资源
// @Success		 200	{object} model.ResultVO
// @Router       /admin/resources [POST]
func SaveOrUpdateResource(c *gin.Context) {
	c.JSON(http.StatusOK, resourceService.SaveOrUpdateResource(c))
}

// ListResourceOption
// @Summary		 资源模块
// @Description  查看角色资源选项
// @Success		 200	{object} model.ResultVO
// @Router       /admin/role/resources [GET]
func ListResourceOption(c *gin.Context) {
	c.JSON(http.StatusOK, resourceService.ListResourceOption())
}
