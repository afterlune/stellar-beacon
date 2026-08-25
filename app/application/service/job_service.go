package service

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/shared"
	"container/list"
	"context"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
)

type JobService interface {
	SaveJob(c *gin.Context) model.ResultVO
	UpdateJob(c *gin.Context) model.ResultVO
	DeleteJobById(c *gin.Context) model.ResultVO
	GetJobById(c *gin.Context) model.ResultVO
	ListJobs(c *gin.Context) model.ResultVO
	UpdateJobStatus(c *gin.Context) model.ResultVO
	RunJob(c *gin.Context) model.ResultVO
	ListJobGroup() model.ResultVO
	checkCronIsValid(vo model.JobVO)
}

type MyJobService struct{ repo port.JobRepository }

func NewJobService(repo port.JobRepository) *MyJobService { return &MyJobService{repo: repo} }

func (j *MyJobService) jobRepository() port.JobRepository {
	if j.repo != nil {
		return j.repo
	}
	return jobRepo
}

func (j *MyJobService) SaveJob(c *gin.Context) model.ResultVO {
	var vo model.JobVO
	if err := c.ShouldBind(&vo); err != nil {
		slog.Error("bind job failed", "error", err)
		return model.ResultFail()
	}
	j.checkCronIsValid(vo)
	return model.ResultOk()
}

func (j *MyJobService) UpdateJob(c *gin.Context) model.ResultVO {
	var vo model.JobVO
	if err := c.ShouldBind(&vo); err != nil {
		slog.Error("bind job failed", "error", err)
		return model.ResultFail()
	}
	j.checkCronIsValid(vo)
	return model.ResultOk()
}

func (j *MyJobService) DeleteJobById(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		slog.Error("bind job IDs failed", "error", err)
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (j *MyJobService) GetJobById(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	job, err := j.jobRepository().Get(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	var dto model.JobDTO
	shared.StructCopy(job, &dto)
	return model.ResultOkWithData(dto)
}

func (j *MyJobService) ListJobs(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		current = 1
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		size = 10
	}
	var vo model.JobSearchVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFail()
	}
	jobs, count, err := j.jobRepository().List(c.Request.Context(), current, size, port.JobFilter{JobName: vo.JobName, JobGroup: vo.JobGroup, Status: vo.Status})
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.JobDTO
	shared.StructCopy(jobs, &dtos)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: count})
}

func (j *MyJobService) UpdateJobStatus(c *gin.Context) model.ResultVO { return model.ResultOk() }

func (j *MyJobService) RunJob(c *gin.Context) model.ResultVO { return model.ResultOk() }

func (j *MyJobService) ListJobGroup() model.ResultVO {
	groups, err := j.jobRepository().ListGroups(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(groups)
}

func (j *MyJobService) checkCronIsValid(vo model.JobVO) {}
