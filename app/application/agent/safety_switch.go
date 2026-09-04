package agent

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	stderrors "errors"
	"strings"
)

const agentEmergencyStopKey = "benetnasch:agent:emergency_stop"

type EmergencySwitch struct {
	cache          port.Cache
	defaultStopped bool
}

func NewEmergencySwitch(cache port.Cache, defaultStopped bool) (*EmergencySwitch, error) {
	if cache == nil {
		return nil, errors.Invalid("agent.safety_switch", "cache is required")
	}
	return &EmergencySwitch{cache: cache, defaultStopped: defaultStopped}, nil
}

func (s *EmergencySwitch) IsStopped(ctx context.Context) (bool, error) {
	if s == nil || s.cache == nil {
		return true, errors.Unavailable("agent.safety_switch.read", nil)
	}
	value, err := s.cache.Get(ctx, agentEmergencyStopKey)
	if err != nil {
		if stderrors.Is(err, port.ErrCacheMiss) {
			return s.defaultStopped, nil
		}
		return true, errors.WrapUnavailable("agent.safety_switch.read", err)
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on", "stopped":
		return true, nil
	case "0", "false", "off", "running":
		return false, nil
	default:
		// A corrupted switch value must stop the capability rather than
		// accidentally re-enable an autonomous or public entry point.
		return true, nil
	}
}

func (s *EmergencySwitch) SetStopped(ctx context.Context, stopped bool) error {
	if s == nil || s.cache == nil {
		return errors.Unavailable("agent.safety_switch.write", nil)
	}
	value := "0"
	if stopped {
		value = "1"
	}
	if err := s.cache.Set(ctx, agentEmergencyStopKey, value, 0); err != nil {
		return errors.WrapUnavailable("agent.safety_switch.write", err)
	}
	return nil
}

var _ port.AgentSafetySwitch = (*EmergencySwitch)(nil)
