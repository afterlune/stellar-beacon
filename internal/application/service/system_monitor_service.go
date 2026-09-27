package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type SystemMonitorService interface {
	Snapshot(context.Context) port.MonitorSnapshot
	Trends(context.Context, string) (port.MonitorTrends, error)
	Timeline(context.Context, string) (port.MonitorTimeline, error)
	Incidents(context.Context, string, int) ([]port.MonitorIncident, error)
	Incident(context.Context, int64) (port.MonitorIncident, error)
	AddIncidentUpdate(context.Context, int64, entity.AuthSession, string) (port.MonitorIncidentUpdate, error)
}

type MySystemMonitorService struct{ monitor port.SystemMonitor }

func NewSystemMonitorService(monitor port.SystemMonitor) *MySystemMonitorService {
	return &MySystemMonitorService{monitor: monitor}
}

func (s *MySystemMonitorService) Snapshot(ctx context.Context) port.MonitorSnapshot {
	if s == nil || s.monitor == nil {
		return port.MonitorSnapshot{
			Status: port.MonitorStatusUnknown, HistoryAvailable: false,
			HistoryMessage: "monitorServiceStopped", Instances: []port.MonitorInstance{},
		}
	}
	return s.monitor.Snapshot(ctx)
}

func (s *MySystemMonitorService) Trends(ctx context.Context, rangeValue string) (port.MonitorTrends, error) {
	if s == nil || s.monitor == nil {
		return port.MonitorTrends{Range: "1h", Available: false, Points: []port.MonitorTrendPoint{}}, nil
	}
	if rangeValue == "" {
		rangeValue = "1h"
	}
	trends, err := s.monitor.Trends(ctx, rangeValue)
	if err != nil {
		return port.MonitorTrends{}, apperrors.Invalid("system_monitor.range", "range must be 1h, 24h, or 7d")
	}
	return trends, nil
}

func (s *MySystemMonitorService) Timeline(ctx context.Context, rangeValue string) (port.MonitorTimeline, error) {
	if s == nil || s.monitor == nil {
		return port.MonitorTimeline{
			Range: "90d", Available: false, Message: "monitorServiceStopped", Periods: []port.MonitorStatusPeriod{},
		}, nil
	}
	if rangeValue == "" {
		rangeValue = "90d"
	}
	timeline, err := s.monitor.Timeline(ctx, rangeValue)
	if err != nil {
		return port.MonitorTimeline{}, apperrors.Invalid("system_monitor.timeline.range", "range must be 24h, 7d, 30d, or 90d")
	}
	return timeline, nil
}

func (s *MySystemMonitorService) Incidents(ctx context.Context, rangeValue string, limit int) ([]port.MonitorIncident, error) {
	if s == nil || s.monitor == nil {
		return []port.MonitorIncident{}, nil
	}
	if rangeValue == "" {
		rangeValue = "90d"
	}
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 200 {
		return nil, apperrors.Invalid("system_monitor.incidents.limit", "limit must be between 1 and 200")
	}
	incidents, err := s.monitor.Incidents(ctx, rangeValue, limit)
	if err != nil {
		return nil, apperrors.Invalid("system_monitor.incidents.range", "range must be 24h, 7d, 30d, or 90d")
	}
	return incidents, nil
}

func (s *MySystemMonitorService) Incident(ctx context.Context, id int64) (port.MonitorIncident, error) {
	if s == nil || s.monitor == nil {
		return port.MonitorIncident{}, apperrors.Unavailable("system_monitor.incident", nil)
	}
	if id < 1 {
		return port.MonitorIncident{}, apperrors.Invalid("system_monitor.incident.id", "incidentId must be a positive integer")
	}
	return s.monitor.Incident(ctx, id)
}

func (s *MySystemMonitorService) AddIncidentUpdate(ctx context.Context, id int64, user entity.AuthSession, content string) (port.MonitorIncidentUpdate, error) {
	if s == nil || s.monitor == nil {
		return port.MonitorIncidentUpdate{}, apperrors.Unavailable("system_monitor.incident_update", nil)
	}
	if id < 1 {
		return port.MonitorIncidentUpdate{}, apperrors.Invalid("system_monitor.incident.id", "incidentId must be a positive integer")
	}
	content = strings.TrimSpace(content)
	if content == "" || utf8.RuneCountInString(content) > 4000 {
		return port.MonitorIncidentUpdate{}, apperrors.Invalid("system_monitor.incident_update.content", "content must contain 1 to 4000 characters")
	}
	if user.UserInfoId <= 0 {
		return port.MonitorIncidentUpdate{}, apperrors.New(apperrors.KindForbidden, "system_monitor.incident_update.author", nil)
	}
	name := strings.TrimSpace(user.Nickname)
	if name == "" {
		name = strings.TrimSpace(user.Username)
	}
	if name == "" {
		name = strings.TrimSpace(user.Email)
	}
	return s.monitor.AddIncidentUpdate(ctx, id, user.UserInfoId, name, content)
}
