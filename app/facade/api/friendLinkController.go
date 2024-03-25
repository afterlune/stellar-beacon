package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListFriendLinks
// @Summary		 友链模块
// @Description  查看友链列表
// @Success		 200	{object}	model.ResultVO
// @Router       /links [GET]
func ListFriendLinks(c *gin.Context) {
	res := service.ListFriendLinks()
	c.JSON(http.StatusOK, res)
}

// ListFriendLinkDTO
// @Summary		 友链模块
// @Description  查看后台友链列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/links [GET]
func ListFriendLinkDTO(c *gin.Context) {
	res := service.ListFriendLinkDTO(c)
	c.JSON(http.StatusOK, res)
}

// SaveOrUpdateFriendLink
// @Summary		 友链模块
// @Description  保存或修改友链
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/links [POST]
func SaveOrUpdateFriendLink(c *gin.Context) {
	res := service.SaveOrUpdateFriendLink(c)
	c.JSON(http.StatusOK, res)
}

// DeleteFriendLink
// @Summary		 友链模块
// @Description  删除友链
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/links [DELETE]
func DeleteFriendLink(c *gin.Context) {
	res := service.DeleteFriendLink(c)
	c.JSON(http.StatusOK, res)
}
