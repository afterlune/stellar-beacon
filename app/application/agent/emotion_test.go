package agent

import (
	"benetnasch/app/domain/port"
	"context"
	"math"
	"testing"
	"time"
)

func TestAggregateAgentEmotionNormalizesRecentActivityWithStableTieBreak(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	aggregate := AggregateAgentEmotion([]port.AgentActivity{
		{ID: "gentle-new", Emotion: "温柔", OccurredAt: now},
		{ID: "gentle-new", Emotion: "gentle", OccurredAt: now},
		{ID: "toxic-old", Emotion: "毒舌", OccurredAt: now.Add(-7 * 24 * time.Hour)},
		{ID: "melancholic", Emotion: "melancholy", OccurredAt: now.Add(-60 * 24 * time.Hour)},
		{ID: "future", Emotion: "toxic", OccurredAt: now.Add(time.Hour)},
		{ID: "unknown", Emotion: "neutral", OccurredAt: now},
	}, now)
	if aggregate.Dominant != AgentEmotionGentle {
		t.Fatalf("dominant emotion = %q, want gentle; scores=%v", aggregate.Dominant, aggregate.Scores)
	}
	if len(aggregate.Scores) != 3 {
		t.Fatalf("scores = %v, want three public categories", aggregate.Scores)
	}
	var total float64
	for _, emotion := range []string{AgentEmotionToxic, AgentEmotionGentle, AgentEmotionMelancholic} {
		score, ok := aggregate.Scores[emotion]
		if !ok || score < 0 || score > 1 || math.IsNaN(score) || math.IsInf(score, 0) {
			t.Fatalf("invalid %s score: %v", emotion, score)
		}
		total += score
	}
	if math.Abs(total-1) > 0.000001 {
		t.Fatalf("normalized total = %v, want 1", total)
	}
}

type emotionActivityRepository struct {
	activities []port.AgentActivity
}

func (r emotionActivityRepository) Append(context.Context, port.AgentActivity) error { return nil }

func (r emotionActivityRepository) List(context.Context, port.AgentActivityFilter) ([]port.AgentActivity, error) {
	return r.activities, nil
}

func TestBasicVitalsUsesEmotionOnlyForDisplay(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	provider, err := NewBasicVitalsProvider(VitalsDeps{
		Site:     vitalsSiteRepository{},
		Cache:    vitalsCache{views: map[string]string{}, areas: map[string]string{}},
		Activity: emotionActivityRepository{activities: []port.AgentActivity{{ID: "1", Emotion: "toxic", OccurredAt: now}}},
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	vitals, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if vitals.Emotion != AgentEmotionToxic || vitals.EmotionScores[AgentEmotionToxic] != 1 {
		t.Fatalf("vitals emotion = %q scores=%v, want toxic display signal", vitals.Emotion, vitals.EmotionScores)
	}
	if vitals.ArticleCount != 12 || vitals.ContentCount != 17 {
		t.Fatalf("emotion aggregation changed content counters: %+v", vitals)
	}
}
