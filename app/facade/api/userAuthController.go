package api

import (
	"benetnasch/app/facade/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

// SendCode
// @Summary		 用户账号模块
// @Description  发送邮箱验证码
// @Success		 200	{object}	model.ResultVO
// @Router       /users/code [GET]
func SendCode(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.SendCode(c))
}

// ListUserAreas
// @Summary		 用户账号模块
// @Description  获取用户区域分布
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users/area [GET]
func ListUserAreas(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.ListUserAreas(c))
}

// ListUsers
// @Summary		 用户账号模块
// @Description  查询后台用户列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users [GET]
func ListUsers(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.ListUsers(c))
}

// Register
// @Summary		 用户账号模块
// @Description  用户注册
// @Success		 200	{object}	model.ResultVO
// @Router       /users/register [POST]
func Register(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.Register(c))
}

// UpdatePassword
// @Summary		 用户账号模块
// @Description  修改密码
// @Success		 200	{object}	model.ResultVO
// @Router       /users/password [PUT]
func UpdatePassword(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.UpdatePassword(c))
}

// UpdateAdminPassword
// @Summary		 用户账号模块
// @Description  修改管理员密码
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/users/password [PUT]
func UpdateAdminPassword(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.UpdateAdminPassword(c))
}

// Logout
// @Summary		 用户账号模块
// @Description  用户登出
// @Success		 200	{object}	model.ResultVO
// @Router       /users/logout [POST]
func Logout(c *gin.Context) {
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)
	c.JSON(http.StatusOK, userAuthService.Logout(dto.Id))
}

// QQLogin
// @Summary		 用户账号模块
// @Description  qq登录
// @Success		 200	{object}	model.ResultVO
// @Router       /users/oauth/qq [POST]
func QQLogin(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.QQLogin(c))
}
