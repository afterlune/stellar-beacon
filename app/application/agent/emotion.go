package agent

import (
	"benetnasch/app/domain/port"
	"math"
	"strings"
	"time"
)

const (
	AgentEmotionToxic       = "toxic"
	AgentEmotionGentle      = "gentle"
	AgentEmotionMelancholic = "melancholic"
)

const emotionAggregationWindow = 30 * 24 * time.Hour

var emotionOrder = []string{
	AgentEmotionToxic,
	AgentEmotionGentle,
	AgentEmotionMelancholic,
}

type EmotionAggregate struct {
	Scores   map[string]float64
	Dominant string
}

// AggregateAgentEmotion converts recent, already-recorded activity into a
// normalized display signal. It has no permission or content-selection role;
// callers must keep it outside authorization, scheduling, and persistence
// decisions.
func AggregateAgentEmotion(activities []port.AgentActivity, now time.Time) EmotionAggregate {
	scores := make(map[string]float64, len(emotionOrder))
	for _, emotion := range emotionOrder {
		scores[emotion] = 0
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	seen := make(map[string]struct{}, len(activities))
	for _, activity := range activities {
		if activity.ID != "" {
			if _, exists := seen[activity.ID]; exists {
				continue
			}
			seen[activity.ID] = struct{}{}
		}
		emotion := normalizeAgentEmotion(activity.Emotion)
		if emotion == "" || activity.OccurredAt.IsZero() {
			continue
		}
		age := now.Sub(activity.OccurredAt.UTC())
		if age < 0 || age > emotionAggregationWindow {
			continue
		}
		// Recent activity matters more, but old activity decays gradually and
		// never disappears abruptly at the window boundary.
		weight := 1 / (1 + age.Hours()/(24*7))
		scores[emotion] += weight
	}
	total := 0.0
	for _, emotion := range emotionOrder {
		total += scores[emotion]
	}
	if total == 0 || math.IsNaN(total) || math.IsInf(total, 0) {
		return EmotionAggregate{Scores: scores}
	}
	for _, emotion := range emotionOrder {
		scores[emotion] /= total
	}
	dominant := ""
	maxScore := -1.0
	for _, emotion := range emotionOrder {
		if scores[emotion] > maxScore {
			maxScore = scores[emotion]
			dominant = emotion
		}
	}
	return EmotionAggregate{Scores: scores, Dominant: dominant}
}

func normalizeAgentEmotion(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AgentEmotionToxic, "毒舌":
		return AgentEmotionToxic
	case AgentEmotionGentle, "温柔":
		return AgentEmotionGentle
	case AgentEmotionMelancholic, "忧郁", "melancholy":
		return AgentEmotionMelancholic
	default:
		return ""
	}
}
