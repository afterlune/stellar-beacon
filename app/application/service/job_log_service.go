package service

import (
	"benetnasch/app/application/support"
	"benetnasch/app/domain/port"
	"container/list"
	"context"
	"strconv"
)

type JobLogService interface {
	ListJobLogs(c port.Request) port.ResultVO
	DeleteJobLogs(c port.Request) port.ResultVO
	CleanJobLogs(ctx context.Context) port.ResultVO
	ListJobLogGroups(ctx context.Context) port.ResultVO
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

func (j *MyJobLogService) ListJobLogs(c port.Request) port.ResultVO {
	current, size, ok := parsePageQuery(c.Query("current"), c.Query("size"), 1, 10)
	if !ok {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	var vo port.JobLogSearchVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	var status *int
	if vo.Status != nil {
		value, ok := jobLogStatus(vo.Status)
		if !ok {
			return port.ResultFailWithMessage("状态参数无效")
		}
		status = &value
	}
	logs, count, err := j.jobLogRepository().List(c.Context(), current, size, port.JobLogFilter{
		JobId: vo.JobId, JobName: vo.JobName, JobGroup: vo.JobGroup,
		Status: status, StartTime: vo.StartTime, EndTime: vo.EndTime,
	})
	if err != nil {
		return port.ResultFromError(err)
	}
	var dtos []port.JobLogDTO
	support.StructCopy(logs, &dtos)
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: dtos, Count: int(count)})
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

func (j *MyJobLogService) DeleteJobLogs(c port.Request) port.ResultVO {
	var ids []int
	if err := c.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := j.jobLogRepository().Delete(c.Context(), ids); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (j *MyJobLogService) CleanJobLogs(ctx context.Context) port.ResultVO {
	if err := j.jobLogRepository().Clean(ctx); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (j *MyJobLogService) ListJobLogGroups(ctx context.Context) port.ResultVO {
	groups, err := j.jobLogRepository().ListGroups(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(groups)
}
