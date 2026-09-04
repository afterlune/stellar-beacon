package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListFriendLinks
// @Summary		 友链模块
// @Description  查看友链列表
// @Success		 200	{object} model.ResultVO
// @Router       /links [GET]
func ListFriendLinks(c *gin.Context) {
	c.JSON(http.StatusOK, friendLinkService.ListFriendLinks(c.Request.Context()))
}

// ListFriendLinkDTO
// @Summary		 友链模块
// @Description  查看后台友链列表
// @Success		 200	{object} model.ResultVO
// @Router       /admin/links [GET]
func ListFriendLinkDTO(c *gin.Context) {
	c.JSON(http.StatusOK, friendLinkService.ListFriendLinkDTO(applicationRequest(c)))
}

// SaveOrUpdateFriendLink
// @Summary		 友链模块
// @Description  保存或修改友链
// @Success		 200	{object} model.ResultVO
// @Router       /admin/links [POST]
func SaveOrUpdateFriendLink(c *gin.Context) {
	c.JSON(http.StatusOK, friendLinkService.SaveOrUpdateFriendLink(applicationRequest(c)))
}

// DeleteFriendLink
// @Summary		 友链模块
// @Description  删除友链
// @Success		 200	{object} model.ResultVO
// @Router       /admin/links [DELETE]
func DeleteFriendLink(c *gin.Context) {
	c.JSON(http.StatusOK, friendLinkService.DeleteFriendLink(applicationRequest(c)))
}
