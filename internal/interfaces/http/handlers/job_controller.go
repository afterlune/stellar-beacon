package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

//

// SaveJob
// @Summary		 定时任务模块
// @Description  添加定时任务
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/jobs [POST]
func SaveJob(c *gin.Context) {
	c.JSON(http.StatusOK, jobService.SaveJob(c))
}

// UpdateJob
// @Summary		 定时任务模块
// @Description  修改定时任务
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/jobs [PUT]
func UpdateJob(c *gin.Context) {
	c.JSON(http.StatusOK, jobService.UpdateJob(c))
}

// DeleteJobById
// @Summary		 定时任务模块
// @Description  删除定时任务
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/jobs [DELETE]
func DeleteJobById(c *gin.Context) {
	c.JSON(http.StatusOK, jobService.DeleteJobById(c))
}

// GetJobById
// @Summary		 定时任务模块
// @Description  根据id获取任务
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/jobs/{id} [GET]
func GetJobById(c *gin.Context) {
	c.JSON(http.StatusOK, jobService.GetJobById(c))
}

// ListJobs
// @Summary		 定时任务模块
// @Description  获取任务列表
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/jobs [GET]
func ListJobs(c *gin.Context) {
	c.JSON(http.StatusOK, jobService.ListJobs(c))
}

// UpdateJobStatus
// @Summary		 定时任务模块
// @Description  更改任务的状态
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/jobs/status [PUT]
func UpdateJobStatus(c *gin.Context) {
	c.JSON(http.StatusOK, jobService.UpdateJobStatus(c))
}

// RunJob
// @Summary		 定时任务模块
// @Description  执行某个任务
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/jobs/run [PUT]
func RunJob(c *gin.Context) {
	c.JSON(http.StatusOK, jobService.RunJob(c))
}

// ListJobGroup
// @Summary		 定时任务模块
// @Description 获取所有job分组
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/jobs/groups [GET]
func ListJobGroup(c *gin.Context) {
	c.JSON(http.StatusOK, jobService.ListJobGroup(c))
}

// ListJobTargets
// @Summary      定时任务模块
// @Description  获取可执行的内置任务目标
// @Success      200 {object} model.ResultVO
// @Router       /v1/admin/jobs/targets [GET]
func ListJobTargets(c *gin.Context) {
	c.JSON(http.StatusOK, jobService.ListJobTargets(c))
}
