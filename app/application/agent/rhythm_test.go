package agent

import (
	"benetnasch/app/domain/port"
	"testing"
	"time"
)

func TestRhythmPolicyUsesConfiguredBoundariesAndNextTransition(t *testing.T) {
	policy, err := NewRhythmPolicy(RhythmSettings{
		Timezone:   "Asia/Shanghai",
		AwakeStart: "06:00",
		DuskStart:  "18:00",
		NightStart: "22:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name      string
		now       string
		phase     port.AgentRhythmPhase
		nextLocal string
	}{
		{name: "before awake", now: "2026-08-29T05:59:00+08:00", phase: port.AgentRhythmNight, nextLocal: "2026-08-29T06:00:00+08:00"},
		{name: "awake boundary", now: "2026-08-29T06:00:00+08:00", phase: port.AgentRhythmAwake, nextLocal: "2026-08-29T18:00:00+08:00"},
		{name: "dusk boundary", now: "2026-08-29T18:00:00+08:00", phase: port.AgentRhythmDusk, nextLocal: "2026-08-29T22:00:00+08:00"},
		{name: "night boundary", now: "2026-08-29T22:00:00+08:00", phase: port.AgentRhythmNight, nextLocal: "2026-08-30T06:00:00+08:00"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			now, parseErr := time.Parse(time.RFC3339, tt.now)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			got := policy.Snapshot(now)
			if got.Phase != tt.phase || got.Timezone != "Asia/Shanghai" {
				t.Fatalf("snapshot = %+v, want phase %q in Asia/Shanghai", got, tt.phase)
			}
			next, parseErr := time.Parse(time.RFC3339, tt.nextLocal)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if !got.NextTransitionAt.Equal(next) {
				t.Fatalf("next transition = %s, want %s", got.NextTransitionAt, next)
			}
		})
	}
}

func TestRhythmPolicyRejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewRhythmPolicy(RhythmSettings{Timezone: "Asia/Shanghai", AwakeStart: "18:00", DuskStart: "06:00", NightStart: "22:00"}); err == nil {
		t.Fatal("unordered rhythm starts were accepted")
	}
	if _, err := NewRhythmPolicy(RhythmSettings{Timezone: "not/a/timezone"}); err == nil {
		t.Fatal("invalid timezone was accepted")
	}
}
