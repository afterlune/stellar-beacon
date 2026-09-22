package repository

import (
	"context"
	"strings"
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	pgsql "github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

var _ port.CollectionRepository = (*MyCollectionRepo)(nil)

type MyCollectionRepo struct{ engine *xorm.Engine }

func NewCollectionRepo(engine *xorm.Engine) *MyCollectionRepo {
	return &MyCollectionRepo{engine: engine}
}

type collectionSummaryRow struct {
	ID               int       `xorm:"id"`
	Slug             string    `xorm:"slug"`
	Title            string    `xorm:"title"`
	Description      string    `xorm:"description"`
	Visibility       string    `xorm:"visibility"`
	OwnerID          int       `xorm:"owner_id"`
	OwnerHandle      string    `xorm:"owner_handle"`
	OwnerNickname    string    `xorm:"owner_nickname"`
	OwnerAvatar      string    `xorm:"owner_avatar"`
	Cover            string    `xorm:"cover"`
	ArticleCount     int       `xorm:"article_count"`
	HotScore         int       `xorm:"hot_score"`
	ModerationStatus string    `xorm:"moderation_status"`
	ModerationReason string    `xorm:"moderation_reason"`
	CreatedAt        time.Time `xorm:"created_at"`
	UpdatedAt        time.Time `xorm:"updated_at"`
}

func (row collectionSummaryRow) toPort() *port.CollectionSummary {
	return &port.CollectionSummary{
		ID: row.ID, Slug: row.Slug, Title: row.Title, Description: row.Description,
		Visibility: row.Visibility,
		Owner:      &port.PublicAuthor{Id: row.OwnerID, Handle: row.OwnerHandle, Nickname: row.OwnerNickname, Avatar: row.OwnerAvatar},
		Cover:      row.Cover, ArticleCount: row.ArticleCount, HotScore: row.HotScore,
		ModerationStatus: row.ModerationStatus, ModerationReason: row.ModerationReason,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

const collectionVisibleArticle = `a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'`

const collectionSummarySelect = `
	SELECT c.id, c.slug, c.title, c.description, c.visibility,
	       u.id AS owner_id, u.handle AS owner_handle, u.nickname AS owner_nickname, u.avatar AS owner_avatar,
	       COALESCE((
	         SELECT a.article_cover FROM t_collection_item cover_item
	         JOIN t_article a ON a.id = cover_item.article_id AND ` + collectionVisibleArticle + `
	         WHERE cover_item.collection_id = c.id ORDER BY cover_item.sort_order ASC, cover_item.id ASC LIMIT 1
	       ), '') AS cover,
	       COALESCE((
	         SELECT count(1) FROM t_collection_item visible_item
	         JOIN t_article a ON a.id = visible_item.article_id AND ` + collectionVisibleArticle + `
	         WHERE visible_item.collection_id = c.id
	       ), 0) AS article_count,
	       0 AS hot_score,
	       c.moderation_status, c.moderation_reason,
	       c.create_time AS created_at, c.update_time AS updated_at
	FROM t_collection c
	JOIN t_user_info u ON u.id = c.user_id AND u.is_disable = 0`

func (r *MyCollectionRepo) ListPublic(ctx context.Context, sort string, current, size int) ([]*port.CollectionSummary, int, error) {
	session, err := repoSession(r.engine, ctx, "collection.public.list")
	if err != nil {
		return nil, 0, err
	}
	const filter = `
		WHERE c.visibility = 'public' AND c.moderation_status = 'visible' AND c.is_delete = 0
		  AND EXISTS (
			SELECT 1 FROM t_collection_item i JOIN t_article a ON a.id = i.article_id AND ` + collectionVisibleArticle + `
			WHERE i.collection_id = c.id
		  )`
	var count int
	if _, err := session.SQL("SELECT count(1) FROM t_collection c JOIN t_user_info u ON u.id = c.user_id AND u.is_disable = 0" + filter).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("collection.public.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	var rows []collectionSummaryRow
	if sort == port.CollectionSortHot {
		query := "WITH " + discoveryScoredCTE + `,
			collection_hot AS (
				SELECT i.collection_id, SUM(COALESCE(s.hot_score, 0))::int AS hot_score
				FROM t_collection_item i
				JOIN t_collection c ON c.id = i.collection_id
				JOIN t_article a ON a.id = i.article_id AND ` + collectionVisibleArticle + `
				LEFT JOIN scored s ON s.id = a.id
				WHERE c.visibility = 'public' AND c.moderation_status = 'visible' AND c.is_delete = 0
				GROUP BY i.collection_id
			)
			SELECT base.id, base.slug, base.title, base.description, base.visibility,
			       base.owner_id, base.owner_handle, base.owner_nickname, base.owner_avatar,
			       base.cover, base.article_count, COALESCE(hot.hot_score, 0) AS hot_score,
			       base.moderation_status, base.moderation_reason, base.created_at, base.updated_at
			FROM (` + collectionSummarySelect + filter + `) base
			JOIN collection_hot hot ON hot.collection_id = base.id
			ORDER BY hot.hot_score DESC, base.updated_at DESC, base.id DESC LIMIT ? OFFSET ?`
		windowStart, windowDate := discoveryWindow(time.Now())
		if err := session.SQL(query, windowDate, windowStart, windowStart, limit, offset).Find(&rows); err != nil {
			return nil, 0, apperrors.Unavailable("collection.public.hot", err)
		}
	} else {
		query := collectionSummarySelect + filter + ` ORDER BY c.update_time DESC, c.id DESC LIMIT ? OFFSET ?`
		if err := session.SQL(query, limit, offset).Find(&rows); err != nil {
			return nil, 0, apperrors.Unavailable("collection.public.latest", err)
		}
	}
	items := make([]*port.CollectionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toPort())
	}
	return items, count, nil
}

func (r *MyCollectionRepo) ListPublicByOwner(ctx context.Context, userID, current, size int) ([]*port.CollectionSummary, int, error) {
	session, err := repoSession(r.engine, ctx, "collection.public.owner")
	if err != nil {
		return nil, 0, err
	}
	const filter = `
		WHERE c.user_id = ? AND c.visibility = 'public' AND c.moderation_status = 'visible' AND c.is_delete = 0
		  AND EXISTS (
			SELECT 1 FROM t_collection_item i JOIN t_article a ON a.id = i.article_id AND ` + collectionVisibleArticle + `
			WHERE i.collection_id = c.id
		  )`
	var count int
	if _, err := session.SQL("SELECT count(1) FROM t_collection c"+filter, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("collection.public.owner.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	var rows []collectionSummaryRow
	if err := session.SQL(collectionSummarySelect+filter+" ORDER BY c.update_time DESC, c.id DESC LIMIT ? OFFSET ?", userID, limit, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("collection.public.owner.list", err)
	}
	items := make([]*port.CollectionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toPort())
	}
	return items, count, nil
}

func (r *MyCollectionRepo) getSummary(session *xorm.Session, where string, args ...interface{}) (collectionSummaryRow, error) {
	var row collectionSummaryRow
	found, err := session.SQL(collectionSummarySelect+` WHERE `+where, args...).Get(&row)
	if err != nil {
		return row, apperrors.Unavailable("collection.get", err)
	}
	if !found {
		return row, apperrors.NotFound("collection.get")
	}
	return row, nil
}

func (r *MyCollectionRepo) getItems(session *xorm.Session, collectionID int) ([]port.CollectionItemRecord, error) {
	var rows []struct {
		ArticleID int    `xorm:"article_id"`
		Note      string `xorm:"note"`
		Position  int    `xorm:"position"`
		Available bool   `xorm:"available"`
	}
	if err := session.SQL(`
		SELECT item.article_id, item.note, item.sort_order AS position,
		       EXISTS (
		         SELECT 1 FROM t_article a WHERE a.id = item.article_id AND `+collectionVisibleArticle+`
		       ) AS available
		FROM t_collection_item item
		WHERE item.collection_id = ?
		ORDER BY item.sort_order ASC, item.id ASC`, collectionID).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("collection.items", err)
	}
	items := make([]port.CollectionItemRecord, 0, len(rows))
	for _, row := range rows {
		items = append(items, port.CollectionItemRecord{ArticleID: row.ArticleID, Note: row.Note, Position: row.Position, Available: row.Available})
	}
	return items, nil
}

func (r *MyCollectionRepo) GetPublicBySlug(ctx context.Context, slug string) (port.CollectionRecord, error) {
	var result port.CollectionRecord
	if strings.TrimSpace(slug) == "" {
		return result, apperrors.Invalid("collection.public.get", "invalid slug")
	}
	session, err := repoSession(r.engine, ctx, "collection.public.get")
	if err != nil {
		return result, err
	}
	row, err := r.getSummary(session, `c.slug = ? AND c.visibility IN ('public', 'unlisted')
		AND c.moderation_status = 'visible' AND c.is_delete = 0`, strings.TrimSpace(slug))
	if err != nil {
		return result, err
	}
	items, err := r.getItems(session, row.ID)
	if err != nil {
		return result, err
	}
	result.Collection = *row.toPort()
	result.Items = items
	return result, nil
}

func (r *MyCollectionRepo) ListOwned(ctx context.Context, userID, current, size int) ([]*port.CollectionSummary, int, error) {
	session, err := repoSession(r.engine, ctx, "collection.owned.list")
	if err != nil {
		return nil, 0, err
	}
	const filter = ` WHERE c.user_id = ? AND c.is_delete = 0`
	var count int
	if _, err := session.SQL("SELECT count(1) FROM t_collection c"+filter, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("collection.owned.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	var rows []collectionSummaryRow
	if err := session.SQL(collectionSummarySelect+filter+" ORDER BY c.update_time DESC, c.id DESC LIMIT ? OFFSET ?", userID, limit, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("collection.owned.list", err)
	}
	items := make([]*port.CollectionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toPort())
	}
	return items, count, nil
}

func (r *MyCollectionRepo) GetOwned(ctx context.Context, userID, collectionID int) (port.CollectionRecord, error) {
	var result port.CollectionRecord
	session, err := repoSession(r.engine, ctx, "collection.owned.get")
	if err != nil {
		return result, err
	}
	row, err := r.getSummary(session, `c.id = ? AND c.user_id = ? AND c.is_delete = 0`, collectionID, userID)
	if err != nil {
		return result, err
	}
	items, err := r.getItems(session, collectionID)
	if err != nil {
		return result, err
	}
	result.Collection = *row.toPort()
	result.Items = items
	return result, nil
}

func (r *MyCollectionRepo) ListAdmin(ctx context.Context, current, size int, moderation, keywords string) ([]*port.CollectionSummary, int, error) {
	session, err := repoSession(r.engine, ctx, "collection.admin.list")
	if err != nil {
		return nil, 0, err
	}
	where := `c.is_delete = 0`
	args := []interface{}{}
	if moderation == "visible" || moderation == "hidden" {
		where += ` AND c.moderation_status = ?`
		args = append(args, moderation)
	}
	if keyword := strings.TrimSpace(keywords); keyword != "" {
		where += ` AND (c.title LIKE ? ESCAPE '\' OR c.description LIKE ? ESCAPE '\')`
		pattern := pgsql.ContainsPattern(keyword)
		args = append(args, pattern, pattern)
	}
	var count int
	if _, err := session.SQL("SELECT count(1) FROM t_collection c JOIN t_user_info u ON u.id = c.user_id AND u.is_disable = 0 WHERE "+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("collection.admin.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	pageArgs := append(append([]interface{}{}, args...), limit, offset)
	var rows []collectionSummaryRow
	if err := session.SQL(collectionSummarySelect+" WHERE "+where+" ORDER BY c.update_time DESC, c.id DESC LIMIT ? OFFSET ?", pageArgs...).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("collection.admin.list", err)
	}
	items := make([]*port.CollectionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toPort())
	}
	return items, count, nil
}

func (r *MyCollectionRepo) CreateOwned(ctx context.Context, userID int, slug string, input port.CollectionSaveInput) (port.CollectionSummary, error) {
	var result port.CollectionSummary
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var id int
		if _, err := session.SQL(`
			INSERT INTO t_collection (user_id, slug, title, description, visibility)
			VALUES (?, ?, ?, ?, ?)
			RETURNING id`, userID, slug, input.Title, input.Description, input.Visibility).Get(&id); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return apperrors.Conflict("collection.create.slug", "slug already exists")
			}
			return apperrors.Unavailable("collection.create", err)
		}
		row, err := r.getSummary(session, `c.id = ? AND c.user_id = ?`, id, userID)
		if err != nil {
			return err
		}
		result = *row.toPort()
		return nil
	})
	return result, err
}

func (r *MyCollectionRepo) UpdateOwned(ctx context.Context, userID, collectionID int, input port.CollectionSaveInput) (port.CollectionSummary, error) {
	var result port.CollectionSummary
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		affected, err := session.Exec(`
			UPDATE t_collection SET title = ?, description = ?, visibility = ?, update_time = CURRENT_TIMESTAMP
			WHERE id = ? AND user_id = ? AND is_delete = 0`,
			input.Title, input.Description, input.Visibility, collectionID, userID)
		if err != nil {
			return apperrors.Unavailable("collection.update", err)
		}
		rows, err := affected.RowsAffected()
		if err != nil {
			return apperrors.Unavailable("collection.update.rows", err)
		}
		if rows == 0 {
			return apperrors.NotFound("collection.update")
		}
		row, err := r.getSummary(session, `c.id = ? AND c.user_id = ?`, collectionID, userID)
		if err != nil {
			return err
		}
		result = *row.toPort()
		return nil
	})
	return result, err
}

func (r *MyCollectionRepo) DeleteOwned(ctx context.Context, userID, collectionID int) error {
	session, err := repoSession(r.engine, ctx, "collection.delete")
	if err != nil {
		return err
	}
	result, err := session.Exec(`UPDATE t_collection SET is_delete = 1, update_time = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ? AND is_delete = 0`, collectionID, userID)
	if err != nil {
		return apperrors.Unavailable("collection.delete", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("collection.delete.rows", err)
	}
	if rows == 0 {
		return apperrors.NotFound("collection.delete")
	}
	return nil
}

func (r *MyCollectionRepo) touchOwned(session *xorm.Session, userID, collectionID int) error {
	result, err := session.Exec(`UPDATE t_collection SET update_time = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ? AND is_delete = 0`, collectionID, userID)
	if err != nil {
		return apperrors.Unavailable("collection.touch", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("collection.touch.rows", err)
	}
	if rows == 0 {
		return apperrors.NotFound("collection.touch")
	}
	return nil
}

func (r *MyCollectionRepo) AddItem(ctx context.Context, userID, collectionID, articleID int, note string) error {
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var collection struct {
			Visibility       string `xorm:"visibility"`
			ModerationStatus string `xorm:"moderation_status"`
		}
		found, err := session.SQL(`SELECT visibility, moderation_status FROM t_collection
			WHERE id = ? AND user_id = ? AND is_delete = 0 FOR UPDATE`, collectionID, userID).Get(&collection)
		if err != nil {
			return apperrors.Unavailable("collection.item.owner", err)
		}
		if !found {
			return apperrors.NotFound("collection.item.owner")
		}
		var existed bool
		if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_collection_item WHERE collection_id = ? AND article_id = ?)`,
			collectionID, articleID).Get(&existed); err != nil {
			return apperrors.Unavailable("collection.item.exists", err)
		}
		var next int
		if _, err := session.SQL(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM t_collection_item WHERE collection_id = ?`, collectionID).Get(&next); err != nil {
			return apperrors.Unavailable("collection.item.position", err)
		}
		if _, err := session.Exec(`
			INSERT INTO t_collection_item (collection_id, article_id, note, sort_order)
			VALUES (?, ?, ?, ?)
		ON CONFLICT (collection_id, article_id)
		DO UPDATE SET note = EXCLUDED.note, update_time = CURRENT_TIMESTAMP`, collectionID, articleID, note, next); err != nil {
			return apperrors.Unavailable("collection.item.add", err)
		}
		if !existed && collection.ModerationStatus == "visible" &&
			(collection.Visibility == port.CollectionVisibilityPublic || collection.Visibility == port.CollectionVisibilityUnlisted) {
			var hasSubscriber bool
			if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_collection_subscription WHERE collection_id = ?)`, collectionID).Get(&hasSubscriber); err != nil {
				return apperrors.Unavailable("collection.item.subscribers", err)
			}
			if hasSubscriber {
				if _, err := session.Exec(`INSERT INTO t_collection_update_event (collection_id, owner_id, article_id) VALUES (?, ?, ?)`,
					collectionID, userID, articleID); err != nil {
					return apperrors.Unavailable("collection.item.event", err)
				}
			}
		}
		if _, err := session.Exec(`UPDATE t_collection SET update_time = CURRENT_TIMESTAMP WHERE id = ?`, collectionID); err != nil {
			return apperrors.Unavailable("collection.item.touch", err)
		}
		return nil
	})
}

func (r *MyCollectionRepo) RemoveItem(ctx context.Context, userID, collectionID, articleID int) error {
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if err := r.touchOwned(session, userID, collectionID); err != nil {
			return err
		}
		if _, err := session.Exec("DELETE FROM t_collection_item WHERE collection_id = ? AND article_id = ?", collectionID, articleID); err != nil {
			return apperrors.Unavailable("collection.item.remove", err)
		}
		return nil
	})
}

func (r *MyCollectionRepo) Reorder(ctx context.Context, userID, collectionID int, articleIDs []int) error {
	if len(articleIDs) == 0 {
		return apperrors.Invalid("collection.reorder", "article order is required")
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if err := r.touchOwned(session, userID, collectionID); err != nil {
			return err
		}
		var existing int
		if _, err := session.SQL("SELECT count(1) FROM t_collection_item WHERE collection_id = ?", collectionID).Get(&existing); err != nil {
			return apperrors.Unavailable("collection.reorder.count", err)
		}
		if existing != len(articleIDs) {
			return apperrors.Invalid("collection.reorder", "article order must include every item")
		}
		for index, articleID := range articleIDs {
			result, err := session.Exec(`UPDATE t_collection_item SET sort_order = ?, update_time = CURRENT_TIMESTAMP
				WHERE collection_id = ? AND article_id = ?`, index+1, collectionID, articleID)
			if err != nil {
				return apperrors.Unavailable("collection.reorder.item", err)
			}
			rows, err := result.RowsAffected()
			if err != nil {
				return apperrors.Unavailable("collection.reorder.rows", err)
			}
			if rows == 0 {
				return apperrors.Invalid("collection.reorder", "unknown collection item")
			}
		}
		return nil
	})
}
