package repository

import (
	"context"
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"xorm.io/xorm"
)

var _ port.StudioOperationsRepository = (*MyStudioOperationsRepo)(nil)

type MyStudioOperationsRepo struct{ engine *xorm.Engine }

func NewStudioOperationsRepo(engine *xorm.Engine) *MyStudioOperationsRepo {
	return &MyStudioOperationsRepo{engine: engine}
}

func (r *MyStudioOperationsRepo) StudioOperationsSummary(ctx context.Context, userID int, start, end time.Time) (port.StudioOperationsSummary, error) {
	session, err := repoSession(r.engine, ctx, "studio.operations.summary")
	if err != nil {
		return port.StudioOperationsSummary{}, err
	}
	var summary port.StudioOperationsSummary
	queries := []struct {
		sql    string
		args   []interface{}
		target *int64
	}{
		{`SELECT count(1) FROM t_article_publish_record WHERE user_id = ? AND published_at >= ? AND published_at < ?`, []interface{}{userID, start, end}, &summary.PublishedArticles},
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 4 AND scheduled_at IS NOT NULL`, []interface{}{userID}, &summary.ScheduledArticles},
		{`SELECT count(1) FROM t_article_publish_record WHERE user_id = ? AND notification_state = 'failed'`, []interface{}{userID}, &summary.FailedNotifications},
		{`SELECT count(1) FROM t_article_publish_record WHERE user_id = ? AND notification_state = 'failed' AND next_retry_at IS NOT NULL`, []interface{}{userID}, &summary.RetryingNotifications},
		{`SELECT count(1) FROM t_content_operation_audit WHERE operator_id = ? AND operation IN ('batch_status', 'batch_delete') AND created_at >= ? AND created_at < ?`, []interface{}{userID, start, end}, &summary.BatchOperations},
	}
	for _, item := range queries {
		if _, err := session.SQL(item.sql, item.args...).Get(item.target); err != nil {
			return port.StudioOperationsSummary{}, apperrors.Unavailable("studio.operations.summary", err)
		}
	}
	var run struct {
		StartTime time.Time `xorm:"start_time"`
		EndTime   time.Time `xorm:"end_time"`
		Status    int       `xorm:"status"`
		Message   string    `xorm:"job_message"`
	}
	found, err := session.SQL(`SELECT start_time, end_time, status, job_message FROM t_job_log WHERE invoke_target = 'article.publishScheduled' ORDER BY id DESC LIMIT 1`).Get(&run)
	if err != nil {
		return port.StudioOperationsSummary{}, apperrors.Unavailable("studio.operations.latest_job", err)
	}
	if found {
		summary.LastRun = &port.StudioJobRun{StartedAt: run.StartTime, FinishedAt: run.EndTime, Status: run.Status, Message: run.Message}
	}
	return summary, nil
}

func (r *MyStudioOperationsRepo) StudioAnalyticsTrend(ctx context.Context, userID int, start, end time.Time) ([]port.StudioAnalyticsTrendPoint, error) {
	session, err := repoSession(r.engine, ctx, "studio.operations.trend")
	if err != nil {
		return nil, err
	}
	type metricRow struct {
		Date               time.Time `xorm:"metric_date"`
		Views              int64     `xorm:"views"`
		UniqueReaders      int64     `xorm:"unique_readers"`
		EffectiveSessions  int64     `xorm:"effective_sessions"`
		TotalActiveMs      int64     `xorm:"total_active_ms"`
		CompletedSessions  int64     `xorm:"completed_sessions"`
		SeriesImpressions  int64     `xorm:"series_impressions"`
		SeriesClicks       int64     `xorm:"series_clicks"`
		RelatedImpressions int64     `xorm:"related_impressions"`
		RelatedClicks      int64     `xorm:"related_clicks"`
	}
	var metrics []metricRow
	if err := session.SQL(`
		SELECT m.metric_date,
		       COALESCE(SUM(m.views), 0) AS views,
		       COALESCE(SUM(m.unique_readers), 0) AS unique_readers,
		       COALESCE(SUM(m.effective_sessions), 0) AS effective_sessions,
		       COALESCE(SUM(m.total_active_ms), 0) AS total_active_ms,
		       COALESCE(SUM(m.completed_sessions), 0) AS completed_sessions,
		       COALESCE(SUM(m.series_impressions), 0) AS series_impressions,
		       COALESCE(SUM(m.series_clicks), 0) AS series_clicks,
		       COALESCE(SUM(m.related_impressions), 0) AS related_impressions,
		       COALESCE(SUM(m.related_clicks), 0) AS related_clicks
		FROM t_article_daily_metric m
		JOIN t_article a ON a.id = m.article_id
		WHERE a.user_id = ? AND m.metric_date >= ?::date AND m.metric_date <= ?::date
		GROUP BY m.metric_date
	`, userID, start.Format("2006-01-02"), end.Format("2006-01-02")).Find(&metrics); err != nil {
		return nil, apperrors.Unavailable("studio.operations.trend.metrics", err)
	}
	type publishRow struct {
		Date  time.Time `xorm:"published_day"`
		Count int64     `xorm:"published_count"`
	}
	var publishes []publishRow
	if err := session.SQL(`
		SELECT published_at::date AS published_day, count(1) AS published_count
		FROM t_article_publish_record
		WHERE user_id = ? AND published_at >= ? AND published_at < ?
		GROUP BY published_at::date
	`, userID, start, end.AddDate(0, 0, 1)).Find(&publishes); err != nil {
		return nil, apperrors.Unavailable("studio.operations.trend.publishes", err)
	}
	metricsByDate := make(map[string]metricRow, len(metrics))
	for _, row := range metrics {
		metricsByDate[row.Date.Format("2006-01-02")] = row
	}
	publishesByDate := make(map[string]int64, len(publishes))
	for _, row := range publishes {
		publishesByDate[row.Date.Format("2006-01-02")] = row.Count
	}
	result := make([]port.StudioAnalyticsTrendPoint, 0, int(end.Sub(start).Hours()/24)+1)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		metric := metricsByDate[key]
		result = append(result, port.StudioAnalyticsTrendPoint{
			Date: key, PublishedArticles: publishesByDate[key], Views: metric.Views, UniqueReaders: metric.UniqueReaders,
			EffectiveSessions: metric.EffectiveSessions, TotalActiveMs: metric.TotalActiveMs, CompletedSessions: metric.CompletedSessions,
			SeriesImpressions: metric.SeriesImpressions, SeriesClicks: metric.SeriesClicks, RelatedImpressions: metric.RelatedImpressions, RelatedClicks: metric.RelatedClicks,
		})
	}
	return result, nil
}

func (r *MyStudioOperationsRepo) StudioTopArticles(ctx context.Context, userID int, start, end time.Time, limit int) ([]port.StudioTopArticle, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	session, err := repoSession(r.engine, ctx, "studio.operations.top_articles")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ArticleID         int    `xorm:"article_id"`
		Title             string `xorm:"title"`
		Cover             string `xorm:"cover"`
		Views             int64  `xorm:"views"`
		UniqueReaders     int64  `xorm:"unique_readers"`
		EffectiveSessions int64  `xorm:"effective_sessions"`
		CompletedSessions int64  `xorm:"completed_sessions"`
	}
	if err := session.SQL(`
		SELECT a.id AS article_id, a.article_title AS title, COALESCE(a.article_cover, '') AS cover,
		       COALESCE(SUM(m.views), 0) AS views, COALESCE(SUM(m.unique_readers), 0) AS unique_readers,
		       COALESCE(SUM(m.effective_sessions), 0) AS effective_sessions,
		       COALESCE(SUM(m.completed_sessions), 0) AS completed_sessions
		FROM t_article a
		JOIN t_article_daily_metric m ON m.article_id = a.id
		WHERE a.user_id = ? AND a.is_delete = 0 AND m.metric_date >= ?::date AND m.metric_date <= ?::date
		GROUP BY a.id, a.article_title, a.article_cover
		ORDER BY SUM(m.views) DESC, a.id DESC
		LIMIT ?
	`, userID, start.Format("2006-01-02"), end.Format("2006-01-02"), limit).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("studio.operations.top_articles", err)
	}
	result := make([]port.StudioTopArticle, 0, len(rows))
	for _, row := range rows {
		rate := float64(0)
		if row.EffectiveSessions > 0 {
			rate = float64(row.CompletedSessions) * 100 / float64(row.EffectiveSessions)
		}
		result = append(result, port.StudioTopArticle{ArticleID: row.ArticleID, Title: row.Title, Cover: row.Cover, Views: row.Views, UniqueReaders: row.UniqueReaders, CompletionRate: rate})
	}
	return result, nil
}

func (r *MyStudioOperationsRepo) StudioCalendar(ctx context.Context, userID int, start, end time.Time) ([]port.StudioCalendarEvent, error) {
	session, err := repoSession(r.engine, ctx, "studio.operations.calendar")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ArticleID         int        `xorm:"article_id"`
		Title             string     `xorm:"title"`
		ScheduledAt       time.Time  `xorm:"scheduled_at"`
		PublishedAt       *time.Time `xorm:"published_at"`
		State             string     `xorm:"state"`
		NotificationState string     `xorm:"notification_state"`
		LastError         string     `xorm:"last_error"`
	}
	if err := session.SQL(`
		SELECT a.id AS article_id, a.article_title AS title, a.scheduled_at,
		       r.published_at, r.notification_state, COALESCE(r.last_error, '') AS last_error,
		       CASE
		         WHEN r.notification_state = 'suppressed' THEN 'suppressed'
		         WHEN r.notification_state = 'failed' THEN 'notification_failed'
		         WHEN r.id IS NOT NULL THEN 'published'
		         WHEN a.status = 4 AND a.scheduled_at < CURRENT_TIMESTAMP THEN 'overdue'
		         ELSE 'scheduled'
		       END AS state
		FROM t_article a
		LEFT JOIN LATERAL (
			SELECT id, published_at, notification_state, last_error
			FROM t_article_publish_record
			WHERE article_id = a.id
			ORDER BY published_at DESC, id DESC
			LIMIT 1
		) r ON TRUE
		WHERE a.user_id = ? AND a.is_delete = 0 AND a.scheduled_at >= ? AND a.scheduled_at < ?
		ORDER BY COALESCE(r.published_at, a.scheduled_at) ASC, a.id ASC
	`, userID, start, end).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("studio.operations.calendar", err)
	}
	result := make([]port.StudioCalendarEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, port.StudioCalendarEvent{
			ArticleID: row.ArticleID, Title: row.Title, ScheduledAt: row.ScheduledAt, PublishedAt: row.PublishedAt,
			State: row.State, NotificationState: row.NotificationState, LastError: row.LastError,
		})
	}
	return result, nil
}
