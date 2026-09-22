package repository

import (
	"context"
	"fmt"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"xorm.io/xorm"
)

var _ port.CollectionReactionRepository = (*MyCollectionReactionRepo)(nil)

type MyCollectionReactionRepo struct{ engine *xorm.Engine }

func NewCollectionReactionRepo(engine *xorm.Engine) *MyCollectionReactionRepo {
	return &MyCollectionReactionRepo{engine: engine}
}

func collectionInteractionOwner(session *xorm.Session, collectionID int) (int, error) {
	var ownerID int
	found, err := session.SQL(`
		SELECT c.user_id
		FROM t_collection c
		JOIN t_user_info owner ON owner.id = c.user_id AND owner.is_disable = 0
		WHERE c.id = ? AND c.is_delete = 0 AND c.moderation_status = 'visible'
		  AND c.visibility IN ('public', 'unlisted')`, collectionID).Get(&ownerID)
	if err != nil {
		return 0, apperrors.Unavailable("collection_reaction.target", err)
	}
	if !found {
		return 0, apperrors.NotFound("collection_reaction.target")
	}
	return ownerID, nil
}

func (r *MyCollectionReactionRepo) Set(ctx context.Context, collectionID, userInfoID int, active bool) (bool, int, error) {
	if collectionID <= 0 || userInfoID <= 0 {
		return false, 0, apperrors.Invalid("collection_reaction.set", "invalid reaction")
	}
	likeCount := 0
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		ownerID, err := collectionInteractionOwner(session, collectionID)
		if err != nil {
			return err
		}
		if active {
			result, err := session.Exec(`INSERT INTO t_collection_reaction (collection_id, user_info_id, reaction)
				VALUES (?, ?, 'like') ON CONFLICT (collection_id, user_info_id) DO NOTHING`, collectionID, userInfoID)
			if err != nil {
				return apperrors.Unavailable("collection_reaction.insert", err)
			}
			inserted, err := result.RowsAffected()
			if err != nil {
				return apperrors.Unavailable("collection_reaction.insert.rows", err)
			}
			if inserted > 0 && ownerID != userInfoID {
				if err := insertInteractionNotification(session, ownerID, userInfoID, port.NotificationTypeLike,
					port.FollowContentCollection, collectionID, 0,
					fmt.Sprintf("collection_like:%d:%d", collectionID, userInfoID)); err != nil {
					return err
				}
			}
		} else if _, err := session.Exec(`DELETE FROM t_collection_reaction WHERE collection_id = ? AND user_info_id = ?`, collectionID, userInfoID); err != nil {
			return apperrors.Unavailable("collection_reaction.delete", err)
		}
		if _, err := session.SQL(`SELECT count(1) FROM t_collection_reaction WHERE collection_id = ? AND reaction = 'like'`, collectionID).Get(&likeCount); err != nil {
			return apperrors.Unavailable("collection_reaction.count", err)
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}
	return active, likeCount, nil
}

func (r *MyCollectionReactionRepo) Counts(ctx context.Context, collectionIDs []int) (map[int]int, error) {
	counts := make(map[int]int, len(collectionIDs))
	if len(collectionIDs) == 0 {
		return counts, nil
	}
	session, err := repoSession(r.engine, ctx, "collection_reaction.counts")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		CollectionID int `xorm:"collection_id"`
		Total        int `xorm:"total"`
	}
	if err := session.Table("t_collection_reaction").Select("collection_id, count(1) AS total").
		In("collection_id", collectionIDs).Where("reaction = 'like'").
		GroupBy("collection_id").Find(&rows); err != nil {
		return nil, apperrors.Unavailable("collection_reaction.counts", err)
	}
	for _, row := range rows {
		counts[row.CollectionID] = row.Total
	}
	return counts, nil
}

func (r *MyCollectionReactionRepo) States(ctx context.Context, userInfoID int, collectionIDs []int) (map[int]bool, error) {
	states := make(map[int]bool, len(collectionIDs))
	if userInfoID <= 0 || len(collectionIDs) == 0 {
		return states, nil
	}
	session, err := repoSession(r.engine, ctx, "collection_reaction.states")
	if err != nil {
		return nil, err
	}
	var ids []int
	if err := session.Table("t_collection_reaction").Cols("collection_id").
		In("collection_id", collectionIDs).Where("user_info_id = ? AND reaction = 'like'", userInfoID).Find(&ids); err != nil {
		return nil, apperrors.Unavailable("collection_reaction.states", err)
	}
	for _, id := range ids {
		states[id] = true
	}
	return states, nil
}
