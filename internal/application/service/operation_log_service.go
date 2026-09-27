package service

import (
	"context"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type OperationLogService interface {
	ListOperationLogs(context.Context, int, int, string) ([]entity.TOperationLog, int64, error)
	DeleteOperationLogs(context.Context, []int) error
}

type MyOperationLogService struct{ repo port.OperationLogRepository }

func NewOperationLogService(repo port.OperationLogRepository) *MyOperationLogService {
	return &MyOperationLogService{repo: repo}
}

func (o *MyOperationLogService) operationLogRepository() port.OperationLogRepository {
	if o.repo != nil {
		return o.repo
	}
	return operationLogRepo
}

func (o *MyOperationLogService) ListOperationLogs(ctx context.Context, current, size int, keywords string) ([]entity.TOperationLog, int64, error) {
	return o.operationLogRepository().List(ctx, current, size, keywords)
}

func (o *MyOperationLogService) DeleteOperationLogs(ctx context.Context, ids []int) error {
	return o.operationLogRepository().Delete(ctx, ids)
}
