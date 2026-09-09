package service

import (
	"benetnasch/internal/application/support"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/interfaces/http/model"
	"container/list"

	"github.com/gin-gonic/gin"
)

type ErrorLogService interface {
	ListErrorLogs(c *gin.Context) model.ResultVO
	DeleteErrorLogs(c *gin.Context) model.ResultVO
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

func (e *MyErrorLogService) ListErrorLogs(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	logs, count, err := e.errorLogRepository().List(c.Request.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.ExceptionLogDTO
	support.StructCopy(logs, &dtos)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)})
}

func (e *MyErrorLogService) DeleteErrorLogs(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := e.errorLogRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}
