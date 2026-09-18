package repository

import (
	"context"
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"

	"xorm.io/xorm"
)

var _ port.ContentAnalyticsRepository = (*MyContentAnalyticsRepo)(nil)

type MyContentAnalyticsRepo struct {
	engine *xorm.Engine
}

func NewContentAnalyticsRepo(engine *xorm.Engine) *MyContentAnalyticsRepo {
	return &MyContentAnalyticsRepo{engine: engine}
}

func (r *MyContentAnalyticsRepo) session(ctx context.Context, op string) (*xorm.Session, error) {
	return repoSession(r.engine, ctx, op)
}

func (r *MyContentAnalyticsRepo) RecordView(ctx context.Context, articleID int, day time.Time) error {
	if articleID <= 0 {
		return nil
	}
	session, err := r.session(ctx, "content_analytics.record_view")
	if err != nil {
		return err
	}
	_, err = session.Exec(`
		INSERT INTO t_article_daily_metric (article_id, metric_date, views)
		VALUES (?, ?, 1)
		ON CONFLICT (article_id, metric_date) DO UPDATE
		SET views = t_article_daily_metric.views + 1,
		    update_time = CURRENT_TIMESTAMP
	`, articleID, day.Format("2006-01-02"))
	if err != nil {
		return apperrors.Unavailable("content_analytics.record_view", err)
	}
	return nil
}

func (r *MyContentAnalyticsRepo) RecordReadSession(ctx context.Context, articleID int, day time.Time, activeMs, maxScrollPercent int, uniqueReaders int64) error {
	if articleID <= 0 {
		return nil
	}
	session, err := r.session(ctx, "content_analytics.record_read_session")
	if err != nil {
		return err
	}
	effective := 0
	if activeMs >= 3000 {
		effective = 1
	}
	completed := 0
	if maxScrollPercent >= 90 {
		completed = 1
	}
	_, err = session.Exec(`
		INSERT INTO t_article_daily_metric (
			article_id, metric_date, unique_readers, effective_sessions,
			total_active_ms, completed_sessions
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (article_id, metric_date) DO UPDATE
		SET unique_readers = GREATEST(t_article_daily_metric.unique_readers, EXCLUDED.unique_readers),
		    effective_sessions = t_article_daily_metric.effective_sessions + EXCLUDED.effective_sessions,
		    total_active_ms = t_article_daily_metric.total_active_ms + EXCLUDED.total_active_ms,
		    completed_sessions = t_article_daily_metric.completed_sessions + EXCLUDED.completed_sessions,
		    update_time = CURRENT_TIMESTAMP
	`, articleID, day.Format("2006-01-02"), uniqueReaders, effective, activeMs, completed)
	if err != nil {
		return apperrors.Unavailable("content_analytics.record_read_session", err)
	}
	return nil
}

func (r *MyContentAnalyticsRepo) ListDailyMetrics(ctx context.Context, startDate, endDate string) ([]port.ContentDailyMetric, error) {
	session, err := r.session(ctx, "content_analytics.daily_metrics")
	if err != nil {
		return nil, err
	}
	var rows []port.ContentDailyMetric
	if err := session.SQL(`
		SELECT metric_date::text AS date,
		       COALESCE(SUM(views), 0) AS views,
		       COALESCE(SUM(unique_readers), 0) AS unique_readers,
		       COALESCE(SUM(effective_sessions), 0) AS effective_sessions,
		       COALESCE(SUM(total_active_ms), 0) AS total_active_ms,
		       COALESCE(SUM(completed_sessions), 0) AS completed_sessions
		FROM t_article_daily_metric
		WHERE metric_date >= ? AND metric_date <= ?
		GROUP BY metric_date
		ORDER BY metric_date ASC
	`, startDate, endDate).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("content_analytics.daily_metrics", err)
	}
	return rows, nil
}

func (r *MyContentAnalyticsRepo) ListArticleMetrics(ctx context.Context, startDate, endDate string) ([]port.ContentArticleMetric, error) {
	session, err := r.session(ctx, "content_analytics.article_metrics")
	if err != nil {
		return nil, err
	}
	var rows []port.ContentArticleMetric
	if err := session.SQL(`
		SELECT m.article_id AS article_id,
		       COALESCE(a.article_title, '') AS article_title,
		       COALESCE(a.article_cover, '') AS article_cover,
		       COALESCE(c.category_name, '') AS category_name,
		       COALESCE(a.create_time, to_timestamp(0)) AS create_time,
		       COALESCE(SUM(m.views), 0) AS views,
		       COALESCE(SUM(m.unique_readers), 0) AS unique_readers,
		       COALESCE(SUM(m.effective_sessions), 0) AS effective_sessions,
		       COALESCE(SUM(m.total_active_ms), 0) AS total_active_ms,
		       COALESCE(SUM(m.completed_sessions), 0) AS completed_sessions
		FROM t_article_daily_metric m
		LEFT JOIN t_article a ON a.id = m.article_id
		LEFT JOIN t_category c ON c.id = a.category_id
		WHERE m.metric_date >= ? AND m.metric_date <= ?
		GROUP BY m.article_id, a.id, c.category_name
		ORDER BY SUM(m.views) DESC, m.article_id DESC
	`, startDate, endDate).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("content_analytics.article_metrics", err)
	}
	return rows, nil
}

func (r *MyContentAnalyticsRepo) GetArticleDailyMetrics(ctx context.Context, articleID int, startDate, endDate string) ([]port.ContentDailyMetric, error) {
	if articleID <= 0 {
		return []port.ContentDailyMetric{}, nil
	}
	session, err := r.session(ctx, "content_analytics.article_daily_metrics")
	if err != nil {
		return nil, err
	}
	var rows []port.ContentDailyMetric
	if err := session.SQL(`
		SELECT metric_date::text AS date,
		       COALESCE(SUM(views), 0) AS views,
		       COALESCE(SUM(unique_readers), 0) AS unique_readers,
		       COALESCE(SUM(effective_sessions), 0) AS effective_sessions,
		       COALESCE(SUM(total_active_ms), 0) AS total_active_ms,
		       COALESCE(SUM(completed_sessions), 0) AS completed_sessions
		FROM t_article_daily_metric
		WHERE article_id = ? AND metric_date >= ? AND metric_date <= ?
		GROUP BY metric_date
		ORDER BY metric_date ASC
	`, articleID, startDate, endDate).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("content_analytics.article_daily_metrics", err)
	}
	return rows, nil
}
