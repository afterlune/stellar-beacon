package service

import (
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type GrowthService interface {
	Track(*gin.Context) model.ResultVO
	Summary(*gin.Context) model.ResultVO
}

type GrowthServiceDeps struct {
	Repo    port.GrowthRepository
	Limiter port.RateLimiter
}

type MyGrowthService struct {
	repo    port.GrowthRepository
	limiter port.RateLimiter
}

var growthEvents = map[string]struct{}{
	"share_click":       {},
	"subscribe_start":   {},
	"subscribe_confirm": {},
	"unsubscribe":       {},
}

func NewGrowthService(deps GrowthServiceDeps) (*MyGrowthService, error) {
	if deps.Repo == nil {
		return nil, missingServiceDependency("growth", "repository")
	}
	return &MyGrowthService{repo: deps.Repo, limiter: deps.Limiter}, nil
}

func (s *MyGrowthService) Track(c *gin.Context) model.ResultVO {
	var request model.GrowthEventVO
	if err := c.ShouldBindJSON(&request); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	request.EventName = strings.TrimSpace(request.EventName)
	if _, ok := growthEvents[request.EventName]; !ok {
		return model.ResultFailWithMessage("不支持的事件类型")
	}
	if s.limiter != nil {
		allowed, err := allowRateLimit(c.Request.Context(), s.limiter, "growth:event:ip:", c.ClientIP(), 60, time.Minute)
		if err != nil {
			return model.ResultFromError(err)
		}
		if !allowed {
			return model.ResultFailWithCodeAndMessage(42900, "请求过于频繁，请稍后再试")
		}
	}
	path := strings.TrimSpace(request.Path)
	if len(path) > 255 {
		path = path[:255]
	}
	if path == "" {
		path = c.Request.URL.Path
	}
	if err := s.repo.RecordEvent(c.Request.Context(), entity.TGrowthEvent{EventName: request.EventName, ArticleId: request.ArticleId, Path: path}); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyGrowthService) Summary(c *gin.Context) model.ResultVO {
	days := 30
	if value := c.Query("days"); value != "" {
		if parsed, err := parsePositiveID(value); err == nil && parsed <= 90 {
			days = parsed
		}
	}
	items, err := s.repo.Summary(c.Request.Context(), time.Now().AddDate(0, 0, -days))
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(items)
}

var _ GrowthService = (*MyGrowthService)(nil)
