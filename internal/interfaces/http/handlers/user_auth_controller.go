package api

import (
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

// SendCode
// @Summary		 用户账号模块
// @Description  发送邮箱验证码
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/verification-code [GET]
func SendCode(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.SendCode(c))
}

// ListUserAreas
// @Summary		 用户账号模块
// @Description  获取用户区域分布
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/users/areas [GET]
func ListUserAreas(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.ListUserAreas(c))
}

// ListUsers
// @Summary		 用户账号模块
// @Description  查询后台用户列表
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/users [GET]
func ListUsers(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.ListUsers(c))
}

// Register
// @Summary		 用户账号模块
// @Description  用户注册
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/register [POST]
func Register(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.Register(c))
}

// UpdatePassword
// @Summary		 用户账号模块
// @Description  修改密码
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/password [PUT]
func UpdatePassword(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.UpdatePassword(c))
}

// UpdateAdminPassword
// @Summary		 用户账号模块
// @Description  修改管理员密码
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/users/password [PUT]
func UpdateAdminPassword(c *gin.Context) {
	c.JSON(http.StatusOK, userAuthService.UpdateAdminPassword(c))
}

// Logout
// @Summary		 用户账号模块
// @Description  用户登出
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/logout [POST]
func Logout(c *gin.Context) {
	value, exists := c.Get("userInfo")
	if !exists || value == nil {
		c.JSON(http.StatusOK, model.ResultVO{
			Code:    401,
			Message: "用户未登录",
			Data:    nil,
		})
		return
	}
	dto, ok := value.(model.UserDetailsDTO)
	if !ok {
		c.JSON(http.StatusOK, model.ResultVO{
			Code:    500,
			Message: "用户信息类型错误",
			Data:    nil,
		})
		return
	}
	c.JSON(http.StatusOK, userAuthService.Logout(c.Request.Context(), dto.Id))
}
