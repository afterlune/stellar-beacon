package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

//

// SaveJob
// @Summary		 定时任务模块
// @Description  添加定时任务
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobs [POST]
func SaveJob(c *gin.Context) {
	res := service.SaveJob(c)
	c.JSON(http.StatusOK, res)
}

// UpdateJob
// @Summary		 定时任务模块
// @Description  修改定时任务
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobs [PUT]
func UpdateJob(c *gin.Context) {
	res := service.UpdateJob(c)
	c.JSON(http.StatusOK, res)
}

// DeleteJobById
// @Summary		 定时任务模块
// @Description  删除定时任务
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobs [DELETE]
func DeleteJobById(c *gin.Context) {
	res := service.DeleteJobById(c)
	c.JSON(http.StatusOK, res)
}

// GetJobById
// @Summary		 定时任务模块
// @Description  根据id获取任务
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobs/:id [GET]
func GetJobById(c *gin.Context) {
	res := service.GetJobById(c)
	c.JSON(http.StatusOK, res)
}

// ListJobs
// @Summary		 定时任务模块
// @Description  获取任务列表
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobs [GET]
func ListJobs(c *gin.Context) {
	res := service.ListJobs(c)
	c.JSON(http.StatusOK, res)
}

// UpdateJobStatus
// @Summary		 定时任务模块
// @Description  更改任务的状态
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobs/status [PUT]
func UpdateJobStatus(c *gin.Context) {
	res := service.UpdateJobStatus(c)
	c.JSON(http.StatusOK, res)
}

// RunJob
// @Summary		 定时任务模块
// @Description  执行某个任务
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobs/run [PUT]
func RunJob(c *gin.Context) {
	res := service.RunJob(c)
	c.JSON(http.StatusOK, res)
}

// ListJobGroup
// @Summary		 定时任务模块
// @Description  获取所有job分组
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/jobs/jobGroups [GET]
func ListJobGroup(c *gin.Context) {
	res := service.ListJobGroup()
	c.JSON(http.StatusOK, res)
}
