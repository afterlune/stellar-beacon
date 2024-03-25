package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListOperationLogs
// @Summary		 操作日志模块
// @Description  查看操作日志
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/operation/logs [GET]
func ListOperationLogs(c *gin.Context) {
	res := service.ListOperationLogs(c)
	c.JSON(http.StatusOK, res)
}

// DeleteOperationLogs
// @Summary		 操作日志模块
// @Description  删除操作日志
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/operation/logs [DELETE]
func DeleteOperationLogs(c *gin.Context) {
	res := service.DeleteOperationLogs(c)
	c.JSON(http.StatusOK, res)
}
