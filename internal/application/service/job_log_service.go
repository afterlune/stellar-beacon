package service

import (
	"context"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type JobLogService interface {
	ListJobLogs(context.Context, int, int, port.JobLogFilter) ([]entity.TJobLog, int64, error)
	DeleteJobLogs(context.Context, []int) error
	CleanJobLogs(context.Context) error
	ListJobLogGroups(context.Context) (string, error)
}

type MyJobLogService struct{ repo port.JobLogRepository }

func NewJobLogService(repo port.JobLogRepository) *MyJobLogService {
	return &MyJobLogService{repo: repo}
}

func (j *MyJobLogService) jobLogRepository() port.JobLogRepository {
	if j.repo != nil {
		return j.repo
	}
	return jobLogRepo
}

func (j *MyJobLogService) ListJobLogs(ctx context.Context, current, size int, filter port.JobLogFilter) ([]entity.TJobLog, int64, error) {
	return j.jobLogRepository().List(ctx, current, size, filter)
}

func (j *MyJobLogService) DeleteJobLogs(ctx context.Context, ids []int) error {
	return j.jobLogRepository().Delete(ctx, ids)
}

func (j *MyJobLogService) CleanJobLogs(ctx context.Context) error {
	return j.jobLogRepository().Clean(ctx)
}

func (j *MyJobLogService) ListJobLogGroups(ctx context.Context) (string, error) {
	return j.jobLogRepository().ListGroups(ctx)
}
