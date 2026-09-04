package service

import (
	"context"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

// OperationalObservabilityService exposes only sanitized runtime aggregates
// to the authenticated administration boundary. It never performs a probe or
// returns provider request/response data.
type OperationalObservabilityService interface {
	Get(context.Context) port.ResultVO
}

type MyOperationalObservabilityService struct {
	provider port.OperationalMetricsProvider
	enabled  bool
}

func NewOperationalObservabilityService(provider port.OperationalMetricsProvider, enabled bool) OperationalObservabilityService {
	return &MyOperationalObservabilityService{provider: provider, enabled: enabled}
}

func NewDisabledOperationalObservabilityService() OperationalObservabilityService {
	return &MyOperationalObservabilityService{}
}

func (s *MyOperationalObservabilityService) Get(ctx context.Context) port.ResultVO {
	if s == nil || !s.enabled {
		return port.ResultFailWithMessage("运行时观测暂未开启")
	}
	if s.provider == nil {
		return port.ResultFromError(apperrors.Unavailable("observability.provider", nil))
	}
	return port.ResultOkWithData(s.provider.Snapshot(ctx))
}

var _ OperationalObservabilityService = (*MyOperationalObservabilityService)(nil)
