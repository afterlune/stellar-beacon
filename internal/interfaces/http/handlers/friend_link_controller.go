package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListFriendLinks
// @Summary		 友链模块
// @Description  查看友链列表
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/links [GET]
func ListFriendLinks(c *gin.Context) {
	c.JSON(http.StatusOK, friendLinkService.ListFriendLinks())
}

// ListFriendLinkDTO
// @Summary		 友链模块
// @Description  查看后台友链列表
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/friend-links [GET]
func ListFriendLinkDTO(c *gin.Context) {
	c.JSON(http.StatusOK, friendLinkService.ListFriendLinkDTO(c))
}

// SaveOrUpdateFriendLink
// @Summary		 友链模块
// @Description  保存或修改友链
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/friend-links [POST]
func SaveOrUpdateFriendLink(c *gin.Context) {
	c.JSON(http.StatusOK, friendLinkService.SaveOrUpdateFriendLink(c))
}

// DeleteFriendLink
// @Summary		 友链模块
// @Description  删除友链
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/friend-links [DELETE]
func DeleteFriendLink(c *gin.Context) {
	c.JSON(http.StatusOK, friendLinkService.DeleteFriendLink(c))
}
