package repository

import (
	"context"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

var _ port.ArticleReactionRepository = (*MyArticleReactionRepo)(nil)

// MyArticleReactionRepo stores reader reactions. Aggregates are derived on
// read so the reaction table stays the single source of truth.
type MyArticleReactionRepo struct{ engine *xorm.Engine }

func NewArticleReactionRepo(engine *xorm.Engine) *MyArticleReactionRepo {
	return &MyArticleReactionRepo{engine: engine}
}

type reactionCountRow struct {
	ArticleId int    `xorm:"article_id"`
	Reaction  string `xorm:"reaction"`
	Total     int    `xorm:"total"`
}

func (r *MyArticleReactionRepo) Toggle(ctx context.Context, articleID, userInfoID int, reaction string) (bool, error) {
	active := false
	err := repoTx(r.engine, ctx, "article_reaction.toggle", func(session *xorm.Session) error {
		result, err := session.Exec(
			`INSERT INTO t_article_reaction (article_id, user_info_id, reaction, create_time)
			 VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			 ON CONFLICT (article_id, user_info_id, reaction) DO NOTHING`,
			articleID, userInfoID, reaction,
		)
		if err != nil {
			return apperrors.Unavailable("article_reaction.toggle", err)
		}
		inserted, err := result.RowsAffected()
		if err != nil {
			return apperrors.Unavailable("article_reaction.toggle", err)
		}
		if inserted > 0 {
			if err := recordReactionNotification(session, articleID, userInfoID, reaction); err != nil {
				return err
			}
			active = true
			return nil
		}
		if _, err := session.Exec(
			"DELETE FROM t_article_reaction WHERE article_id = ? AND user_info_id = ? AND reaction = ?",
			articleID, userInfoID, reaction,
		); err != nil {
			return apperrors.Unavailable("article_reaction.toggle", err)
		}
		active = false
		return nil
	})
	if err != nil {
		return false, err
	}
	return active, nil
}

func (r *MyArticleReactionRepo) Counts(ctx context.Context, articleIDs []int) (map[int]port.ReactionCounts, error) {
	counts := make(map[int]port.ReactionCounts, len(articleIDs))
	if len(articleIDs) == 0 {
		return counts, nil
	}
	session, err := repoSession(r.engine, ctx, "article_reaction.counts")
	if err != nil {
		return nil, err
	}
	var rows []reactionCountRow
	if err := session.Table("t_article_reaction").
		Select("article_id, reaction, COUNT(1) AS total").
		In("article_id", articleIDs).
		GroupBy("article_id, reaction").
		Find(&rows); err != nil {
		return nil, apperrors.Unavailable("article_reaction.counts", err)
	}
	for _, row := range rows {
		entry := counts[row.ArticleId]
		switch row.Reaction {
		case port.ReactionLike:
			entry.LikeCount = row.Total
		case port.ReactionFavorite:
			entry.FavoriteCount = row.Total
		}
		counts[row.ArticleId] = entry
	}
	return counts, nil
}

func (r *MyArticleReactionRepo) States(ctx context.Context, userInfoID int, articleIDs []int) (map[int]map[string]bool, error) {
	states := make(map[int]map[string]bool, len(articleIDs))
	if userInfoID <= 0 || len(articleIDs) == 0 {
		return states, nil
	}
	session, err := repoSession(r.engine, ctx, "article_reaction.states")
	if err != nil {
		return nil, err
	}
	var reactions []entity.TArticleReaction
	if err := session.Table("t_article_reaction").
		In("article_id", articleIDs).
		Where("user_info_id = ?", userInfoID).
		Find(&reactions); err != nil {
		return nil, apperrors.Unavailable("article_reaction.states", err)
	}
	for _, reaction := range reactions {
		entry := states[reaction.ArticleId]
		if entry == nil {
			entry = map[string]bool{}
		}
		entry[reaction.Reaction] = true
		states[reaction.ArticleId] = entry
	}
	return states, nil
}

func (r *MyArticleReactionRepo) ListArticleIDsByUser(ctx context.Context, userInfoID int, reaction string, current, size int) ([]int, int64, error) {
	session, err := repoSession(r.engine, ctx, "article_reaction.list")
	if err != nil {
		return nil, 0, err
	}
	// A reaction on a draft or recycled article is still recorded, but the
	// favourites list renders public cards only, so the count and the page must
	// apply the same visibility filter or they would disagree.
	const visibleFilter = ` FROM t_article_reaction r
		JOIN t_article a ON a.id = r.article_id
		WHERE r.user_info_id = ? AND r.reaction = ? AND a.is_delete = 0 AND a.status IN (1, 2)`
	var total int64
	if _, err := session.SQL("SELECT count(0)"+visibleFilter, userInfoID, reaction).Get(&total); err != nil {
		return nil, 0, apperrors.Unavailable("article_reaction.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	var reactions []entity.TArticleReaction
	if err := session.SQL("SELECT r.id, r.article_id, r.user_info_id, r.reaction, r.create_time"+visibleFilter+" ORDER BY r.id DESC LIMIT ? OFFSET ?", userInfoID, reaction, limit, offset).Find(&reactions); err != nil {
		return nil, 0, apperrors.Unavailable("article_reaction.list", err)
	}
	ids := make([]int, 0, len(reactions))
	for _, reaction := range reactions {
		ids = append(ids, reaction.ArticleId)
	}
	return ids, total, nil
}
