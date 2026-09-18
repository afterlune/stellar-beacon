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
	Date              string `json:"date"`
	Views             int64  `json:"views"`
	UniqueReaders     int64  `json:"uniqueReaders"`
	EffectiveSessions int64  `json:"effectiveSessions"`
	TotalActiveMs     int64  `json:"totalActiveMs"`
	CompletedSessions int64  `json:"completedSessions"`
}

// ContentArticleMetric is one article's aggregate for a reporting range.
type ContentArticleMetric struct {
	ArticleId         int       `json:"articleId"`
	ArticleTitle      string    `json:"articleTitle"`
	ArticleCover      string    `json:"articleCover"`
	CategoryName      string    `json:"categoryName"`
	CreateTime        time.Time `json:"createTime"`
	Views             int64     `json:"views"`
	UniqueReaders     int64     `json:"uniqueReaders"`
	EffectiveSessions int64     `json:"effectiveSessions"`
	TotalActiveMs     int64     `json:"totalActiveMs"`
	CompletedSessions int64     `json:"completedSessions"`
}

// ContentAnalyticsRepository owns the privacy-preserving article performance
// aggregates. It stores only per-article daily counters and never accepts IP,
// user-agent, referrer or account identity values.
type ContentAnalyticsRepository interface {
	RecordView(ctx context.Context, articleID int, day time.Time) error
	RecordReadSession(ctx context.Context, articleID int, day time.Time, activeMs, maxScrollPercent int, uniqueReaders int64) error
	ListDailyMetrics(ctx context.Context, startDate, endDate string) ([]ContentDailyMetric, error)
	ListArticleMetrics(ctx context.Context, startDate, endDate string) ([]ContentArticleMetric, error)
	GetArticleDailyMetrics(ctx context.Context, articleID int, startDate, endDate string) ([]ContentDailyMetric, error)
}
