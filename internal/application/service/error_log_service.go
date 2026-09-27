package service

import (
	"context"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type ErrorLogService interface {
	ListErrorLogs(context.Context, int, int, string) ([]entity.TExceptionLog, int64, error)
	DeleteErrorLogs(context.Context, []int) error
}

type MyErrorLogService struct{ repo port.ErrorLogRepository }

func NewErrorLogService(repo port.ErrorLogRepository) *MyErrorLogService {
	return &MyErrorLogService{repo: repo}
}

func (e *MyErrorLogService) errorLogRepository() port.ErrorLogRepository {
	if e.repo != nil {
		return e.repo
	}
	return errorLogRepo
}

func (e *MyErrorLogService) ListErrorLogs(ctx context.Context, current, size int, keywords string) ([]entity.TExceptionLog, int64, error) {
	return e.errorLogRepository().List(ctx, current, size, keywords)
}

func (e *MyErrorLogService) DeleteErrorLogs(ctx context.Context, ids []int) error {
	return e.errorLogRepository().Delete(ctx, ids)
}
