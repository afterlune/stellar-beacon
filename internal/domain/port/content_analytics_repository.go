package port

import (
	"context"
	"time"
)

// ContentDailyMetric is the site-wide daily aggregate used by the content
// performance dashboard. UniqueReaders is a database fallback; the
// application overrides it from the daily Redis HyperLogLog when available so
// readers are deduplicated across days.
type ContentDailyMetric struct {
	Date               string `json:"date"`
	Views              int64  `json:"views"`
	UniqueReaders      int64  `json:"uniqueReaders"`
	EffectiveSessions  int64  `json:"effectiveSessions"`
	TotalActiveMs      int64  `json:"totalActiveMs"`
	CompletedSessions  int64  `json:"completedSessions"`
	SeriesImpressions  int64  `json:"seriesImpressions"`
	SeriesClicks       int64  `json:"seriesClicks"`
	RelatedImpressions int64  `json:"relatedImpressions"`
	RelatedClicks      int64  `json:"relatedClicks"`
}

// ContentArticleMetric is one article's aggregate for a reporting range.
type ContentArticleMetric struct {
	ArticleId          int       `json:"articleId"`
	ArticleTitle       string    `json:"articleTitle"`
	ArticleCover       string    `json:"articleCover"`
	CategoryName       string    `json:"categoryName"`
	CreateTime         time.Time `json:"createTime"`
	Views              int64     `json:"views"`
	UniqueReaders      int64     `json:"uniqueReaders"`
	EffectiveSessions  int64     `json:"effectiveSessions"`
	TotalActiveMs      int64     `json:"totalActiveMs"`
	CompletedSessions  int64     `json:"completedSessions"`
	SeriesImpressions  int64     `json:"seriesImpressions"`
	SeriesClicks       int64     `json:"seriesClicks"`
	RelatedImpressions int64     `json:"relatedImpressions"`
	RelatedClicks      int64     `json:"relatedClicks"`
}

type ContinuationTargetType string

const (
	ContinuationTargetArticle ContinuationTargetType = "article"
	ContinuationTargetSeries  ContinuationTargetType = "series"
)

func (targetType ContinuationTargetType) Valid() bool {
	return targetType == ContinuationTargetArticle || targetType == ContinuationTargetSeries
}

type ContinuationPlacement string

const (
	ContinuationPlacementRelated        ContinuationPlacement = "related"
	ContinuationPlacementSeriesPrevious ContinuationPlacement = "series_previous"
	ContinuationPlacementSeriesNext     ContinuationPlacement = "series_next"
	ContinuationPlacementSeriesIndex    ContinuationPlacement = "series_index"
)

type ContinuationTarget struct {
	Type      ContinuationTargetType
	Id        int
	Placement ContinuationPlacement
	Position  int
}

type ContinuationTargetMetric struct {
	SourceArticleId    int                    `json:"sourceArticleId"`
	SourceArticleTitle string                 `json:"sourceArticleTitle"`
	TargetType         ContinuationTargetType `json:"targetType"`
	TargetId           int                    `json:"targetId"`
	TargetTitle        string                 `json:"targetTitle"`
	Placement          ContinuationPlacement  `json:"placement"`
	Position           int                    `json:"position"`
	Clicks             int64                  `json:"clicks"`
	ModuleImpressions  int64                  `json:"moduleImpressions"`
}
type ContinuationEventType string

const (
	ContinuationEventSeriesImpression  ContinuationEventType = "series_impression"
	ContinuationEventSeriesClick       ContinuationEventType = "series_click"
	ContinuationEventRelatedImpression ContinuationEventType = "related_impression"
	ContinuationEventRelatedClick      ContinuationEventType = "related_click"
)

func (eventType ContinuationEventType) Valid() bool {
	switch eventType {
	case ContinuationEventSeriesImpression, ContinuationEventSeriesClick, ContinuationEventRelatedImpression, ContinuationEventRelatedClick:
		return true
	default:
		return false
	}
}

func (eventType ContinuationEventType) IsClick() bool {
	return eventType == ContinuationEventSeriesClick || eventType == ContinuationEventRelatedClick
}

func (eventType ContinuationEventType) IsImpression() bool {
	return eventType == ContinuationEventSeriesImpression || eventType == ContinuationEventRelatedImpression
}

// ContentAnalyticsRepository owns the privacy-preserving article performance
// aggregates. It stores only per-article daily counters and never accepts IP,
// user-agent, referrer or account identity values.
type ContentAnalyticsRepository interface {
	RecordView(ctx context.Context, articleID int, day time.Time) error
	RecordReadSession(ctx context.Context, articleID int, day time.Time, activeMs, maxScrollPercent int, uniqueReaders int64) error
	RecordContinuationEvent(ctx context.Context, articleID int, day time.Time, eventType ContinuationEventType, target *ContinuationTarget) error
	ListContinuationTargetMetrics(ctx context.Context, startDate, endDate string, sourceArticleID int) ([]ContinuationTargetMetric, error)
	ListDailyMetrics(ctx context.Context, startDate, endDate string) ([]ContentDailyMetric, error)
	ListArticleMetrics(ctx context.Context, startDate, endDate string) ([]ContentArticleMetric, error)
	GetArticleDailyMetrics(ctx context.Context, articleID int, startDate, endDate string) ([]ContentDailyMetric, error)
}
