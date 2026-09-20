package repository

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"strings"

	"xorm.io/xorm"
)

var _ port.TalkRepository = (*MyTalkRepo)(nil)

type MyTalkRepo struct {
	engine *xorm.Engine
}

func NewTalkRepo(engine *xorm.Engine) *MyTalkRepo {
	return &MyTalkRepo{engine: engine}
}

func (t *MyTalkRepo) talkSession(ctx context.Context) (*xorm.Session, error) {
	return repoSession(t.engine, ctx, "talk")
}

func (t *MyTalkRepo) Count(ctx context.Context, filter port.TalkFilter) (int, error) {
	session, err := t.talkSession(ctx)
	if err != nil {
		return 0, err
	}
	query := "SELECT count(1) FROM t_talk t"
	conditions := make([]string, 0, 4)
	args := []interface{}{}
	if filter.Status != 0 {
		conditions = append(conditions, "t.status = ?")
		args = append(args, filter.Status)
	}
	if filter.ModerationStatus != "" {
		conditions = append(conditions, "t.moderation_status = ?")
		args = append(args, filter.ModerationStatus)
	}
	if filter.UserId > 0 {
		conditions = append(conditions, "t.user_id = ?")
		args = append(args, filter.UserId)
	}
	if keywords := strings.TrimSpace(filter.Keywords); keywords != "" {
		conditions = append(conditions, "t.content LIKE ? ESCAPE '\\'")
		args = append(args, pgsql.ContainsPattern(keywords))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	var count int
	if _, err := session.SQL(query, args...).Get(&count); err != nil {
		return 0, apperrors.Wrap(apperrors.KindUnavailable, "talk.count", err)
	}
	return count, nil
}

func (t *MyTalkRepo) List(ctx context.Context, current, size int) ([]*port.Talk, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := t.talkSession(ctx)
	if err != nil {
		return nil, err
	}
	var talks []*port.Talk
	if err := session.SQL(pgsql.ListTalks, limit, offset).Find(&talks); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "talk.list", err)
	}
	return talks, nil
}

func (t *MyTalkRepo) Get(ctx context.Context, id int) (port.Talk, error) {
	session, err := t.talkSession(ctx)
	if err != nil {
		return port.Talk{}, err
	}
	var talk port.Talk
	found, err := session.SQL(pgsql.GetTalkById, id).Get(&talk)
	if err != nil {
		return port.Talk{}, apperrors.Wrap(apperrors.KindUnavailable, "talk.get", err)
	}
	if !found {
		return port.Talk{}, apperrors.NotFound("talk.get")
	}
	return talk, nil
}

func (t *MyTalkRepo) ListAdmin(ctx context.Context, current, size int, filter port.TalkFilter) ([]*port.TalkAdmin, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := t.talkSession(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT t.id, t.user_id, ui.handle AS handle, ui.nickname AS nickname, ui.avatar AS avatar,
		t.content, t.images, t.is_top, t.status, t.moderation_status, t.moderation_reason,
		t.moderated_by, t.moderated_at, t.create_time
		FROM t_talk t JOIN t_user_info ui ON t.user_id = ui.id`
	conditions := make([]string, 0, 4)
	args := []interface{}{}
	if filter.Status != 0 {
		conditions = append(conditions, "t.status = ?")
		args = append(args, filter.Status)
	}
	if filter.ModerationStatus != "" {
		conditions = append(conditions, "t.moderation_status = ?")
		args = append(args, filter.ModerationStatus)
	}
	if filter.UserId > 0 {
		conditions = append(conditions, "t.user_id = ?")
		args = append(args, filter.UserId)
	}
	if keywords := strings.TrimSpace(filter.Keywords); keywords != "" {
		conditions = append(conditions, "t.content LIKE ? ESCAPE '\\'")
		args = append(args, pgsql.ContainsPattern(keywords))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY t.is_top DESC, t.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var talks []*port.TalkAdmin
	if err := session.SQL(query, args...).Find(&talks); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "talk.list_admin", err)
	}
	return talks, nil
}

func (t *MyTalkRepo) GetAdmin(ctx context.Context, id int) (port.TalkAdmin, error) {
	session, err := t.talkSession(ctx)
	if err != nil {
		return port.TalkAdmin{}, err
	}
	var talk port.TalkAdmin
	found, err := session.SQL(pgsql.GetTalkByIdAdmin, id).Get(&talk)
	if err != nil {
		return port.TalkAdmin{}, apperrors.Wrap(apperrors.KindUnavailable, "talk.get_admin", err)
	}
	if !found {
		return port.TalkAdmin{}, apperrors.NotFound("talk.get_admin")
	}
	return talk, nil
}

func (t *MyTalkRepo) SaveOrUpdate(ctx context.Context, talk entity.TTalk) error {
	return ormInit.WithEngineTx(t.engine, ctx, func(session *xorm.Session) error {
		if talk.Id != 0 {
			result, err := session.Exec(
				"UPDATE t_talk SET content = ?, images = ?, is_top = ?, status = ?, update_time = CURRENT_TIMESTAMP WHERE id = ?",
				talk.Content,
				talk.Images,
				talk.IsTop,
				talk.Status,
				talk.Id,
			)
			if err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "talk.update", err)
			}
			updated, err := result.RowsAffected()
			if err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "talk.update", err)
			}
			if updated == 0 {
				return apperrors.NotFound("talk.update")
			}
			return nil
		}
		if _, err := session.Insert(&talk); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "talk.create", err)
		}
		return nil
	})
}

func (t *MyTalkRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	session, err := t.talkSession(ctx)
	if err != nil {
		return err
	}
	if _, err := session.In("id", ids).Delete(&entity.TTalk{}); err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "talk.delete", err)
	}
	return nil
}
