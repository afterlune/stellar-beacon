package service

import (
	"benetnasch/app/application/support"
	"benetnasch/app/domain/port"
	"container/list"
)

type ErrorLogService interface {
	ListErrorLogs(c port.Request) port.ResultVO
	DeleteErrorLogs(c port.Request) port.ResultVO
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

func (e *MyErrorLogService) ListErrorLogs(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	logs, count, err := e.errorLogRepository().List(c.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	var dtos []port.ExceptionLogDTO
	support.StructCopy(logs, &dtos)
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: dtos, Count: int(count)})
}

func (e *MyErrorLogService) DeleteErrorLogs(c port.Request) port.ResultVO {
	var ids []int
	if err := c.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := e.errorLogRepository().Delete(c.Context(), ids); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}
