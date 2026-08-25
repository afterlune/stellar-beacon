package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/persistence/repository"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"strconv"
	"xorm.io/xorm"
)

type JobLogService interface {
	ListJobLogs(c *gin.Context) model.ResultVO
	DeleteJobLogs(c *gin.Context) model.ResultVO
	CleanJobLogs() model.ResultVO
	ListJobLogGroups() model.ResultVO
}

type MyJobLogService struct{}

func (j *MyJobLogService) ListJobLogs(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		current = 1
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		size = pgsql.DefaultPageSize
	}
	var vo model.JobLogSearchVO
	if err = c.ShouldBind(&vo); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	applyFilters := func(session *xorm.Session) *xorm.Session {
		session = session.Where("1 = 1")
		if vo.JobId != 0 {
			session = session.And("job_id = ?", vo.JobId)
		}
		if vo.JobGroup != "" {
			session = session.And("job_group LIKE ? ESCAPE '\\'", pgsql.ContainsPattern(vo.JobGroup))
		}
		if vo.JobName != "" {
			session = session.And("job_name LIKE ? ESCAPE '\\'", pgsql.ContainsPattern(vo.JobName))
		}
		if vo.Status != nil {
			status, ok := jobLogStatus(vo.Status)
			if !ok {
				return nil
			}
			session = session.And("status = ?", status)
		}
		if vo.StartTime != "" && vo.EndTime != "" {
			session = session.And("create_time BETWEEN ? AND ?", vo.StartTime, vo.EndTime)
		}
		return session
	}
	if vo.Status != nil {
		if _, ok := jobLogStatus(vo.Status); !ok {
			return model.ResultFailWithMessage("状态参数无效")
		}
	}

	engine := ormInit.GetEngine()
	var count int64
	countSession := applyFilters(engine.Context(c.Request.Context()))
	if countSession == nil {
		return model.ResultFailWithMessage("状态参数无效")
	}
	if count, err = countSession.Count(&entity.TJobLog{}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	limit, offset := pgsql.Page(current, size)
	var joblogs []entity.TJobLog
	listSession := applyFilters(engine.Context(c.Request.Context()))
	if listSession == nil {
		return model.ResultFailWithMessage("状态参数无效")
	}
	if err = listSession.OrderBy("id").Desc("id").Limit(limit, offset).Find(&joblogs); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var dtos []model.JobLogDTO
	shared.StructCopy(joblogs, &dtos)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)})
}

func jobLogStatus(value any) (int, bool) {
	switch status := value.(type) {
	case int:
		return status, true
	case int8:
		return int(status), true
	case int16:
		return int(status), true
	case int32:
		return int(status), true
	case int64:
		return int(status), true
	case float32:
		return int(status), float32(int(status)) == status
	case float64:
		return int(status), float64(int(status)) == status
	case string:
		statusInt, err := strconv.Atoi(status)
		return statusInt, err == nil
	default:
		return 0, false
	}
}

func (j *MyJobLogService) DeleteJobLogs(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if len(ids) == 0 {
		return model.ResultOk()
	}
	if _, err := ormInit.GetEngine().Prepare().In("id", ids).Delete(&entity.TJobLog{}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (j *MyJobLogService) CleanJobLogs() model.ResultVO {
	if _, err := ormInit.GetEngine().Prepare().Delete(&entity.TJobLog{}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (j *MyJobLogService) ListJobLogGroups() model.ResultVO {
	return model.ResultOkWithData(repository.ListJobLogGroups())
}
