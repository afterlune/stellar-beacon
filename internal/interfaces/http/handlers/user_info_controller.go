package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// UpdateUserInfo
// @Summary		 用户信息模块
// @Description  更新用户信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/me [PUT]
func UpdateUserInfo(c *gin.Context) {
	c.JSON(http.StatusOK, userInfoService.UpdateUserInfo(c))
}

// UpdateUserAvatar
// @Summary		 用户信息模块
// @Description  更新用户头像
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/me/avatar [POST]
func UpdateUserAvatar(c *gin.Context) {
	c.JSON(http.StatusOK, userInfoService.UpdateUserAvatar(c))
}

// SaveUserEmail
// @Summary		 用户信息模块
// @Description  绑定用户邮箱
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/me/email [PUT]
func SaveUserEmail(c *gin.Context) {
	c.JSON(http.StatusOK, userInfoService.SaveUserEmail(c))
}

// UpdateUserSubscribe 修改用户的订阅状态
// @Summary		 用户信息模块
// @Description  绑定用户邮箱
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/me/subscription [PUT]
func UpdateUserSubscribe(c *gin.Context) {
	c.JSON(http.StatusOK, userInfoService.UpdateUserSubscribe(c))
}

// UpdateUserRole
// @Summary		 用户信息模块
// @Description  修改用户角色
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/users/roles [PUT]
func UpdateUserRole(c *gin.Context) {
	c.JSON(http.StatusOK, userInfoService.UpdateUserRole(c))
}

// UpdateUserDisable
// @Summary		 用户信息模块
// @Description  修改用户禁用状态
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/users/disable [PUT]
func UpdateUserDisable(c *gin.Context) {
	c.JSON(http.StatusOK, userInfoService.UpdateUserDisable(c))
}

// ListOnlineUsers
// @Summary		 用户信息模块
// @Description  查看在线用户
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/users/online [GET]
func ListOnlineUsers(c *gin.Context) {
	c.JSON(http.StatusOK, userInfoService.ListOnlineUsers(c))
}

// RemoveOnlineUser
// @Summary		 用户信息模块
// @Description  下线用户
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/users/{userInfoId}/online [DELETE]
func RemoveOnlineUser(c *gin.Context) {
	c.JSON(http.StatusOK, userInfoService.RemoveOnlineUser(c))
}

// GetUserInfoById
// @Summary		 用户信息模块
// @Description  根据id获取用户信息
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/users/{userInfoId} [GET]
func GetUserInfoById(c *gin.Context) {
	c.JSON(http.StatusOK, userInfoService.GetUserInfoById(c))
}
