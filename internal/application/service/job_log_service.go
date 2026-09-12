package service

import (
	"benetnasch/internal/domain/port"
	"benetnasch/internal/interfaces/http/model"
	"container/list"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
)

type JobLogService interface {
	ListJobLogs(c *gin.Context) model.ResultVO
	DeleteJobLogs(c *gin.Context) model.ResultVO
	CleanJobLogs() model.ResultVO
	ListJobLogGroups() model.ResultVO
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

func (j *MyJobLogService) ListJobLogs(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		current = 1
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		size = 10
	}
	var vo model.JobLogSearchVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var status *int
	if vo.Status != nil {
		value, ok := jobLogStatus(vo.Status)
		if !ok {
			return model.ResultFailWithMessage("状态参数无效")
		}
		status = &value
	}
	logs, count, err := j.jobLogRepository().List(c.Request.Context(), current, size, port.JobLogFilter{
		JobId: vo.JobId, JobName: vo.JobName, JobGroup: vo.JobGroup,
		Status: status, StartTime: vo.StartTime, EndTime: vo.EndTime,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.JobLogDTO
	StructCopy(logs, &dtos)
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
		parsed, err := strconv.Atoi(status)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func (j *MyJobLogService) DeleteJobLogs(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := j.jobLogRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (j *MyJobLogService) CleanJobLogs() model.ResultVO {
	if err := j.jobLogRepository().Clean(context.Background()); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (j *MyJobLogService) ListJobLogGroups() model.ResultVO {
	groups, err := j.jobLogRepository().ListGroups(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(groups)
}
