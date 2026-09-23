package repository

import (
	"context"
	"fmt"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	pgsql "github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
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

func (r *MyCollectionReactionRepo) Set(ctx context.Context, collectionID, userInfoID int, reaction string, active bool) (bool, port.CollectionReactionCounts, error) {
	if collectionID <= 0 || userInfoID <= 0 || !port.IsReactionKind(reaction) {
		return false, port.CollectionReactionCounts{}, apperrors.Invalid("collection_reaction.set", "invalid reaction")
	}
	counts := port.CollectionReactionCounts{}
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		ownerID, err := collectionInteractionOwner(session, collectionID)
		if err != nil {
			return err
		}
		if active {
			result, err := session.Exec(`INSERT INTO t_collection_reaction (collection_id, user_info_id, reaction)
				VALUES (?, ?, ?) ON CONFLICT (collection_id, user_info_id, reaction) DO NOTHING`, collectionID, userInfoID, reaction)
			if err != nil {
				return apperrors.Unavailable("collection_reaction.insert", err)
			}
			inserted, err := result.RowsAffected()
			if err != nil {
				return apperrors.Unavailable("collection_reaction.insert.rows", err)
			}
			if inserted > 0 && ownerID != userInfoID {
				if err := insertInteractionNotification(session, ownerID, userInfoID, reaction,
					port.FollowContentCollection, collectionID, 0,
					fmt.Sprintf("collection_%s:%d:%d", reaction, collectionID, userInfoID)); err != nil {
					return err
				}
			}
		} else if _, err := session.Exec(`DELETE FROM t_collection_reaction
			WHERE collection_id = ? AND user_info_id = ? AND reaction = ?`, collectionID, userInfoID, reaction); err != nil {
			return apperrors.Unavailable("collection_reaction.delete", err)
		}
		var row struct {
			LikeCount     int `xorm:"like_count"`
			FavoriteCount int `xorm:"favorite_count"`
		}
		if _, err := session.SQL(`SELECT
				count(1) FILTER (WHERE reaction = 'like') AS like_count,
				count(1) FILTER (WHERE reaction = 'favorite') AS favorite_count
			FROM t_collection_reaction WHERE collection_id = ?`, collectionID).Get(&row); err != nil {
			return apperrors.Unavailable("collection_reaction.count", err)
		}
		counts = port.CollectionReactionCounts{LikeCount: row.LikeCount, FavoriteCount: row.FavoriteCount}
		return nil
	})
	if err != nil {
		return false, port.CollectionReactionCounts{}, err
	}
	return active, counts, nil
}

func (r *MyCollectionReactionRepo) Counts(ctx context.Context, collectionIDs []int) (map[int]port.CollectionReactionCounts, error) {
	counts := make(map[int]port.CollectionReactionCounts, len(collectionIDs))
	if len(collectionIDs) == 0 {
		return counts, nil
	}
	session, err := repoSession(r.engine, ctx, "collection_reaction.counts")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		CollectionID  int `xorm:"collection_id"`
		LikeCount     int `xorm:"like_count"`
		FavoriteCount int `xorm:"favorite_count"`
	}
	if err := session.Table("t_collection_reaction").
		Select("collection_id, count(1) FILTER (WHERE reaction = 'like') AS like_count, count(1) FILTER (WHERE reaction = 'favorite') AS favorite_count").
		In("collection_id", collectionIDs).GroupBy("collection_id").Find(&rows); err != nil {
		return nil, apperrors.Unavailable("collection_reaction.counts", err)
	}
	for _, row := range rows {
		counts[row.CollectionID] = port.CollectionReactionCounts{LikeCount: row.LikeCount, FavoriteCount: row.FavoriteCount}
	}
	return counts, nil
}

func (r *MyCollectionReactionRepo) States(ctx context.Context, userInfoID int, collectionIDs []int) (map[int]port.CollectionReactionState, error) {
	states := make(map[int]port.CollectionReactionState, len(collectionIDs))
	if userInfoID <= 0 || len(collectionIDs) == 0 {
		return states, nil
	}
	session, err := repoSession(r.engine, ctx, "collection_reaction.states")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		CollectionID int    `xorm:"collection_id"`
		Reaction     string `xorm:"reaction"`
	}
	if err := session.Table("t_collection_reaction").Cols("collection_id", "reaction").
		In("collection_id", collectionIDs).Where("user_info_id = ?", userInfoID).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("collection_reaction.states", err)
	}
	for _, row := range rows {
		state := states[row.CollectionID]
		state.Like = state.Like || row.Reaction == port.ReactionLike
		state.Favorite = state.Favorite || row.Reaction == port.ReactionFavorite
		states[row.CollectionID] = state
	}
	return states, nil
}

func (r *MyCollectionReactionRepo) ListFavoriteCollectionIDsByUser(ctx context.Context, userInfoID, current, size int) ([]int, int, error) {
	if userInfoID <= 0 {
		return nil, 0, apperrors.Invalid("collection_reaction.favorites", "invalid user")
	}
	session, err := repoSession(r.engine, ctx, "collection_reaction.favorites")
	if err != nil {
		return nil, 0, err
	}
	var total int
	if _, err := session.SQL(`SELECT count(1) FROM t_collection_reaction reaction
		JOIN t_collection c ON c.id = reaction.collection_id AND c.is_delete = 0
		  AND c.visibility IN ('public', 'unlisted') AND c.moderation_status = 'visible'
		JOIN t_user_info owner ON owner.id = c.user_id AND owner.is_disable = 0
		WHERE reaction.user_info_id = ? AND reaction.reaction = 'favorite'`, userInfoID).Get(&total); err != nil {
		return nil, 0, apperrors.Unavailable("collection_reaction.favorites.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	var ids []int
	if err := session.SQL(`SELECT reaction.collection_id FROM t_collection_reaction reaction
		JOIN t_collection c ON c.id = reaction.collection_id AND c.is_delete = 0
		  AND c.visibility IN ('public', 'unlisted') AND c.moderation_status = 'visible'
		JOIN t_user_info owner ON owner.id = c.user_id AND owner.is_disable = 0
		WHERE reaction.user_info_id = ? AND reaction.reaction = 'favorite'
		ORDER BY reaction.create_time DESC, reaction.id DESC LIMIT ? OFFSET ?`, userInfoID, limit, offset).Find(&ids); err != nil {
		return nil, 0, apperrors.Unavailable("collection_reaction.favorites.list", err)
	}
	return ids, total, nil
}
