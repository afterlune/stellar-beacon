package service

import (
	"strconv"
	"strings"
	"unicode/utf8"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type SystemMonitorService interface {
	Snapshot(*gin.Context) model.ResultVO
	Trends(*gin.Context) model.ResultVO
	Timeline(*gin.Context) model.ResultVO
	Incidents(*gin.Context) model.ResultVO
	Incident(*gin.Context) model.ResultVO
	AddIncidentUpdate(*gin.Context) model.ResultVO
}

type MySystemMonitorService struct{ monitor port.SystemMonitor }

func NewSystemMonitorService(monitor port.SystemMonitor) *MySystemMonitorService {
	return &MySystemMonitorService{monitor: monitor}
}

func (s *MySystemMonitorService) Snapshot(c *gin.Context) model.ResultVO {
	if s == nil || s.monitor == nil {
		return model.ResultOkWithData(port.MonitorSnapshot{
			Status: port.MonitorStatusUnknown, HistoryAvailable: false,
			HistoryMessage: "monitorServiceStopped", Instances: []port.MonitorInstance{},
		})
	}
	return model.ResultOkWithData(s.monitor.Snapshot(c.Request.Context()))
}

func (s *MySystemMonitorService) Trends(c *gin.Context) model.ResultVO {
	if s == nil || s.monitor == nil {
		return model.ResultOkWithData(port.MonitorTrends{
			Range: "1h", Available: false, Points: []port.MonitorTrendPoint{},
		})
	}
	rangeValue := c.Query("range")
	if rangeValue == "" {
		rangeValue = "1h"
	}
	trends, err := s.monitor.Trends(c.Request.Context(), rangeValue)
	if err != nil {
		return model.ResultFromError(apperrors.Invalid("system_monitor.range", "range must be 1h, 24h, or 7d"))
	}
	return model.ResultOkWithData(trends)
}

func (s *MySystemMonitorService) Timeline(c *gin.Context) model.ResultVO {
	if s == nil || s.monitor == nil {
		return model.ResultOkWithData(port.MonitorTimeline{
			Range: "90d", Available: false, Message: "monitorServiceStopped", Periods: []port.MonitorStatusPeriod{},
		})
	}
	rangeValue := c.Query("range")
	if rangeValue == "" {
		rangeValue = "90d"
	}
	timeline, err := s.monitor.Timeline(c.Request.Context(), rangeValue)
	if err != nil {
		return model.ResultFromError(apperrors.Invalid("system_monitor.timeline.range", "range must be 24h, 7d, 30d, or 90d"))
	}
	return model.ResultOkWithData(timeline)
}

func (s *MySystemMonitorService) Incidents(c *gin.Context) model.ResultVO {
	if s == nil || s.monitor == nil {
		return model.ResultOkWithData([]port.MonitorIncident{})
	}
	rangeValue := c.Query("range")
	if rangeValue == "" {
		rangeValue = "90d"
	}
	limit := 100
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 200 {
			return model.ResultFromError(apperrors.Invalid("system_monitor.incidents.limit", "limit must be between 1 and 200"))
		}
		limit = parsed
	}
	incidents, err := s.monitor.Incidents(c.Request.Context(), rangeValue, limit)
	if err != nil {
		return model.ResultFromError(apperrors.Invalid("system_monitor.incidents.range", "range must be 24h, 7d, 30d, or 90d"))
	}
	return model.ResultOkWithData(incidents)
}

func (s *MySystemMonitorService) Incident(c *gin.Context) model.ResultVO {
	if s == nil || s.monitor == nil {
		return model.ResultFromError(apperrors.Unavailable("system_monitor.incident", nil))
	}
	id, err := strconv.ParseInt(c.Param("incidentId"), 10, 64)
	if err != nil || id < 1 {
		return model.ResultFromError(apperrors.Invalid("system_monitor.incident.id", "incidentId must be a positive integer"))
	}
	incident, err := s.monitor.Incident(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(incident)
}

func (s *MySystemMonitorService) AddIncidentUpdate(c *gin.Context) model.ResultVO {
	if s == nil || s.monitor == nil {
		return model.ResultFromError(apperrors.Unavailable("system_monitor.incident_update", nil))
	}
	id, err := strconv.ParseInt(c.Param("incidentId"), 10, 64)
	if err != nil || id < 1 {
		return model.ResultFromError(apperrors.Invalid("system_monitor.incident.id", "incidentId must be a positive integer"))
	}
	var request struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		return model.ResultFromError(apperrors.Invalid("system_monitor.incident_update.body", "content is required"))
	}
	content := strings.TrimSpace(request.Content)
	if content == "" || utf8.RuneCountInString(content) > 4000 {
		return model.ResultFromError(apperrors.Invalid("system_monitor.incident_update.content", "content must contain 1 to 4000 characters"))
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindForbidden, "system_monitor.incident_update.author", nil))
	}
	user, ok := value.(model.UserDetailsDTO)
	if !ok || user.UserInfoId <= 0 {
		return model.ResultFromError(apperrors.New(apperrors.KindForbidden, "system_monitor.incident_update.author", nil))
	}
	name := strings.TrimSpace(user.Nickname)
	if name == "" {
		name = strings.TrimSpace(user.Username)
	}
	if name == "" {
		name = strings.TrimSpace(user.Email)
	}
	update, err := s.monitor.AddIncidentUpdate(c.Request.Context(), id, user.UserInfoId, name, content)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(update)
}
