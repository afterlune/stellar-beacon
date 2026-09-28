package service

import (
	"context"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type JobService interface {
	SaveJob(ctx context.Context, input JobInput) error
	UpdateJob(ctx context.Context, input JobInput) error
	DeleteJobs(ctx context.Context, ids []int) error
	GetJob(ctx context.Context, id int) (JobDetail, error)
	ListJobs(ctx context.Context, current, size int, filter port.JobFilter) ([]JobDetail, int, error)
	UpdateJobStatus(ctx context.Context, id, status int) error
	RunJob(ctx context.Context, id int) (JobRunOutcome, error)
	ListJobGroups(ctx context.Context) ([]string, error)
	ListJobTargets(ctx context.Context) ([]port.JobTarget, error)
}

type JobInput struct {
	ID             int
	JobName        string
	JobGroup       string
	InvokeTarget   string
	CronExpression string
	Concurrent     int
	Status         int
	Remark         string
}

type JobDetail struct {
	Job           entity.TJob
	CanRunOnce    bool
	RunOnceReason string
	NextValidTime *time.Time
}

type JobRunOutcome struct {
	JobID     int
	Target    string
	Processed bool
	Message   string
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

func (j *MyJobService) SaveJob(ctx context.Context, input JobInput) error {
	if err := j.validateJob(input); err != nil {
		return err
	}
	if err := j.jobRepository().SaveOrUpdate(ctx, jobEntity(input)); err != nil {
		return err
	}
	return j.reload(ctx)
}

func (j *MyJobService) UpdateJob(ctx context.Context, input JobInput) error {
	if input.ID == 0 {
		return apperrors.Invalid("job.update", "job id is required")
	}
	if err := j.validateJob(input); err != nil {
		return err
	}
	if err := j.jobRepository().SaveOrUpdate(ctx, jobEntity(input)); err != nil {
		return err
	}
	return j.reload(ctx)
}

func (j *MyJobService) DeleteJobs(ctx context.Context, ids []int) error {
	if err := j.jobRepository().Delete(ctx, ids); err != nil {
		return err
	}
	return j.reload(ctx)
}

func (j *MyJobService) GetJob(ctx context.Context, id int) (JobDetail, error) {
	job, err := j.jobRepository().Get(ctx, id)
	if err != nil {
		return JobDetail{}, err
	}
	return j.jobDetail(job), nil
}

func (j *MyJobService) ListJobs(ctx context.Context, current, size int, filter port.JobFilter) ([]JobDetail, int, error) {
	jobs, count, err := j.jobRepository().List(ctx, current, size, filter)
	if err != nil {
		return nil, 0, err
	}
	details := make([]JobDetail, 0, len(jobs))
	for _, job := range jobs {
		details = append(details, j.jobDetail(job))
	}
	return details, count, nil
}

func (j *MyJobService) UpdateJobStatus(ctx context.Context, id, status int) error {
	if id <= 0 || (status != 0 && status != 1) {
		return apperrors.Invalid("job.status", "job id or status is invalid")
	}
	if err := j.jobRepository().UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	return j.reload(ctx)
}

func (j *MyJobService) RunJob(ctx context.Context, id int) (JobRunOutcome, error) {
	if id <= 0 {
		return JobRunOutcome{}, apperrors.Invalid("job.run", "job id is invalid")
	}
	if j.scheduler == nil {
		return JobRunOutcome{}, apperrors.Unavailable("job.run", nil)
	}
	job, err := j.jobRepository().Get(ctx, id)
	if err != nil {
		return JobRunOutcome{}, err
	}
	result, err := j.scheduler.Run(ctx, job, "manual")
	if err != nil {
		return JobRunOutcome{}, err
	}
	return JobRunOutcome{JobID: job.Id, Target: job.InvokeTarget, Processed: result.Processed, Message: result.Message}, nil
}

func (j *MyJobService) ListJobGroups(ctx context.Context) ([]string, error) {
	return j.jobRepository().ListGroups(ctx)
}

func (j *MyJobService) ListJobTargets(context.Context) ([]port.JobTarget, error) {
	if j.scheduler == nil {
		return nil, apperrors.Unavailable("job.targets", nil)
	}
	return j.scheduler.Targets(), nil
}

func (j *MyJobService) validateJob(input JobInput) error {
	if strings.TrimSpace(input.JobName) == "" || strings.TrimSpace(input.JobGroup) == "" || strings.TrimSpace(input.InvokeTarget) == "" || strings.TrimSpace(input.CronExpression) == "" {
		return apperrors.Invalid("job.validate", "job fields are required")
	}
	if input.Status != 0 && input.Status != 1 {
		return apperrors.Invalid("job.validate", "job status is invalid")
	}
	if input.Concurrent != 0 && input.Concurrent != 1 {
		return apperrors.Invalid("job.validate", "job concurrency is invalid")
	}
	if j.scheduler == nil || !j.scheduler.SupportsTarget(strings.TrimSpace(input.InvokeTarget)) {
		return apperrors.Invalid("job.validate", "job target is not registered")
	}
	if _, err := j.scheduler.NextRun(strings.TrimSpace(input.CronExpression), time.Now()); err != nil {
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

func (j *MyJobService) jobDetail(job entity.TJob) JobDetail {
	detail := JobDetail{Job: job}
	if j.scheduler == nil || !j.scheduler.SupportsTarget(job.InvokeTarget) {
		detail.RunOnceReason = "job target is not registered"
		return detail
	}
	detail.CanRunOnce = true
	next, err := j.scheduler.NextRun(job.CronExpression, time.Now())
	if err == nil {
		detail.NextValidTime = &next
	}
	return detail
}

func jobEntity(input JobInput) entity.TJob {
	return entity.TJob{
		Id:             input.ID,
		JobName:        strings.TrimSpace(input.JobName),
		JobGroup:       strings.TrimSpace(input.JobGroup),
		InvokeTarget:   strings.TrimSpace(input.InvokeTarget),
		CronExpression: strings.TrimSpace(input.CronExpression),
		MisfirePolicy:  3,
		Concurrent:     input.Concurrent,
		Status:         input.Status,
		Remark:         strings.TrimSpace(input.Remark),
	}
}
