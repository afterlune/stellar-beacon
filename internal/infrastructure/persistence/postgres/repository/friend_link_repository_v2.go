package repository

import (
	"context"
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"time"

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
	if err := session.Where("status = ?", port.FriendLinkStatusApproved).OrderBy("id DESC").Find(&links); err != nil {
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
		result, err := session.Exec(
			"UPDATE t_friend_link SET link_name = ?, link_avatar = ?, link_address = ?, link_intro = ?, update_time = CURRENT_TIMESTAMP WHERE id = ?",
			link.LinkName,
			link.LinkAvatar,
			link.LinkAddress,
			link.LinkIntro,
			link.Id,
		)
		if err != nil {
			return apperrors.Unavailable("friend_link.update", err)
		}
		updated, err := result.RowsAffected()
		if err != nil {
			return apperrors.Unavailable("friend_link.update", err)
		}
		if updated == 0 {
			return apperrors.NotFound("friend_link.update")
		}
		return nil
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

// CreateApplication stores a reader submission as pending and returns its id.
func (r *MyFriendLinkRepo) CreateApplication(ctx context.Context, link entity.TFriendLink) (int, error) {
	link.Status = port.FriendLinkStatusPending
	err := repoTx(r.engine, ctx, "friend_link.apply", func(session *xorm.Session) error {
		if _, err := session.Insert(&link); err != nil {
			return apperrors.Unavailable("friend_link.apply", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return link.Id, nil
}

// Review approves or rejects submissions and stamps the audit time. The admin
// acts on a handful of rows at a time, so the per-row update keeps the bound
// arguments explicit.
func (r *MyFriendLinkRepo) Review(ctx context.Context, ids []int, status int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "friend_link.review", func(session *xorm.Session) error {
		for _, id := range ids {
			if _, err := session.ID(id).MustCols("status", "audit_time").Update(&entity.TFriendLink{
				Id: id, Status: status, AuditTime: time.Now(),
			}); err != nil {
				return apperrors.Unavailable("friend_link.review", err)
			}
		}
		return nil
	})
}

// AddressExists reports whether the address is already stored in a state that
// should block a duplicate application. Rejected entries do not block.
func (r *MyFriendLinkRepo) AddressExists(ctx context.Context, address string) (bool, error) {
	session, err := repoSession(r.engine, ctx, "friend_link.exists")
	if err != nil {
		return false, err
	}
	var total int64
	if _, err := session.SQL(
		"SELECT count(0) FROM t_friend_link WHERE link_address = ? AND status <> ?",
		address, port.FriendLinkStatusRejected,
	).Get(&total); err != nil {
		return false, apperrors.Unavailable("friend_link.exists", err)
	}
	return total > 0, nil
}
