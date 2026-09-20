package port

import (
	"context"
	"time"
)

type StudioJobRun struct {
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Status     int       `json:"status"`
	Message    string    `json:"message"`
}

type StudioOperationsSummary struct {
	PublishedArticles     int64         `json:"publishedArticles"`
	ScheduledArticles     int64         `json:"scheduledArticles"`
	FailedNotifications   int64         `json:"failedNotifications"`
	RetryingNotifications int64         `json:"retryingNotifications"`
	BatchOperations       int64         `json:"batchOperations"`
	LastRun               *StudioJobRun `json:"lastRun,omitempty"`
}

type StudioAnalyticsTrendPoint struct {
	Date               string `json:"date"`
	PublishedArticles  int64  `json:"publishedArticles"`
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

type StudioTopArticle struct {
	ArticleID      int     `json:"articleId"`
	Title          string  `json:"title"`
	Cover          string  `json:"cover"`
	Views          int64   `json:"views"`
	UniqueReaders  int64   `json:"uniqueReaders"`
	CompletionRate float64 `json:"completionRate"`
}

type StudioCalendarEvent struct {
	ArticleID         int        `json:"articleId"`
	Title             string     `json:"title"`
	ScheduledAt       time.Time  `json:"scheduledAt"`
	PublishedAt       *time.Time `json:"publishedAt,omitempty"`
	State             string     `json:"state"`
	NotificationState string     `json:"notificationState,omitempty"`
	LastError         string     `json:"lastError,omitempty"`
}

type StudioOperationsRepository interface {
	StudioOperationsSummary(ctx context.Context, userID int, start, end time.Time) (StudioOperationsSummary, error)
	StudioAnalyticsTrend(ctx context.Context, userID int, start, end time.Time) ([]StudioAnalyticsTrendPoint, error)
	StudioTopArticles(ctx context.Context, userID int, start, end time.Time, limit int) ([]StudioTopArticle, error)
	StudioCalendar(ctx context.Context, userID int, start, end time.Time) ([]StudioCalendarEvent, error)
}
