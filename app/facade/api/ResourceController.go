package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListResources
// @Summary		 资源模块
// @Description  查看资源列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/resources [GET]
func ListResources(c *gin.Context) {
	res := service.ListResources(c)
	c.JSON(http.StatusOK, res)
}

// DeleteResource
// @Summary		 资源模块
// @Description  删除资源
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/resources/:resourceId [DELETE]
func DeleteResource(c *gin.Context) {
	res := service.DeleteResource(c)
	c.JSON(http.StatusOK, res)
}

// SaveOrUpdateResource
// @Summary		 资源模块
// @Description  新增或修改资源
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/resources [POST]
func SaveOrUpdateResource(c *gin.Context) {
	res := service.SaveOrUpdateResource(c)
	c.JSON(http.StatusOK, res)
}

// ListResourceOption
// @Summary		 资源模块
// @Description  查看角色资源选项
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/role/resources [GET]
func ListResourceOption(c *gin.Context) {
	res := service.ListResourceOption()
	c.JSON(http.StatusOK, res)
}
