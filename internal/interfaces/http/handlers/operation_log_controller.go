package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListOperationLogs
// @Summary		 操作日志模块
// @Description  查看操作日志
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/logs/operations [GET]
func ListOperationLogs(c *gin.Context) {
	c.JSON(http.StatusOK, operationLogService.ListOperationLogs(c))
}

// DeleteOperationLogs
// @Summary		 操作日志模块
// @Description  删除操作日志
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/logs/operations [DELETE]
func DeleteOperationLogs(c *gin.Context) {
	c.JSON(http.StatusOK, operationLogService.DeleteOperationLogs(c))
}
