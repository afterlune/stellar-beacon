package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListJobLogs
// @Summary		 定时任务日志模块
// @Description  获取定时任务的日志列表
// @Success		 200	{object} model.ResultVO
// @Router       /admin/jobLogs [GET]
func ListJobLogs(c *gin.Context) {
	c.JSON(http.StatusOK, jobLogService.ListJobLogs(c))
}

// DeleteJobLogs
// @Summary		 定时任务日志模块
// @Description  删除定时任务的日志
// @Success		 200	{object} model.ResultVO
// @Router       /admin/jobLogs [DELETE]
func DeleteJobLogs(c *gin.Context) {
	c.JSON(http.StatusOK, jobLogService.DeleteJobLogs(c))
}

// CleanJobLogs
// @Summary		 定时任务日志模块
// @Description  清除定时任务的日志
// @Success		 200	{object} model.ResultVO
// @Router       /admin/jobLogs/clean [DELETE]
func CleanJobLogs(c *gin.Context) {
	c.JSON(http.StatusOK, jobLogService.CleanJobLogs())
}

// ListJobLogGroups
// @Summary		 定时任务日志模块
// @Description  获取定时任务日志的所有组名
// @Success		 200	{object} model.ResultVO
// @Router       /admin/jobLogs/jobGroups [GET]
func ListJobLogGroups(c *gin.Context) {
	c.JSON(http.StatusOK, jobLogService.CleanJobLogs())
}