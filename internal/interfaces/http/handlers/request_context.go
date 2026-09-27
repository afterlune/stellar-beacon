package api

import (
	"strconv"

	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func authenticatedUserInfoID(c *gin.Context) (int, bool) {
	value, ok := c.Get("userInfo")
	if !ok {
		return 0, false
	}
	user, ok := value.(model.UserDetailsDTO)
	return user.UserInfoId, ok && user.UserInfoId > 0
}

func requestPageParams(c *gin.Context) (int, int, bool) {
	current, err := strconv.Atoi(c.DefaultQuery("current", "1"))
	if err != nil || current < 1 {
		return 0, 0, false
	}
	size, err := strconv.Atoi(c.DefaultQuery("size", "12"))
	if err != nil || size < 1 {
		return 0, 0, false
	}
	if size > 100 {
		size = 100
	}
	return current, size, true
}
