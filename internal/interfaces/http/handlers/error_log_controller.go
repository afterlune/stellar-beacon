package api

import (
	"net/http"

	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ListErrorLogs
// @Summary         异常日志模块
// @Description    获取异常日志
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/logs/exceptions [GET]
func ListErrorLogs(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	records, count, err := errorLogService.ListErrorLogs(c.Request.Context(), query.Current, query.Size, query.Keywords)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	dtos, err := mapResponseDTO[[]model.ExceptionLogDTO](records)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	if dtos == nil {
		dtos = []model.ExceptionLogDTO{}
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)}))
}

// DeleteErrorLogs
// @Summary         异常日志模块
// @Description    删除异常日志
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/logs/exceptions [DELETE]
func DeleteErrorLogs(c *gin.Context) {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := errorLogService.DeleteErrorLogs(c.Request.Context(), ids); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}
