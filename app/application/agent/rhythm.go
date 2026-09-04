package agent

import (
	"benetnasch/app/domain/port"
	"fmt"
	"strings"
	"time"
)

const (
	defaultAgentRhythmTimezone = "Asia/Shanghai"
	defaultAgentAwakeStart     = "06:00"
	defaultAgentDuskStart      = "18:00"
	defaultAgentNightStart     = "22:00"
)

// RhythmSettings is the reviewed, human-readable configuration for the
// deterministic day/night cycle. The model never supplies or interprets
// these values.
type RhythmSettings struct {
	Timezone   string
	AwakeStart string
	DuskStart  string
	NightStart string
}

func DefaultRhythmSettings() RhythmSettings {
	return RhythmSettings{
		Timezone:   defaultAgentRhythmTimezone,
		AwakeStart: defaultAgentAwakeStart,
		DuskStart:  defaultAgentDuskStart,
		NightStart: defaultAgentNightStart,
	}
}

// RhythmPolicy is immutable after construction and therefore safe to share
// between the public chat service and the public vitals provider.
type RhythmPolicy struct {
	location   *time.Location
	timezone   string
	awakeStart time.Duration
	duskStart  time.Duration
	nightStart time.Duration
}

func NewRhythmPolicy(settings RhythmSettings) (*RhythmPolicy, error) {
	defaults := DefaultRhythmSettings()
	if strings.TrimSpace(settings.Timezone) == "" {
		settings.Timezone = defaults.Timezone
	}
	if strings.TrimSpace(settings.AwakeStart) == "" {
		settings.AwakeStart = defaults.AwakeStart
	}
	if strings.TrimSpace(settings.DuskStart) == "" {
		settings.DuskStart = defaults.DuskStart
	}
	if strings.TrimSpace(settings.NightStart) == "" {
		settings.NightStart = defaults.NightStart
	}
	location, err := time.LoadLocation(strings.TrimSpace(settings.Timezone))
	if err != nil {
		return nil, fmt.Errorf("load agent rhythm timezone %q: %w", settings.Timezone, err)
	}
	awake, err := parseClockTime(settings.AwakeStart)
	if err != nil {
		return nil, fmt.Errorf("parse agent awake start: %w", err)
	}
	dusk, err := parseClockTime(settings.DuskStart)
	if err != nil {
		return nil, fmt.Errorf("parse agent dusk start: %w", err)
	}
	night, err := parseClockTime(settings.NightStart)
	if err != nil {
		return nil, fmt.Errorf("parse agent night start: %w", err)
	}
	if awake >= dusk || dusk >= night {
		return nil, fmt.Errorf("agent rhythm starts must be ordered awake < dusk < night")
	}
	return &RhythmPolicy{
		location:   location,
		timezone:   location.String(),
		awakeStart: awake,
		duskStart:  dusk,
		nightStart: night,
	}, nil
}

func DefaultRhythmPolicy() *RhythmPolicy {
	policy, err := NewRhythmPolicy(DefaultRhythmSettings())
	if err != nil {
		// The built-in defaults are constants and validated by tests. Keep a
		// fixed-zone fallback so a stripped tzdata package cannot disable the
		// whole public API at construction time.
		return &RhythmPolicy{
			location:   time.FixedZone(defaultAgentRhythmTimezone, 8*60*60),
			timezone:   defaultAgentRhythmTimezone,
			awakeStart: 6 * time.Hour,
			duskStart:  18 * time.Hour,
			nightStart: 22 * time.Hour,
		}
	}
	return policy
}

func parseClockTime(value string) (time.Duration, error) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("clock value %q must use HH:MM", value)
	}
	return time.Duration(parsed.Hour())*time.Hour + time.Duration(parsed.Minute())*time.Minute, nil
}

func (p *RhythmPolicy) Snapshot(now time.Time) port.AgentRhythmSnapshot {
	if p == nil || p.location == nil {
		return port.AgentRhythmSnapshot{}
	}
	local := now.In(p.location)
	minute := time.Duration(local.Hour())*time.Hour + time.Duration(local.Minute())*time.Minute
	phase := port.AgentRhythmAwake
	transition := p.duskStart
	switch {
	case minute >= p.nightStart || minute < p.awakeStart:
		phase = port.AgentRhythmNight
		transition = p.awakeStart
	case minute >= p.duskStart:
		phase = port.AgentRhythmDusk
		transition = p.nightStart
	}
	nextDate := local
	if phase == port.AgentRhythmNight && minute >= p.nightStart {
		nextDate = nextDate.AddDate(0, 0, 1)
	}
	next := time.Date(nextDate.Year(), nextDate.Month(), nextDate.Day(), 0, 0, 0, 0, p.location).Add(transition)
	return port.AgentRhythmSnapshot{
		Phase:            phase,
		Timezone:         p.timezone,
		LocalTime:        local.Format(time.RFC3339),
		NextTransitionAt: next,
	}
}

var _ port.AgentRhythm = (*RhythmPolicy)(nil)
