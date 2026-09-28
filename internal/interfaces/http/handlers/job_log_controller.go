package api

import (
	"net/http"
	"strconv"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ListJobLogs
// @Summary         定时任务日志模块
// @Description    获取定时任务的日志列表
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/logs/jobs [GET]
func ListJobLogs(c *gin.Context) {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		current = 1
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		size = 10
	}
	var query model.JobLogSearchVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	var status *int
	if query.Status != nil {
		value, ok := jobLogStatus(query.Status)
		if !ok {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("状态参数无效"))
			return
		}
		status = &value
	}
	filter := port.JobLogFilter{
		JobId: query.JobId, JobName: query.JobName, JobGroup: query.JobGroup,
		Status: status, StartTime: query.StartTime, EndTime: query.EndTime,
	}
	logs, count, err := jobLogService.ListJobLogs(c.Request.Context(), current, size, filter)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	dtos, err := mapResponseDTO[[]model.JobLogDTO](logs)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	if dtos == nil {
		dtos = []model.JobLogDTO{}
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)}))
}

// DeleteJobLogs
// @Summary         定时任务日志模块
// @Description    删除定时任务的日志
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/logs/jobs [DELETE]
func DeleteJobLogs(c *gin.Context) {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := jobLogService.DeleteJobLogs(c.Request.Context(), ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// CleanJobLogs
// @Summary         定时任务日志模块
// @Description    清除定时任务的日志
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/logs/jobs/clean [DELETE]
func CleanJobLogs(c *gin.Context) {
	if err := jobLogService.CleanJobLogs(c.Request.Context()); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// ListJobLogGroups
// @Summary         定时任务日志模块
// @Description    获取定时任务日志的所有组名
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/logs/jobs/groups [GET]
func ListJobLogGroups(c *gin.Context) {
	groups, err := jobLogService.ListJobLogGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(groups))
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
