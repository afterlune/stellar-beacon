package service

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"container/list"
	"context"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"
)

type JobService interface {
	SaveJob(c port.Request) port.ResultVO
	UpdateJob(c port.Request) port.ResultVO
	DeleteJobById(c port.Request) port.ResultVO
	GetJobById(c port.Request) port.ResultVO
	ListJobs(c port.Request) port.ResultVO
	UpdateJobStatus(c port.Request) port.ResultVO
	RunJob(c port.Request) port.ResultVO
	ListJobGroup(ctx context.Context) port.ResultVO
}

type MyJobService struct {
	repo   port.JobRepository
	runner port.JobRunner
}

func NewJobService(repo port.JobRepository, runner port.JobRunner) *MyJobService {
	return &MyJobService{repo: repo, runner: runner}
}

func (j *MyJobService) jobRepository() port.JobRepository {
	if j.repo != nil {
		return j.repo
	}
	return jobRepo
}

func (j *MyJobService) SaveJob(c port.Request) port.ResultVO {
	var vo port.JobVO
	if err := c.Bind(&vo); err != nil {
		slog.Error("bind job failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := validateJobVO(vo, false); err != nil {
		return port.ResultFromError(err)
	}
	job := jobEntity(vo)
	job.Id = 0
	if err := j.jobRepository().Save(c.Context(), job); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (j *MyJobService) UpdateJob(c port.Request) port.ResultVO {
	var vo port.JobVO
	if err := c.Bind(&vo); err != nil {
		slog.Error("bind job failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := validateJobVO(vo, true); err != nil {
		return port.ResultFromError(err)
	}
	if err := j.jobRepository().Update(c.Context(), jobEntity(vo)); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (j *MyJobService) DeleteJobById(c port.Request) port.ResultVO {
	var ids []int
	if err := c.Bind(&ids); err != nil {
		slog.Error("bind job IDs failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := validateIDs(ids); err != nil {
		return port.ResultFromError(err)
	}
	if err := j.jobRepository().Delete(c.Context(), ids); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (j *MyJobService) GetJobById(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	job, err := j.jobRepository().Get(c.Context(), id)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(j.jobDTO(job))
}

func (j *MyJobService) ListJobs(c port.Request) port.ResultVO {
	current, size, ok := parsePageQuery(c.Query("current"), c.Query("size"), 1, 10)
	if !ok {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	var vo port.JobSearchVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFail()
	}
	jobs, count, err := j.jobRepository().List(c.Context(), current, size, port.JobFilter{JobName: vo.JobName, JobGroup: vo.JobGroup, Status: vo.Status})
	if err != nil {
		return port.ResultFromError(err)
	}
	dtos := make([]port.JobDTO, 0, len(jobs))
	for _, job := range jobs {
		dtos = append(dtos, j.jobDTO(job))
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: dtos, Count: count})
}

func (j *MyJobService) UpdateJobStatus(c port.Request) port.ResultVO {
	var vo port.JobStatusVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if vo.Id <= 0 || (vo.Status != 0 && vo.Status != 1) {
		return port.ResultFromError(apperrors.Invalid("job.status", "job id or status is invalid"))
	}
	if err := j.jobRepository().UpdateStatus(c.Context(), vo.Id, vo.Status); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (j *MyJobService) RunJob(c port.Request) port.ResultVO {
	var vo port.JobRunVO
	if err := c.Bind(&vo); err != nil {
		slog.Error("bind job run request failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if vo.Id <= 0 {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	job, err := j.jobRepository().Get(c.Context(), vo.Id)
	if err != nil {
		return port.ResultFromError(err)
	}
	if group := strings.TrimSpace(vo.JobGroup); group != "" && group != job.JobGroup {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if j.runner == nil {
		return port.ResultFailWithMessage("手动执行任务暂未启用")
	}
	if !j.runner.CanRun(job.InvokeTarget) {
		return port.ResultFailWithMessage("该任务目标暂不支持手动执行")
	}
	outcome, err := j.runner.Run(c.Context(), port.JobRunRequest{
		ID:           job.Id,
		JobGroup:     job.JobGroup,
		JobName:      job.JobName,
		InvokeTarget: job.InvokeTarget,
	})
	if err != nil {
		return port.ResultFromError(err)
	}
	message := "没有可执行的队列任务"
	if outcome.Processed {
		message = "任务已执行一次"
	}
	return port.ResultOkWithDataAndMessage(port.JobRunOutcomeDTO{
		JobID:     outcome.JobID,
		Target:    outcome.Target,
		Processed: outcome.Processed,
	}, message)
}

func (j *MyJobService) ListJobGroup(ctx context.Context) port.ResultVO {
	groups, err := j.jobRepository().ListGroups(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(groups)
}

func validateJobVO(vo port.JobVO, requireID bool) error {
	if requireID && vo.Id <= 0 {
		return apperrors.Invalid("job.validate", "job id is required")
	}
	if !boundedJobField(vo.JobName, 64) || !boundedJobField(vo.JobGroup, 64) ||
		!boundedJobField(vo.InvokeTarget, 500) || !boundedJobField(vo.CronExpression, 255) ||
		!boundedJobField(vo.Remark, 500) {
		return apperrors.Invalid("job.validate", "job field is invalid")
	}
	if strings.TrimSpace(vo.JobName) == "" || strings.TrimSpace(vo.JobGroup) == "" ||
		strings.TrimSpace(vo.InvokeTarget) == "" || strings.TrimSpace(vo.CronExpression) == "" {
		return apperrors.Invalid("job.validate", "job name, group, invoke target and cron expression are required")
	}
	if vo.MisfirePolicy < 0 || vo.MisfirePolicy > 3 || (vo.Concurrent != 0 && vo.Concurrent != 1) || (vo.Status != 0 && vo.Status != 1) {
		return apperrors.Invalid("job.validate", "job policy or status is invalid")
	}
	return nil
}

func boundedJobField(value string, maxRunes int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(strings.TrimSpace(value)) <= maxRunes
}

func validateIDs(ids []int) error {
	if len(ids) == 0 {
		return apperrors.Invalid("job.ids", "at least one job id is required")
	}
	for _, id := range ids {
		if id <= 0 {
			return apperrors.Invalid("job.ids", "job id must be positive")
		}
	}
	return nil
}

func jobEntity(vo port.JobVO) port.TJob {
	return port.TJob{
		Id:             vo.Id,
		JobName:        strings.TrimSpace(vo.JobName),
		JobGroup:       strings.TrimSpace(vo.JobGroup),
		InvokeTarget:   strings.TrimSpace(vo.InvokeTarget),
		CronExpression: strings.TrimSpace(vo.CronExpression),
		MisfirePolicy:  vo.MisfirePolicy,
		Concurrent:     vo.Concurrent,
		Status:         vo.Status,
		Remark:         strings.TrimSpace(vo.Remark),
	}
}

func (j *MyJobService) jobDTO(job port.TJob) port.JobDTO {
	dto := port.JobDTO{
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
	if j.runner == nil {
		dto.RunOnceReason = "安全执行器未配置"
	} else if j.runner.CanRun(job.InvokeTarget) {
		dto.CanRunOnce = true
	} else {
		dto.RunOnceReason = "目标不在安全白名单或对应 Worker 未启用"
	}
	return dto
}
