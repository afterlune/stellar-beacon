package middlewares

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/config"
	"benetnasch/app/infrastructure/persistence/repository"
	"benetnasch/app/infrastructure/shared"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

func AccessLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := shared.GetUserDetailsDTO().UserInfoId
		roles := repository.ListRolesByUserInfoId(userId)
		hm := make(map[string]struct{}, len(roles))
		for _, v := range roles {
			hm[v] = struct{}{}
		}
		_, ok := hm["admin"]
		if strings.Contains(c.Request.RequestURI, "/admin") && ok {
			c.Next()
		} else {
			ip := shared.GetIpAddress(c.Request)
			count := shared.IncrExpire(ip, time.Second*60)
			if count > shared.ACCESS_LIMIT {
				c.Abort()
				c.JSON(http.StatusOK, model.ResultFailWithMessage("请求过于频繁"))
				return
			}
			c.Next()
		}
	}
}

func SpiderReject() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shared.IsBot(c.Request) || !strings.Contains(c.Request.Host, config.Verification) ||
			!strings.Contains(c.Request.Referer(), config.Verification) {
			c.Abort()
			c.JSON(http.StatusForbidden, model.ResultFailWithMessage("You may be a robot！"))
			return
		}
		c.Next()
	}
}
