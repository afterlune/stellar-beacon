package repository

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/pgsql"
	"context"

	"xorm.io/xorm"
)

var _ port.FriendLinkRepository = (*MyFriendLinkRepo)(nil)

type MyFriendLinkRepo struct{ engine *xorm.Engine }

func NewFriendLinkRepo(engine *xorm.Engine) *MyFriendLinkRepo {
	return &MyFriendLinkRepo{engine: engine}
}

func (r *MyFriendLinkRepo) ListPublic(ctx context.Context) ([]entity.TFriendLink, error) {
	session, err := repoSession(r.engine, ctx, "friend_link.public")
	if err != nil {
		return nil, err
	}
	var links []entity.TFriendLink
	if err := session.OrderBy("id DESC").Find(&links); err != nil {
		return nil, apperrors.Unavailable("friend_link.public", err)
	}
	return links, nil
}

func (r *MyFriendLinkRepo) ListAdmin(ctx context.Context, current, size int, keywords string) ([]entity.TFriendLink, int64, error) {
	session, err := repoSession(r.engine, ctx, "friend_link.admin")
	if err != nil {
		return nil, 0, err
	}
	where := ""
	args := []interface{}{}
	if keywords != "" {
		where = " WHERE link_name LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(keywords))
	}
	var count int64
	if _, err := session.SQL("SELECT count(0) FROM t_friend_link"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("friend_link.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	args = append(args, limit, offset)
	var links []entity.TFriendLink
	if err := session.SQL("SELECT * FROM t_friend_link"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", args...).Find(&links); err != nil {
		return nil, 0, apperrors.Unavailable("friend_link.admin", err)
	}
	return links, count, nil
}

func (r *MyFriendLinkRepo) SaveOrUpdate(ctx context.Context, link entity.TFriendLink) error {
	return repoTx(r.engine, ctx, "friend_link.save", func(session *xorm.Session) error {
		if link.Id == 0 {
			_, err := session.Insert(&link)
			return err
		}
		_, err := session.ID(link.Id).Update(&link)
		return err
	})
}

func (r *MyFriendLinkRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "friend_link.delete", func(session *xorm.Session) error {
		_, err := session.In("id", ids).Delete(&entity.TFriendLink{})
		return err
	})
}
