package middlewares

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/config"
	"benetnasch/app/infrastructure/persistence/repository"
	"benetnasch/app/infrastructure/shared"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	globalLimiter = rate.NewLimiter(rate.Every(time.Second/20), 1200)
	ipLimiter     = make(map[string]*rate.Limiter)
	mutex         sync.Mutex
)

func AccessLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		value, _ := c.Get("userInfo")
		dto := value.(model.UserDetailsDTO)
		roles := repository.ListRolesByUserInfoId(dto.UserInfoId)
		hm := make(map[string]struct{}, len(roles))
		for _, v := range roles {
			hm[v] = struct{}{}
		}
		_, ok := hm["admin"]
		if strings.Contains(c.Request.RequestURI, "/admin") && ok {
			c.Next()
		} else {
			ip := shared.GetIpAddress(c.Request)
			mutex.Lock()
			limiter, ok := ipLimiter[ip]
			if !ok {
				limiter = rate.NewLimiter(rate.Every(time.Minute/60), 60)
				ipLimiter[ip] = limiter
			}
			mutex.Unlock()

			if !limiter.Allow() || !globalLimiter.Allow() {
				c.Abort()
				c.JSON(http.StatusTooManyRequests, model.ResultFailWithMessage("请求过于频繁"))
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
