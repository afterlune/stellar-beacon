package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// UpdateUserInfo
// @Summary		 用户信息模块
// @Description  更新用户信息
// @Success		 200	{object}	model.ResultVO
// @Router       /users/info [PUT]
func UpdateUserInfo(c *gin.Context) {
	res := service.UpdateUserInfo(c)
	c.JSON(http.StatusOK, res)
}

// UpdateUserAvatar
// @Summary		 用户信息模块
// @Description  更新用户头像
// @Success		 200	{object}	model.ResultVO
// @Router       /users/avatar [POST]
func UpdateUserAvatar(c *gin.Context) {
	res := service.UpdateUserAvatar(c)
	c.JSON(http.StatusOK, res)
}

// SaveUserEmail
// @Summary		 用户信息模块
// @Description  绑定用户邮箱
// @Success		 200	{object}	model.ResultVO
// @Router       /users/email [PUT]
func SaveUserEmail(c *gin.Context) {
	res := service.SaveUserEmail(c)
	c.JSON(http.StatusOK, res)
}

// UpdateUserSubscribe 修改用户的订阅状态
// @Summary		 用户信息模块
// @Description  绑定用户邮箱
// @Success		 200	{object}	model.ResultVO
// @Router       /users/subscribe [PUT]
func UpdateUserSubscribe(c *gin.Context) {
	res := service.UpdateUserSubscribe(c)
	c.JSON(http.StatusOK, res)
}

// UpdateUserRole
// @Summary		 用户信息模块
// @Description  修改用户角色
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users/role [PUT]
func UpdateUserRole(c *gin.Context) {
	res := service.UpdateUserRole(c)
	c.JSON(http.StatusOK, res)
}

// UpdateUserDisable
// @Summary		 用户信息模块
// @Description  修改用户禁用状态
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users/disable [PUT]
func UpdateUserDisable(c *gin.Context) {
	res := service.UpdateUserDisable(c)
	c.JSON(http.StatusOK, res)
}

// ListOnlineUsers
// @Summary		 用户信息模块
// @Description  查看在线用户
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users/online [GET]
func ListOnlineUsers(c *gin.Context) {
	res := service.ListOnlineUsers(c)
	c.JSON(http.StatusOK, res)
}

// RemoveOnlineUser
// @Summary		 用户信息模块
// @Description  下线用户
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users/:userInfoId/online [DELETE]
func RemoveOnlineUser(c *gin.Context) {
	res := service.RemoveOnlineUser(c)
	c.JSON(http.StatusOK, res)
}

// GetUserInfoById
// @Summary		 用户信息模块
// @Description  根据id获取用户信息
// @Success		 200	{object}	model.ResultVO
// @Router       /users/info/:userInfoId [GET]
func GetUserInfoById(c *gin.Context) {
	res := service.GetUserInfoById(c)
	c.JSON(http.StatusOK, res)
}
