package agent

import (
	"benetnasch/app/application/support"
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	stderrors "errors"
	"strconv"
	"strings"
	"time"
)

type VitalsDeps struct {
	Site     port.SiteInfoRepository
	Cache    port.Cache
	Rhythm   port.AgentRhythm
	Activity port.AgentActivityRepository
	Now      func() time.Time
}

type BasicVitalsProvider struct {
	site     port.SiteInfoRepository
	cache    port.Cache
	rhythm   port.AgentRhythm
	activity port.AgentActivityRepository
	now      func() time.Time
}

func NewBasicVitalsProvider(deps VitalsDeps) (*BasicVitalsProvider, error) {
	if deps.Site == nil {
		return nil, errors.Invalid("agent.vitals.dependencies", "site repository is required")
	}
	if deps.Cache == nil {
		return nil, errors.Invalid("agent.vitals.dependencies", "cache is required")
	}
	return &BasicVitalsProvider{site: deps.Site, cache: deps.Cache, rhythm: deps.Rhythm, activity: deps.Activity, now: deps.Now}, nil
}

// Snapshot is intentionally deterministic and read-only. It aggregates
// already-public counters and does not expose visitor identity or raw Redis
// keys. Activity-based emotion is optional and remains a display-only signal.
func (p *BasicVitalsProvider) Snapshot(ctx context.Context) (port.AgentVitals, error) {
	if p == nil || p.site == nil || p.cache == nil {
		return port.AgentVitals{}, errors.Unavailable("agent.vitals.snapshot", nil)
	}
	now := time.Now().UTC()
	if p.now != nil {
		now = p.now().UTC()
	}
	articles, err := p.site.CountArticles(ctx)
	if err != nil {
		return port.AgentVitals{}, err
	}
	categories, err := p.site.CountCategories(ctx)
	if err != nil {
		return port.AgentVitals{}, err
	}
	tags, err := p.site.CountTags(ctx)
	if err != nil {
		return port.AgentVitals{}, err
	}
	talks, err := p.site.CountTalks(ctx)
	if err != nil {
		return port.AgentVitals{}, err
	}
	recentContent, err := p.site.CountRecentContent(ctx, now.Add(-24*time.Hour))
	if err != nil {
		return port.AgentVitals{}, err
	}
	viewCount := int64(0)
	if raw, getErr := p.cache.Get(ctx, support.BlogViewsCount); getErr == nil {
		if parsed, parseErr := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); parseErr == nil && parsed >= 0 {
			viewCount = parsed
		}
	} else if !stderrors.Is(getErr, port.ErrCacheMiss) {
		return port.AgentVitals{}, errors.WrapUnavailable("agent.vitals.views", getErr)
	}
	uniqueVisitorCount, err := countVisitorAreas(ctx, p.cache)
	if err != nil {
		return port.AgentVitals{}, err
	}
	lifeStage := "newborn"
	if articles > 0 {
		lifeStage = "growing"
	}
	snapshot := port.AgentRhythmSnapshot{}
	if p.rhythm != nil {
		snapshot = p.rhythm.Snapshot(now)
	}
	status := "awake"
	emotion := "calm"
	switch snapshot.Phase {
	case port.AgentRhythmDusk:
		status = "dusk"
		emotion = "reflective"
	case port.AgentRhythmNight:
		status = "resting"
		emotion = "quiet"
	}
	var emotionScores map[string]float64
	if p.activity != nil {
		activities, activityErr := p.activity.List(ctx, port.AgentActivityFilter{
			From:  now.Add(-emotionAggregationWindow),
			To:    now,
			Limit: 200,
		})
		if activityErr != nil {
			return port.AgentVitals{}, activityErr
		}
		aggregate := AggregateAgentEmotion(activities, now)
		emotionScores = aggregate.Scores
		if aggregate.Dominant != "" {
			emotion = aggregate.Dominant
		}
	}
	return port.AgentVitals{
		Status:             status,
		LifeStage:          lifeStage,
		Emotion:            emotion,
		EmotionScores:      emotionScores,
		Phase:              snapshot.Phase,
		Timezone:           snapshot.Timezone,
		LocalTime:          snapshot.LocalTime,
		NextTransitionAt:   snapshot.NextTransitionAt,
		ArticleCount:       articles,
		CategoryCount:      categories,
		TagCount:           tags,
		TalkCount:          talks,
		ContentCount:       articles + talks,
		RecentContentCount: recentContent,
		ViewCount:          viewCount,
		UniqueVisitorCount: uniqueVisitorCount,
		UpdatedAt:          now,
	}, nil
}

func countVisitorAreas(ctx context.Context, cache port.Cache) (int64, error) {
	areas, err := cache.HGetAll(ctx, support.VisitorArea)
	if err != nil && !stderrors.Is(err, port.ErrCacheMiss) {
		return 0, errors.WrapUnavailable("agent.vitals.visitors", err)
	}
	var total int64
	for _, raw := range areas {
		count, parseErr := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if parseErr != nil || count < 0 {
			continue
		}
		if count > (1<<63-1)-total {
			return 1<<63 - 1, nil
		}
		total += count
	}
	return total, nil
}

var _ port.AgentVitalsProvider = (*BasicVitalsProvider)(nil)
