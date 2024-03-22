package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ListJobLogs
// @Summary		 定时任务日志模块
// @Description  获取定时任务的日志列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobLogs [GET]
func ListJobLogs(c *gin.Context) {
	res := service.ListJobLogs(c)
	c.JSON(http.StatusOK, res)
}

// DeleteJobLogs
// @Summary		 定时任务日志模块
// @Description  删除定时任务的日志
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobLogs [DELETE]
func DeleteJobLogs(c *gin.Context) {
	res := service.DeleteJobLogs(c)
	c.JSON(http.StatusOK, res)
}

// CleanJobLogs
// @Summary		 定时任务日志模块
// @Description  清除定时任务的日志
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobLogs/clean [DELETE]
func CleanJobLogs(c *gin.Context) {
	res := service.CleanJobLogs()
	c.JSON(http.StatusOK, res)
}

// ListJobLogGroups
// @Summary		 定时任务日志模块
// @Description  获取定时任务日志的所有组名
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobLogs/jobGroups [GET]
func ListJobLogGroups(c *gin.Context) {
	res := service.ListJobLogGroups()
	c.JSON(http.StatusOK, res)
}
