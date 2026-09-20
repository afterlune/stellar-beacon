package repository

import (
	"context"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"strings"
	"xorm.io/xorm"
)

var _ port.SeriesRepository = (*MySeriesRepo)(nil)

type MySeriesRepo struct{ engine *xorm.Engine }

func NewSeriesRepo(engine *xorm.Engine) *MySeriesRepo { return &MySeriesRepo{engine: engine} }

const seriesCountSQL = `SELECT s.id, s.series_name, s.series_desc, s.cover, s.update_time,
	COUNT(a.id) AS article_count
	FROM t_series s
	LEFT JOIN t_article a ON a.series_id = s.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
	WHERE s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'`

func (r *MySeriesRepo) ListPublic(ctx context.Context) ([]*port.Series, error) {
	session, err := repoSession(r.engine, ctx, "series.public")
	if err != nil {
		return nil, err
	}
	var series []*port.Series
	if err := session.SQL(seriesCountSQL + " GROUP BY s.id ORDER BY s.update_time DESC, s.id DESC").Find(&series); err != nil {
		return nil, apperrors.Unavailable("series.public", err)
	}
	return series, nil
}

func (r *MySeriesRepo) ListOptions(ctx context.Context) ([]*port.Series, error) {
	session, err := repoSession(r.engine, ctx, "series.options")
	if err != nil {
		return nil, err
	}
	var series []*entity.TSeries
	if err := session.Where("is_delete = 0").OrderBy("series_name").Find(&series); err != nil {
		return nil, apperrors.Unavailable("series.options", err)
	}
	options := make([]*port.Series, 0, len(series))
	for _, item := range series {
		options = append(options, &port.Series{Id: item.Id, SeriesName: item.SeriesName})
	}
	return options, nil
}

func (r *MySeriesRepo) ListAdmin(ctx context.Context, current, size int, keywords string) ([]*port.Series, int64, error) {
	session, err := repoSession(r.engine, ctx, "series.admin")
	if err != nil {
		return nil, 0, err
	}
	where := ""
	args := []interface{}{}
	if strings.TrimSpace(keywords) != "" {
		where = " AND s.series_name LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(keywords))
	}
	var total int64
	if _, err := session.SQL("SELECT count(0) FROM t_series s WHERE s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'"+where, args...).Get(&total); err != nil {
		return nil, 0, apperrors.Unavailable("series.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	pageArgs := append(append([]interface{}{}, args...), limit, offset)
	var series []*port.Series
	if err := session.SQL(seriesCountSQL+where+" GROUP BY s.id ORDER BY s.update_time DESC, s.id DESC LIMIT ? OFFSET ?", pageArgs...).Find(&series); err != nil {
		return nil, 0, apperrors.Unavailable("series.admin", err)
	}
	return series, total, nil
}

func (r *MySeriesRepo) Get(ctx context.Context, seriesID int) (entity.TSeries, error) {
	session, err := repoSession(r.engine, ctx, "series.get")
	if err != nil {
		return entity.TSeries{}, err
	}
	var series entity.TSeries
	found, err := session.Where("id = ? AND is_delete = 0", seriesID).Get(&series)
	if err != nil {
		return entity.TSeries{}, apperrors.Unavailable("series.get", err)
	}
	if !found {
		return entity.TSeries{}, apperrors.NotFound("series.get")
	}
	return series, nil
}

func (r *MySeriesRepo) SaveOrUpdate(ctx context.Context, series entity.TSeries) (entity.TSeries, error) {
	err := repoTx(r.engine, ctx, "series.save", func(session *xorm.Session) error {
		if series.Id == 0 {
			if _, err := session.Insert(&series); err != nil {
				return apperrors.Unavailable("series.create", err)
			}
			return nil
		}
		result, err := session.Exec(
			"UPDATE t_series SET series_name = ?, series_desc = ?, cover = ?, update_time = CURRENT_TIMESTAMP WHERE id = ? AND is_delete = 0",
			series.SeriesName, series.SeriesDesc, series.Cover, series.Id,
		)
		if err != nil {
			return apperrors.Unavailable("series.update", err)
		}
		updated, err := result.RowsAffected()
		if err != nil {
			return apperrors.Unavailable("series.update", err)
		}
		if updated == 0 {
			return apperrors.NotFound("series.update")
		}
		return nil
	})
	if err != nil {
		return entity.TSeries{}, err
	}
	return series, nil
}

// Delete soft-deletes the series and detaches its articles so the public list
// can never point at a hidden collection.
func (r *MySeriesRepo) Delete(ctx context.Context, seriesID int) error {
	return repoTx(r.engine, ctx, "series.delete", func(session *xorm.Session) error {
		if _, err := session.Exec("UPDATE t_article SET series_id = NULL, series_order = 0 WHERE series_id = ?", seriesID); err != nil {
			return apperrors.Unavailable("series.detach", err)
		}
		result, err := session.Exec("UPDATE t_series SET is_delete = 1, update_time = CURRENT_TIMESTAMP WHERE id = ? AND is_delete = 0", seriesID)
		if err != nil {
			return apperrors.Unavailable("series.delete", err)
		}
		deleted, err := result.RowsAffected()
		if err != nil {
			return apperrors.Unavailable("series.delete", err)
		}
		if deleted == 0 {
			return apperrors.NotFound("series.delete")
		}
		return nil
	})
}
