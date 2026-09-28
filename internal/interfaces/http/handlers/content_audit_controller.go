package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func ListContentAudits(c *gin.Context) {
	var request model.ContentAuditFilterVO
	if err := c.ShouldBindQuery(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	filter := port.ContentAuditFilter{
		Current: current, Size: size,
		ContentType: strings.ToLower(strings.TrimSpace(request.ContentType)),
		Operation:   strings.TrimSpace(request.Operation),
		Result:      strings.TrimSpace(request.Result),
		Keyword:     strings.TrimSpace(request.Keywords),
	}
	if filter.ContentType != "" && filter.ContentType != string(port.StudioContentArticle) && filter.ContentType != string(port.StudioContentTalk) && filter.ContentType != string(port.StudioContentSeries) {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("内容类型不正确"))
		return
	}
	if filter.Operation != "" && filter.Operation != "batch_status" && filter.Operation != "batch_delete" && filter.Operation != "publish_retry" {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("操作类型不正确"))
		return
	}
	if filter.Result != "" && filter.Result != "success" && filter.Result != "failed" {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("结果类型不正确"))
		return
	}
	if value := strings.TrimSpace(request.StartDate); value != "" {
		var err error
		filter.StartDate, err = time.ParseInLocation("2006-01-02", value, time.Local)
		if err != nil {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("开始日期不正确"))
			return
		}
	}
	if value := strings.TrimSpace(request.EndDate); value != "" {
		var err error
		filter.EndDate, err = time.ParseInLocation("2006-01-02", value, time.Local)
		if err != nil {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("结束日期不正确"))
			return
		}
		filter.EndDate = filter.EndDate.AddDate(0, 0, 1)
	}
	if !filter.StartDate.IsZero() && !filter.EndDate.IsZero() && !filter.EndDate.After(filter.StartDate) {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("日期范围不正确"))
		return
	}
	records, count, err := contentAuditService.ListContentAudits(c.Request.Context(), filter)
	if err != nil {
		writeContentAuditError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: int(count), Page: current, PageSize: size}))
}

func ListContentAuditItems(c *gin.Context) {
	auditID, err := strconv.Atoi(c.Param("auditId"))
	if err != nil || auditID <= 0 {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	items, count, err := contentAuditService.ListContentAuditItems(c.Request.Context(), auditID, current, size)
	if err != nil {
		writeContentAuditError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: items, Count: int(count), Page: current, PageSize: size}))
}

func writeContentAuditError(c *gin.Context, err error) {
	if apperrors.Op(err) == "content_audit.service" {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("内容审计暂不可用"))
		return
	}
	c.JSON(http.StatusOK, model.ResultFromError(err))
}
