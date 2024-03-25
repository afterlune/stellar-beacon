package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// SendCode
// @Summary		 用户账号模块
// @Description  发送邮箱验证码
// @Success		 200	{object}	model.ResultVO
// @Router       /users/code [GET]
func SendCode(c *gin.Context) {
	res := service.SendCode(c)
	c.JSON(http.StatusOK, res)
}

// ListUserAreas
// @Summary		 用户账号模块
// @Description  获取用户区域分布
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users/area [GET]
func ListUserAreas(c *gin.Context) {
	res := service.ListUserAreas(c)
	c.JSON(http.StatusOK, res)
}

// ListUsers
// @Summary		 用户账号模块
// @Description  查询后台用户列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users [GET]
func ListUsers(c *gin.Context) {
	res := service.ListUsers(c)
	c.JSON(http.StatusOK, res)
}

// Register
// @Summary		 用户账号模块
// @Description  用户注册
// @Success		 200	{object}	model.ResultVO
// @Router       /users/register [POST]
func Register(c *gin.Context) {
	res := service.Register(c)
	c.JSON(http.StatusOK, res)
}

// UpdatePassword
// @Summary		 用户账号模块
// @Description  修改密码
// @Success		 200	{object}	model.ResultVO
// @Router       /users/password [PUT]
func UpdatePassword(c *gin.Context) {
	res := service.UpdatePassword(c)
	c.JSON(http.StatusOK, res)
}

// UpdateAdminPassword
// @Summary		 用户账号模块
// @Description  修改管理员密码
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users/password [PUT]
func UpdateAdminPassword(c *gin.Context) {
	res := service.UpdateAdminPassword(c)
	c.JSON(http.StatusOK, res)
}

// Logout
// @Summary		 用户账号模块
// @Description  用户登出
// @Success		 200	{object}	model.ResultVO
// @Router       /users/logout [POST]
func Logout(c *gin.Context) {
	res := service.Logout()
	c.JSON(http.StatusOK, res)
}

// QQLogin
// @Summary		 用户账号模块
// @Description  qq登录
// @Success		 200	{object}	model.ResultVO
// @Router       /users/oauth/qq [POST]
func QQLogin(c *gin.Context) {
	res := service.QQLogin(c)
	c.JSON(http.StatusOK, res)
}
