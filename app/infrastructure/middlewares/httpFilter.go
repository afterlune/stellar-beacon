package middlewares

import (
	"benetnasch/app/application/service"
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/repository"
	"benetnasch/app/infrastructure/shared"
	"benetnasch/app/infrastructure/zlog"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"net/http"
	"strings"
	"time"
)

func LoginFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request
		uri := req.RequestURI
		if uri == "/users/login" && req.Method == http.MethodPost {
			Users(c)
		}
		c.Next()
	}
}

func AuthorizationFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.Request.Header.Get(shared.TOKEN_HEADER)
		if strings.Contains(authorization, shared.TOKEN_PREFIX) {
			strs := strings.Split(authorization, shared.TOKEN_PREFIX)
			if len(strs) == 2 {
				token := strs[1]
				if token == "null" {
					c.Next()
				} else {
					hm := shared.TokenParse(token)
					if hm == nil {
						c.Abort()
						c.JSON(http.StatusUnauthorized, model.ResultFailWithMessage("非法操作"))
						return
					}
					userAuthId := hm["sub"].(string)
					var userDetailsDTO model.UserDetailsDTO
					dto := shared.HGet(shared.LOGIN_USER, userAuthId)
					if dto == "" {
						c.Abort()
						c.JSON(http.StatusOK, model.ResultFailWithCode(41000))
						return
					}
					err := json.Unmarshal([]byte(dto), &userDetailsDTO)
					if err != nil {
						zlog.Error(err.Error())
						c.Abort()
						c.JSON(http.StatusInternalServerError, model.ResultFailWithMessage("server error"))
						return
					}
					shared.SetUserDetails(userDetailsDTO)
					c.Next()
				}
			}
		}
		if !strings.Contains(authorization, shared.TOKEN_PREFIX) {
			c.Abort()
			c.JSON(http.StatusOK, model.ResultFailWithMessage("非法操作"))
			return
		}
	}
}

func AdminResourceFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.Contains(c.Request.RequestURI, "/admin") {
			userId := shared.GetUserDetailsDTO().UserInfoId
			roles := repository.ListRolesByUserInfoId(userId)
			hm := make(map[string]struct{}, len(roles))
			for _, v := range roles {
				hm[v] = struct{}{}
			}
			_, ok := hm["admin"]
			if !ok {
				c.Abort()
				c.JSON(http.StatusOK, model.ResultFailWithMessage("权限不足"))
				return
			}
		}
		c.Next()
	}
}

func Users(c *gin.Context) {
	var userVO model.UserVO
	err := c.ShouldBind(&userVO)
	if err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusBadGateway, model.ResultFailWithMessage("登录失败，请联系管理员"))
		return
	}
	if !shared.CheckEmail(userVO.Username) {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("邮箱格式不正确！"))
		return
	}
	userDetailsDTO := service.CheckUserAuth(userVO)
	if userDetailsDTO == nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("账号或密码不正确！"))
		return
	}
	ipAddress := shared.GetIpAddress(c.Request)
	region := shared.GetIpSource(ipAddress)
	userAuth := entity.TUserAuth{
		Id:            userDetailsDTO.Id,
		IpAddress:     ipAddress,
		IpSource:      region,
		LastLoginTime: time.Now(),
	}
	service.UpdateUserIp(userAuth)

	regionlist := strings.Split(region, "|")
	ipSource := regionlist[2] + "|" + regionlist[3]
	userDetailsDTO.IpAddress = ipAddress
	userDetailsDTO.IpSource = ipSource
	name, version := shared.GetBrowser(c.Request)
	os := shared.GetOS(c.Request)
	userDetailsDTO.Browser = name + version
	userDetailsDTO.Os = os

	token := shared.CreateToken(userDetailsDTO)
	var userInfoDTO model.UserInfoDTO
	marshal, err := json.Marshal(userDetailsDTO)
	if err != nil {
		zlog.Error(err.Error())
	}

	err = json.Unmarshal(marshal, &userInfoDTO)
	if err != nil {
		zlog.Error(err.Error())
	}

	userInfoDTO.Token = token
	c.JSON(http.StatusOK, model.ResultOkWithData(userInfoDTO))
}
