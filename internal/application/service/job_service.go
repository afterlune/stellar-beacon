package service

import (
	"container/list"
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
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
	ListJobGroup(c *gin.Context) model.ResultVO
	ListJobTargets(c *gin.Context) model.ResultVO
}

type MyJobService struct {
	repo      port.JobRepository
	scheduler port.JobScheduler
}

func NewJobService(repo port.JobRepository, scheduler port.JobScheduler) *MyJobService {
	return &MyJobService{repo: repo, scheduler: scheduler}
}

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
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := j.validateJob(vo); err != nil {
		return model.ResultFromError(err)
	}
	if err := j.jobRepository().SaveOrUpdate(c.Request.Context(), jobEntity(vo)); err != nil {
		return model.ResultFromError(err)
	}
	if err := j.reload(c.Request.Context()); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (j *MyJobService) UpdateJob(c *gin.Context) model.ResultVO {
	var vo model.JobVO
	if err := c.ShouldBind(&vo); err != nil {
		slog.Error("bind job failed", "error", err)
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if vo.Id == 0 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := j.validateJob(vo); err != nil {
		return model.ResultFromError(err)
	}
	if err := j.jobRepository().SaveOrUpdate(c.Request.Context(), jobEntity(vo)); err != nil {
		return model.ResultFromError(err)
	}
	if err := j.reload(c.Request.Context()); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (j *MyJobService) DeleteJobById(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		slog.Error("bind job IDs failed", "error", err)
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := j.jobRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	if err := j.reload(c.Request.Context()); err != nil {
		return model.ResultFromError(err)
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
	return model.ResultOkWithData(j.jobDTO(job))
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
	jobs, count, err := j.jobRepository().List(c.Request.Context(), current, size, port.JobFilter{
		JobName: vo.JobName, JobGroup: vo.JobGroup, Status: vo.Status,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	dtos := make([]model.JobDTO, 0, len(jobs))
	for _, job := range jobs {
		dtos = append(dtos, j.jobDTO(job))
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: count})
}

func (j *MyJobService) UpdateJobStatus(c *gin.Context) model.ResultVO {
	var vo model.JobStatusVO
	if err := c.ShouldBind(&vo); err != nil || vo.Id <= 0 || (vo.Status != 0 && vo.Status != 1) {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := j.jobRepository().UpdateStatus(c.Request.Context(), vo.Id, vo.Status); err != nil {
		return model.ResultFromError(err)
	}
	if err := j.reload(c.Request.Context()); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (j *MyJobService) RunJob(c *gin.Context) model.ResultVO {
	var request model.JobRunVO
	if err := c.ShouldBind(&request); err != nil || request.Id <= 0 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if j.scheduler == nil {
		return model.ResultFromError(apperrors.Unavailable("job.run", nil))
	}
	job, err := j.jobRepository().Get(c.Request.Context(), request.Id)
	if err != nil {
		return model.ResultFromError(err)
	}
	result, err := j.scheduler.Run(c.Request.Context(), job, "manual")
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.JobRunOutcomeDTO{
		JobId: job.Id, Target: job.InvokeTarget, Processed: result.Processed, Message: result.Message,
	})
}

func (j *MyJobService) ListJobGroup(c *gin.Context) model.ResultVO {
	groups, err := j.jobRepository().ListGroups(c.Request.Context())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(groups)
}

func (j *MyJobService) ListJobTargets(c *gin.Context) model.ResultVO {
	if j.scheduler == nil {
		return model.ResultFromError(apperrors.Unavailable("job.targets", nil))
	}
	primitive := j.scheduler.Targets()
	targets := make([]model.JobTargetDTO, 0, len(primitive))
	for _, target := range primitive {
		targets = append(targets, model.JobTargetDTO{
			Target: target.Target, Name: target.Name,
			Description: target.Description, CronExample: target.CronExample,
		})
	}
	return model.ResultOkWithData(targets)
}

func (j *MyJobService) validateJob(vo model.JobVO) error {
	if strings.TrimSpace(vo.JobName) == "" || strings.TrimSpace(vo.JobGroup) == "" || strings.TrimSpace(vo.InvokeTarget) == "" || strings.TrimSpace(vo.CronExpression) == "" {
		return apperrors.Invalid("job.validate", "job fields are required")
	}
	if vo.Status != 0 && vo.Status != 1 {
		return apperrors.Invalid("job.validate", "job status is invalid")
	}
	if vo.Concurrent != 0 && vo.Concurrent != 1 {
		return apperrors.Invalid("job.validate", "job concurrency is invalid")
	}
	if j.scheduler == nil || !j.scheduler.SupportsTarget(strings.TrimSpace(vo.InvokeTarget)) {
		return apperrors.Invalid("job.validate", "job target is not registered")
	}
	if _, err := j.scheduler.NextRun(strings.TrimSpace(vo.CronExpression), time.Now()); err != nil {
		return err
	}
	return nil
}

func (j *MyJobService) reload(ctx context.Context) error {
	if j.scheduler == nil {
		return apperrors.Unavailable("job.reload", nil)
	}
	return j.scheduler.Reload(ctx)
}

func (j *MyJobService) jobDTO(job entity.TJob) model.JobDTO {
	dto := model.JobDTO{
		Id:             job.Id,
		JobName:        job.JobName,
		JobGroup:       job.JobGroup,
		InvokeTarget:   job.InvokeTarget,
		CronExpression: job.CronExpression,
		MisfirePolicy:  strconv.Itoa(job.MisfirePolicy),
		Concurrent:     job.Concurrent,
		Status:         job.Status,
		CreateTime:     job.CreateTime,
		Remark:         job.Remark,
	}
	if j.scheduler == nil || !j.scheduler.SupportsTarget(job.InvokeTarget) {
		dto.RunOnceReason = "job target is not registered"
		return dto
	}
	dto.CanRunOnce = true
	next, err := j.scheduler.NextRun(job.CronExpression, time.Now())
	if err != nil {
		return dto
	}
	dto.NextValidTime = &next
	return dto
}

func jobEntity(vo model.JobVO) entity.TJob {
	return entity.TJob{
		Id:             vo.Id,
		JobName:        strings.TrimSpace(vo.JobName),
		JobGroup:       strings.TrimSpace(vo.JobGroup),
		InvokeTarget:   strings.TrimSpace(vo.InvokeTarget),
		CronExpression: strings.TrimSpace(vo.CronExpression),
		MisfirePolicy:  3,
		Concurrent:     vo.Concurrent,
		Status:         vo.Status,
		Remark:         strings.TrimSpace(vo.Remark),
	}
}
