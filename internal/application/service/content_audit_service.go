package service

import (
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type ContentAuditService interface {
	ListContentAudits(c *gin.Context) model.ResultVO
	ListContentAuditItems(c *gin.Context) model.ResultVO
}

type MyContentAuditService struct{ repo port.ContentAuditRepository }

func NewContentAuditService(repo port.ContentAuditRepository) *MyContentAuditService {
	return &MyContentAuditService{repo: repo}
}

func (s *MyContentAuditService) ListContentAudits(c *gin.Context) model.ResultVO {
	if s == nil || s.repo == nil {
		return model.ResultFailWithMessage("内容审计暂不可用")
	}
	var vo model.ContentAuditFilterVO
	if err := c.ShouldBindQuery(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.ContentAuditFilter{
		Current: current, Size: size,
		ContentType: strings.ToLower(strings.TrimSpace(vo.ContentType)),
		Operation:   strings.TrimSpace(vo.Operation), Result: strings.TrimSpace(vo.Result),
		Keyword: strings.TrimSpace(vo.Keywords),
	}
	if filter.ContentType != "" && filter.ContentType != string(port.StudioContentArticle) && filter.ContentType != string(port.StudioContentTalk) && filter.ContentType != string(port.StudioContentSeries) {
		return model.ResultFailWithMessage("内容类型不正确")
	}
	if filter.Operation != "" && filter.Operation != "batch_status" && filter.Operation != "batch_delete" && filter.Operation != "publish_retry" {
		return model.ResultFailWithMessage("操作类型不正确")
	}
	if filter.Result != "" && filter.Result != "success" && filter.Result != "failed" {
		return model.ResultFailWithMessage("结果类型不正确")
	}
	if value := strings.TrimSpace(vo.StartDate); value != "" {
		filter.StartDate, err = time.ParseInLocation("2006-01-02", value, time.Local)
		if err != nil {
			return model.ResultFailWithMessage("开始日期不正确")
		}
	}
	if value := strings.TrimSpace(vo.EndDate); value != "" {
		filter.EndDate, err = time.ParseInLocation("2006-01-02", value, time.Local)
		if err != nil {
			return model.ResultFailWithMessage("结束日期不正确")
		}
		filter.EndDate = filter.EndDate.AddDate(0, 0, 1)
	}
	if !filter.StartDate.IsZero() && !filter.EndDate.IsZero() && !filter.EndDate.After(filter.StartDate) {
		return model.ResultFailWithMessage("日期范围不正确")
	}
	records, count, err := s.repo.List(c.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: int(count), Page: current, PageSize: size})
}

func (s *MyContentAuditService) ListContentAuditItems(c *gin.Context) model.ResultVO {
	if s == nil || s.repo == nil {
		return model.ResultFailWithMessage("内容审计暂不可用")
	}
	auditID, err := pathID(c, "auditId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	items, count, err := s.repo.ListItems(c.Request.Context(), auditID, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: items, Count: int(count), Page: current, PageSize: size})
}
