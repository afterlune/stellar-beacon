package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListJobLogs
// @Summary		 定时任务日志模块
// @Description  获取定时任务的日志列表
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/logs/jobs [GET]
func ListJobLogs(c *gin.Context) {
	c.JSON(http.StatusOK, jobLogService.ListJobLogs(c))
}

// DeleteJobLogs
// @Summary		 定时任务日志模块
// @Description  删除定时任务的日志
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/logs/jobs [DELETE]
func DeleteJobLogs(c *gin.Context) {
	c.JSON(http.StatusOK, jobLogService.DeleteJobLogs(c))
}

// CleanJobLogs
// @Summary		 定时任务日志模块
// @Description  清除定时任务的日志
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/logs/jobs/clean [DELETE]
func CleanJobLogs(c *gin.Context) {
	c.JSON(http.StatusOK, jobLogService.CleanJobLogs())
}

// ListJobLogGroups
// @Summary		 定时任务日志模块
// @Description  获取定时任务日志的所有组名
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/logs/jobs/groups [GET]
func ListJobLogGroups(c *gin.Context) {
	c.JSON(http.StatusOK, jobLogService.CleanJobLogs())
}
