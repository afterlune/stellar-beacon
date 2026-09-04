package service

import (
	"benetnasch/app/application/support"
	"benetnasch/app/domain/port"
	"container/list"
)

type OperationLogService interface {
	ListOperationLogs(c port.Request) port.ResultVO
	DeleteOperationLogs(c port.Request) port.ResultVO
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

func (o *MyOperationLogService) ListOperationLogs(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	logs, count, err := o.operationLogRepository().List(c.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	var dtos []port.OperationLogDTO
	support.StructCopy(logs, &dtos)
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: dtos, Count: int(count)})
}

func (o *MyOperationLogService) DeleteOperationLogs(c port.Request) port.ResultVO {
	var ids []int
	if err := c.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := o.operationLogRepository().Delete(c.Context(), ids); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}
