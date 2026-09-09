package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListErrorLogs
// @Summary		 异常日志模块
// @Description  获取异常日志
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/logs/exceptions [GET]
func ListErrorLogs(c *gin.Context) {
	c.JSON(http.StatusOK, errorLogService.ListErrorLogs(c))
}

// DeleteErrorLogs
// @Summary		 异常日志模块
// @Description  删除异常日志
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/logs/exceptions [DELETE]
func DeleteErrorLogs(c *gin.Context) {
	c.JSON(http.StatusOK, errorLogService.DeleteErrorLogs(c))
}
