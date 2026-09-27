package api

import (
	"container/list"
	"net/http"
	"strconv"

	"github.com/eternallyzzz/stellar-beacon/internal/application/service"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// SaveJob
// @Summary         定时任务模块
// @Description    添加定时任务
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/jobs [POST]
func SaveJob(c *gin.Context) {
	var vo model.JobVO
	if err := c.ShouldBind(&vo); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := jobService.SaveJob(c.Request.Context(), jobInput(vo)); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// UpdateJob
// @Summary         定时任务模块
// @Description    修改定时任务
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/jobs [PUT]
func UpdateJob(c *gin.Context) {
	var vo model.JobVO
	if err := c.ShouldBind(&vo); err != nil || vo.Id == 0 {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := jobService.UpdateJob(c.Request.Context(), jobInput(vo)); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// DeleteJobById
// @Summary         定时任务模块
// @Description    删除定时任务
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/jobs [DELETE]
func DeleteJobById(c *gin.Context) {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := jobService.DeleteJobs(c.Request.Context(), ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// GetJobById
// @Summary         定时任务模块
// @Description    根据 id 获取任务
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/jobs/{id} [GET]
func GetJobById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	job, err := jobService.GetJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(jobResponseDTO(job)))
}

// ListJobs
// @Summary         定时任务模块
// @Description    获取任务列表
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/jobs [GET]
func ListJobs(c *gin.Context) {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		current = 1
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		size = 10
	}
	var query model.JobSearchVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFail())
		return
	}
	jobs, count, err := jobService.ListJobs(c.Request.Context(), current, size, port.JobFilter{
		JobName: query.JobName, JobGroup: query.JobGroup, Status: query.Status,
	})
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	dtos := make([]model.JobDTO, 0, len(jobs))
	for _, job := range jobs {
		dtos = append(dtos, jobResponseDTO(job))
	}
	var records any = dtos
	if count == 0 {
		records = list.New()
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count}))
}

// UpdateJobStatus
// @Summary         定时任务模块
// @Description    更改任务的状态
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/jobs/status [PUT]
func UpdateJobStatus(c *gin.Context) {
	var vo model.JobStatusVO
	if err := c.ShouldBind(&vo); err != nil || vo.Id <= 0 || (vo.Status != 0 && vo.Status != 1) {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := jobService.UpdateJobStatus(c.Request.Context(), vo.Id, vo.Status); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// RunJob
// @Summary         定时任务模块
// @Description    执行某个任务
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/jobs/run [PUT]
func RunJob(c *gin.Context) {
	var request model.JobRunVO
	if err := c.ShouldBind(&request); err != nil || request.Id <= 0 {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	result, err := jobService.RunJob(c.Request.Context(), request.Id)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.JobRunOutcomeDTO{
		JobId: result.JobID, Target: result.Target, Processed: result.Processed, Message: result.Message,
	}))
}

// ListJobGroup
// @Summary         定时任务模块
// @Description    获取所有 job 分组
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/jobs/groups [GET]
func ListJobGroup(c *gin.Context) {
	groups, err := jobService.ListJobGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(groups))
}

// ListJobTargets
// @Summary      定时任务模块
// @Description  获取可执行的内置任务目标
// @Success      200 {object} model.ResultVO
// @Router       /v1/admin/jobs/targets [GET]
func ListJobTargets(c *gin.Context) {
	targets, err := jobService.ListJobTargets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	dtos, err := mapResponseDTO[[]model.JobTargetDTO](targets)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	if dtos == nil {
		dtos = []model.JobTargetDTO{}
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(dtos))
}

func jobInput(vo model.JobVO) service.JobInput {
	return service.JobInput{
		ID: vo.Id, JobName: vo.JobName, JobGroup: vo.JobGroup, InvokeTarget: vo.InvokeTarget,
		CronExpression: vo.CronExpression, Concurrent: vo.Concurrent, Status: vo.Status, Remark: vo.Remark,
	}
}

func jobResponseDTO(detail service.JobDetail) model.JobDTO {
	job := detail.Job
	return model.JobDTO{
		Id: job.Id, JobName: job.JobName, JobGroup: job.JobGroup, InvokeTarget: job.InvokeTarget,
		CronExpression: job.CronExpression, MisfirePolicy: strconv.Itoa(job.MisfirePolicy),
		Concurrent: job.Concurrent, Status: job.Status, CreateTime: job.CreateTime, Remark: job.Remark,
		NextValidTime: detail.NextValidTime, CanRunOnce: detail.CanRunOnce, RunOnceReason: detail.RunOnceReason,
	}
}
