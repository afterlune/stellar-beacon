package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListExceptionLogs
// @Summary		 异常日志模块
// @Description  获取异常日志
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/exception/logs [GET]
func ListExceptionLogs(c *gin.Context) {
	res := service.ListExceptionLogs(c)
	c.JSON(http.StatusOK, res)
}

// DeleteExceptionLogs
// @Summary		 异常日志模块
// @Description  删除异常日志
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/exception/logs [DELETE]
func DeleteExceptionLogs(c *gin.Context) {
	res := service.DeleteExceptionLogs(c)
	c.JSON(http.StatusOK, res)
}
