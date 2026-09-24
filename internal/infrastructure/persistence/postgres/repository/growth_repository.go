package repository

import (
	"context"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"xorm.io/xorm"
)

var _ port.GrowthRepository = (*MyGrowthRepo)(nil)

type MyGrowthRepo struct{ engine *xorm.Engine }

func NewGrowthRepo(engine *xorm.Engine) *MyGrowthRepo { return &MyGrowthRepo{engine: engine} }

func (r *MyGrowthRepo) RecordEvent(ctx context.Context, event entity.TGrowthEvent) error {
	session, err := repoSession(r.engine, ctx, "growth.record")
	if err != nil {
		return err
	}
	var insertErr error
	if event.ArticleId > 0 {
		_, insertErr = session.Exec(`
			INSERT INTO t_growth_event (event_name, article_id, path, created_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		`, event.EventName, event.ArticleId, event.Path)
	} else {
		_, insertErr = session.Exec(`
			INSERT INTO t_growth_event (event_name, path, created_at)
			VALUES (?, ?, CURRENT_TIMESTAMP)
		`, event.EventName, event.Path)
	}
	if insertErr != nil {
		return apperrors.Unavailable("growth.record", insertErr)
	}
	return nil
}

func (r *MyGrowthRepo) Summary(ctx context.Context, since time.Time) ([]port.GrowthSummary, error) {
	session, err := repoSession(r.engine, ctx, "growth.summary")
	if err != nil {
		return nil, err
	}
	var result []port.GrowthSummary
	if err := session.SQL(`
		SELECT event_name, to_char(created_at, 'YYYY-MM-DD') AS day,
		       COUNT(*) AS count, MAX(created_at) AS created_at
		FROM t_growth_event
		WHERE created_at >= ?
		GROUP BY event_name, day
		ORDER BY day ASC, event_name ASC
	`, since).Find(&result); err != nil {
		return nil, apperrors.Unavailable("growth.summary", err)
	}
	return result, nil
}

func (r *MyGrowthRepo) SummaryByPeriod(ctx context.Context, since time.Time, unit string) ([]port.GrowthTrend, error) {
	session, err := repoSession(r.engine, ctx, "growth.summary_period")
	if err != nil {
		return nil, err
	}
	format := "YYYY-MM-DD"
	if unit == "month" {
		format = "YYYY-MM"
	}
	var rows []struct {
		Period    string
		EventName string
		Count     int64
	}
	query := `
		SELECT to_char(created_at, '` + format + `') AS period, event_name, COUNT(*) AS count
		FROM t_growth_event
		WHERE created_at >= ?
		GROUP BY period, event_name
		ORDER BY period ASC, event_name ASC`
	if err := session.SQL(query, since).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("growth.summary_period", err)
	}
	result := make([]port.GrowthTrend, 0, len(rows))
	byPeriod := make(map[string]int)
	for _, row := range rows {
		index, ok := byPeriod[row.Period]
		if !ok {
			result = append(result, port.GrowthTrend{Period: row.Period})
			index = len(result) - 1
			byPeriod[row.Period] = index
		}
		switch row.EventName {
		case "share_click":
			result[index].ShareClicks = row.Count
		case "subscribe_start":
			result[index].SubscribeStarts = row.Count
		case "subscribe_confirm":
			result[index].SubscribeConfirms = row.Count
		case "unsubscribe":
			result[index].Unsubscribes = row.Count
		}
	}
	return result, nil
}

func (r *MyGrowthRepo) StudioActivationFunnel(ctx context.Context, since time.Time) (port.StudioActivationFunnel, error) {
	session, err := repoSession(r.engine, ctx, "growth.studio_activation_funnel")
	if err != nil {
		return port.StudioActivationFunnel{}, err
	}
	var result port.StudioActivationFunnel
	if _, err := session.SQL(`
		SELECT COUNT(*) AS started,
		       COUNT(identity_completed_at) AS identity_completed,
		       COUNT(content_completed_at) AS content_completed,
		       COUNT(profile_visited_at) AS profile_visited,
		       COUNT(completed_at) AS completed
		FROM t_studio_activation
		WHERE started_at IS NOT NULL AND started_at >= ?
	`, since).Get(&result); err != nil {
		return port.StudioActivationFunnel{}, apperrors.Unavailable("growth.studio_activation_funnel", err)
	}
	return result, nil
}

func (r *MyGrowthRepo) Cleanup(ctx context.Context, before time.Time) error {
	session, err := repoSession(r.engine, ctx, "growth.cleanup")
	if err != nil {
		return err
	}
	if _, err := session.Exec("DELETE FROM t_growth_event WHERE created_at < ?", before); err != nil {
		return apperrors.Unavailable("growth.cleanup", err)
	}
	return nil
}
