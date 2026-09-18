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

func (r *MyContentAnalyticsRepo) RecordContinuationEvent(ctx context.Context, articleID int, day time.Time, eventType port.ContinuationEventType, target *port.ContinuationTarget) error {
	if articleID <= 0 || !eventType.Valid() {
		return nil
	}
	if eventType.IsClick() && target == nil {
		return apperrors.Invalid("content_analytics.record_continuation_event", "continuation click target is required")
	}
	var seriesImpressions, seriesClicks, relatedImpressions, relatedClicks int
	switch eventType {
	case port.ContinuationEventSeriesImpression:
		seriesImpressions = 1
	case port.ContinuationEventSeriesClick:
		seriesClicks = 1
	case port.ContinuationEventRelatedImpression:
		relatedImpressions = 1
	case port.ContinuationEventRelatedClick:
		relatedClicks = 1
	}
	return repoTx(r.engine, ctx, "content_analytics.record_continuation_event", func(session *xorm.Session) error {
		if _, err := session.Exec(`
			INSERT INTO t_article_daily_metric (
				article_id, metric_date, series_impressions, series_clicks,
				related_impressions, related_clicks
			)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT (article_id, metric_date) DO UPDATE
			SET series_impressions = t_article_daily_metric.series_impressions + EXCLUDED.series_impressions,
			    series_clicks = t_article_daily_metric.series_clicks + EXCLUDED.series_clicks,
			    related_impressions = t_article_daily_metric.related_impressions + EXCLUDED.related_impressions,
			    related_clicks = t_article_daily_metric.related_clicks + EXCLUDED.related_clicks,
			    update_time = CURRENT_TIMESTAMP
		`, articleID, day.Format("2006-01-02"), seriesImpressions, seriesClicks, relatedImpressions, relatedClicks); err != nil {
			return apperrors.Unavailable("content_analytics.record_continuation_event", err)
		}
		if target == nil {
			return nil
		}
		if _, err := session.Exec(`
			INSERT INTO t_article_continuation_target (
				source_article_id, target_type, target_id, placement, position, metric_date, clicks
			)
			VALUES (?, ?, ?, ?, ?, ?, 1)
			ON CONFLICT (source_article_id, target_type, target_id, placement, position, metric_date) DO UPDATE
			SET clicks = t_article_continuation_target.clicks + EXCLUDED.clicks,
			    update_time = CURRENT_TIMESTAMP
		`, articleID, target.Type, target.Id, target.Placement, target.Position, day.Format("2006-01-02")); err != nil {
			return apperrors.Unavailable("content_analytics.record_continuation_target", err)
		}
		return nil
	})
}

func (r *MyContentAnalyticsRepo) ListContinuationTargetMetrics(ctx context.Context, startDate, endDate string, sourceArticleID int) ([]port.ContinuationTargetMetric, error) {
	session, err := r.session(ctx, "content_analytics.continuation_targets")
	if err != nil {
		return nil, err
	}
	query := `
		WITH click_rows AS (
			SELECT source_article_id, target_type, target_id, placement, position, SUM(clicks) AS clicks
			FROM t_article_continuation_target
			WHERE metric_date >= ? AND metric_date <= ?`
	args := []interface{}{startDate, endDate}
	if sourceArticleID > 0 {
		query += " AND source_article_id = ?"
		args = append(args, sourceArticleID)
	}
	query += `
			GROUP BY source_article_id, target_type, target_id, placement, position
		),
		impression_rows AS (
			SELECT article_id,
			       COALESCE(SUM(series_impressions), 0) AS series_impressions,
			       COALESCE(SUM(related_impressions), 0) AS related_impressions
			FROM t_article_daily_metric
			WHERE metric_date >= ? AND metric_date <= ?
			GROUP BY article_id
		)
		SELECT c.source_article_id AS source_article_id,
		       COALESCE(source.article_title, '') AS source_article_title,
		       c.target_type AS target_type,
		       c.target_id AS target_id,
		       CASE
		           WHEN c.target_type = 'article' THEN COALESCE(target_article.article_title, '')
		           ELSE COALESCE(target_series.series_name, '')
		       END AS target_title,
		       c.placement AS placement,
		       c.position AS position,
		       c.clicks AS clicks,
		       CASE
		           WHEN c.target_type = 'series' THEN COALESCE(i.series_impressions, 0)
		           ELSE COALESCE(i.related_impressions, 0)
		       END AS module_impressions
		FROM click_rows c
		LEFT JOIN t_article source ON source.id = c.source_article_id
		LEFT JOIN t_article target_article ON c.target_type = 'article' AND target_article.id = c.target_id
		LEFT JOIN t_series target_series ON c.target_type = 'series' AND target_series.id = c.target_id
		LEFT JOIN impression_rows i ON i.article_id = c.source_article_id
		ORDER BY c.clicks DESC, c.source_article_id DESC, c.target_id DESC`
	args = append(args, startDate, endDate)
	var rows []port.ContinuationTargetMetric
	if err := session.SQL(query, args...).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("content_analytics.continuation_targets", err)
	}
	return rows, nil
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
		       COALESCE(SUM(completed_sessions), 0) AS completed_sessions,
		       COALESCE(SUM(series_impressions), 0) AS series_impressions,
		       COALESCE(SUM(series_clicks), 0) AS series_clicks,
		       COALESCE(SUM(related_impressions), 0) AS related_impressions,
		       COALESCE(SUM(related_clicks), 0) AS related_clicks
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
		       COALESCE(SUM(m.completed_sessions), 0) AS completed_sessions,
		       COALESCE(SUM(m.series_impressions), 0) AS series_impressions,
		       COALESCE(SUM(m.series_clicks), 0) AS series_clicks,
		       COALESCE(SUM(m.related_impressions), 0) AS related_impressions,
		       COALESCE(SUM(m.related_clicks), 0) AS related_clicks
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
		       COALESCE(SUM(completed_sessions), 0) AS completed_sessions,
		       COALESCE(SUM(series_impressions), 0) AS series_impressions,
		       COALESCE(SUM(series_clicks), 0) AS series_clicks,
		       COALESCE(SUM(related_impressions), 0) AS related_impressions,
		       COALESCE(SUM(related_clicks), 0) AS related_clicks
		FROM t_article_daily_metric
		WHERE article_id = ? AND metric_date >= ? AND metric_date <= ?
		GROUP BY metric_date
		ORDER BY metric_date ASC
	`, articleID, startDate, endDate).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("content_analytics.article_daily_metrics", err)
	}
	return rows, nil
}
