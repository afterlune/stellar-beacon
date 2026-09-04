package repository

import (
	"benetnasch/app/domain/port"
	"strings"
	"testing"
	"time"
)

func TestNormalizeAgentActivityCanonicalizesDisplayEmotionAndMetadata(t *testing.T) {
	when := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	activity, metadata, err := normalizeAgentActivity(port.AgentActivity{
		ID:         "activity-1",
		Kind:       "agent.review.published",
		Emotion:    "温柔",
		Metadata:   map[string]string{"review_id": "review-1"},
		OccurredAt: when,
	})
	if err != nil {
		t.Fatal(err)
	}
	if activity.Emotion != "gentle" || activity.LifeStage != port.LifeStageGrowing || !strings.Contains(metadata, `"review_id":"review-1"`) {
		t.Fatalf("normalized activity=%+v metadata=%q", activity, metadata)
	}
}

func TestNormalizeAgentActivityRejectsUnknownEmotion(t *testing.T) {
	base := port.AgentActivity{ID: "activity-1", Kind: "test", Emotion: "unknown", OccurredAt: time.Now().UTC()}
	if _, _, err := normalizeAgentActivity(base); err == nil {
		t.Fatal("unknown emotion was accepted")
	}
}
