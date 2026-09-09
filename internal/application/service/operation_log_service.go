package service

import (
	"benetnasch/internal/application/support"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/interfaces/http/model"
	"container/list"

	"github.com/gin-gonic/gin"
)

type OperationLogService interface {
	ListOperationLogs(c *gin.Context) model.ResultVO
	DeleteOperationLogs(c *gin.Context) model.ResultVO
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

func (o *MyOperationLogService) ListOperationLogs(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	logs, count, err := o.operationLogRepository().List(c.Request.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.OperationLogDTO
	support.StructCopy(logs, &dtos)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)})
}

func (o *MyOperationLogService) DeleteOperationLogs(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := o.operationLogRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}
