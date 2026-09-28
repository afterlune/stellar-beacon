package api

import (
	"net/http"

	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ListOperationLogs
// @Summary         操作日志模块
// @Description    查看操作日志
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/logs/operations [GET]
func ListOperationLogs(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	records, count, err := operationLogService.ListOperationLogs(c.Request.Context(), query.Current, query.Size, query.Keywords)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	dtos, err := mapResponseDTO[[]model.OperationLogDTO](records)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	if dtos == nil {
		dtos = []model.OperationLogDTO{}
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)}))
}

// DeleteOperationLogs
// @Summary         操作日志模块
// @Description    删除操作日志
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/logs/operations [DELETE]
func DeleteOperationLogs(c *gin.Context) {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := operationLogService.DeleteOperationLogs(c.Request.Context(), ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}
