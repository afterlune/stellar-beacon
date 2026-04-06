package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"xorm.io/builder"
)

type OperationLogService interface {
	ListOperationLogs(c *gin.Context) model.ResultVO
	DeleteOperationLogs(c *gin.Context) model.ResultVO
}

type MyOperationLogService struct{}

func (o *MyOperationLogService) ListOperationLogs(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var operationLogs []entity.TOperationLog
	engine := ormInit.GetEngine()
	if vo.Keywords != "" {
		err = engine.Prepare().Where(builder.Like{"opt_module", vo.Keywords}).Or(builder.Like{"opt_desc", vo.Keywords}).
			OrderBy("id").Desc("id").Limit(vo.Size, vo.Size*(vo.Current-1)).Find(&operationLogs)
	} else {
		err = engine.Prepare().OrderBy("id").Desc("id").Limit(vo.Size, vo.Size*(vo.Current-1)).Find(&operationLogs)
	}
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	count, err := engine.Count(&entity.TOperationLog{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var operationLogDTOs []model.OperationLogDTO
	shared.StructCopy(operationLogs, &operationLogDTOs)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: operationLogDTOs, Count: int(count)})
}

func (o *MyOperationLogService) DeleteOperationLogs(c *gin.Context) model.ResultVO {
	var iDs []int
	err := c.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	_, err = ormInit.GetEngine().In("id", iDs).Delete(&entity.TOperationLog{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}
