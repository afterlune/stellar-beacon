package repository

import (
	"context"
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	pgsql "github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

var _ port.CollectionSubscriptionRepository = (*MyCollectionSubscriptionRepo)(nil)

type MyCollectionSubscriptionRepo struct{ engine *xorm.Engine }

func NewCollectionSubscriptionRepo(engine *xorm.Engine) *MyCollectionSubscriptionRepo {
	return &MyCollectionSubscriptionRepo{engine: engine}
}

func (r *MyCollectionSubscriptionRepo) Subscribe(ctx context.Context, userID, collectionID int) error {
	if userID <= 0 || collectionID <= 0 {
		return apperrors.Invalid("collection_subscription.create", "invalid subscription")
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var row struct {
			OwnerID          int    `xorm:"user_id"`
			Visibility       string `xorm:"visibility"`
			ModerationStatus string `xorm:"moderation_status"`
		}
		found, err := session.SQL(`
			SELECT c.user_id, c.visibility, c.moderation_status
			FROM t_collection c
			JOIN t_user_info owner ON owner.id = c.user_id AND owner.is_disable = 0
			WHERE c.id = ? AND c.is_delete = 0`, collectionID).Get(&row)
		if err != nil {
			return apperrors.Unavailable("collection_subscription.resolve", err)
		}
		if !found {
			return apperrors.NotFound("collection_subscription.resolve")
		}
		if row.OwnerID == userID {
			return apperrors.Invalid("collection_subscription.self", "cannot subscribe to own collection")
		}
		if row.ModerationStatus != "visible" || (row.Visibility != port.CollectionVisibilityPublic && row.Visibility != port.CollectionVisibilityUnlisted) {
			return apperrors.NotFound("collection_subscription.resolve")
		}
		var cursor int64
		if _, err := session.SQL("SELECT COALESCE(MAX(id), 0) FROM t_collection_update_event").Get(&cursor); err != nil {
			return apperrors.Unavailable("collection_subscription.cursor", err)
		}
		if _, err := session.Exec(`
			INSERT INTO t_collection_subscription (user_id, collection_id, muted, start_event_id, last_read_event_id)
			VALUES (?, ?, 0, ?, ?)
			ON CONFLICT (user_id, collection_id)
			DO UPDATE SET updated_at = CURRENT_TIMESTAMP`, userID, collectionID, cursor, cursor); err != nil {
			return apperrors.Unavailable("collection_subscription.create", err)
		}
		return nil
	})
}

func (r *MyCollectionSubscriptionRepo) Unsubscribe(ctx context.Context, userID, collectionID int) error {
	if userID <= 0 || collectionID <= 0 {
		return apperrors.Invalid("collection_subscription.delete", "invalid subscription")
	}
	session, err := repoSession(r.engine, ctx, "collection_subscription.delete")
	if err != nil {
		return err
	}
	if _, err := session.Exec("DELETE FROM t_collection_subscription WHERE user_id = ? AND collection_id = ?", userID, collectionID); err != nil {
		return apperrors.Unavailable("collection_subscription.delete", err)
	}
	return nil
}

func (r *MyCollectionSubscriptionRepo) SetMuted(ctx context.Context, userID, collectionID int, muted bool) error {
	if userID <= 0 || collectionID <= 0 {
		return apperrors.Invalid("collection_subscription.mute", "invalid subscription")
	}
	value := 0
	if muted {
		value = 1
	}
	session, err := repoSession(r.engine, ctx, "collection_subscription.mute")
	if err != nil {
		return err
	}
	result, err := session.Exec(`UPDATE t_collection_subscription SET muted = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND collection_id = ?`, value, userID, collectionID)
	if err != nil {
		return apperrors.Unavailable("collection_subscription.mute", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return apperrors.NotFound("collection_subscription.mute")
	}
	return nil
}

func (r *MyCollectionSubscriptionRepo) GetStatus(ctx context.Context, userID, collectionID int) (port.CollectionSubscriptionStatus, error) {
	var result port.CollectionSubscriptionStatus
	if userID <= 0 || collectionID <= 0 {
		return result, apperrors.Invalid("collection_subscription.status", "invalid subscription")
	}
	session, err := repoSession(r.engine, ctx, "collection_subscription.status")
	if err != nil {
		return result, err
	}
	var row struct {
		Subscribed bool `xorm:"subscribed"`
		Muted      bool `xorm:"muted"`
	}
	if _, err := session.SQL(`
		SELECT EXISTS (SELECT 1 FROM t_collection_subscription WHERE user_id = ? AND collection_id = ?) AS subscribed,
		       COALESCE((SELECT muted = 1 FROM t_collection_subscription WHERE user_id = ? AND collection_id = ?), false) AS muted`,
		userID, collectionID, userID, collectionID).Get(&row); err != nil {
		return result, apperrors.Unavailable("collection_subscription.status", err)
	}
	result.Subscribed = row.Subscribed
	result.Muted = row.Muted
	return result, nil
}

func (r *MyCollectionSubscriptionRepo) ListSubscriptions(ctx context.Context, userID, current, size int) ([]port.CollectionSubscription, int, error) {
	if userID <= 0 {
		return nil, 0, apperrors.Invalid("collection_subscription.list", "invalid user")
	}
	session, err := repoSession(r.engine, ctx, "collection_subscription.list")
	if err != nil {
		return nil, 0, err
	}
	const joins = `
		FROM t_collection_subscription subscription
		JOIN t_collection c ON c.id = subscription.collection_id AND c.is_delete = 0
			AND c.visibility IN ('public', 'unlisted') AND c.moderation_status = 'visible'
		JOIN t_user_info owner ON owner.id = c.user_id AND owner.is_disable = 0
		JOIN t_user_info subscriber ON subscriber.id = subscription.user_id AND subscriber.is_disable = 0
		WHERE subscription.user_id = ?`
	var count int
	if _, err := session.SQL("SELECT count(1)"+joins, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("collection_subscription.list.count", err)
	}
	if count == 0 {
		return []port.CollectionSubscription{}, 0, nil
	}
	limit, offset := pgsql.Page(current, size)
	var rows []struct {
		CollectionID int       `xorm:"collection_id"`
		Slug         string    `xorm:"slug"`
		Title        string    `xorm:"title"`
		Description  string    `xorm:"description"`
		Cover        string    `xorm:"cover"`
		ArticleCount int       `xorm:"article_count"`
		OwnerID      int       `xorm:"owner_id"`
		OwnerHandle  string    `xorm:"owner_handle"`
		OwnerName    string    `xorm:"owner_name"`
		OwnerAvatar  string    `xorm:"owner_avatar"`
		Muted        bool      `xorm:"muted"`
		UnreadCount  int       `xorm:"unread_count"`
		SubscribedAt time.Time `xorm:"subscribed_at"`
	}
	if err := session.SQL(`
		SELECT c.id AS collection_id, c.slug, c.title, c.description,
		       COALESCE((
		         SELECT a.article_cover FROM t_collection_item item
		         JOIN t_article a ON a.id = item.article_id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		         WHERE item.collection_id = c.id ORDER BY item.sort_order ASC, item.id ASC LIMIT 1
		       ), '') AS cover,
		       COALESCE((
		         SELECT count(1) FROM t_collection_item item
		         JOIN t_article a ON a.id = item.article_id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		         WHERE item.collection_id = c.id
		       ), 0) AS article_count,
		       owner.id AS owner_id, owner.handle AS owner_handle, owner.nickname AS owner_name, owner.avatar AS owner_avatar,
		       subscription.muted = 1 AS muted,
		       CASE WHEN subscription.muted = 1 OR subscriber.notify_collection = 0 THEN 0 ELSE COALESCE((
		         SELECT count(1) FROM t_collection_update_event event
		         JOIN t_article article ON article.id = event.article_id
		           AND article.is_delete = 0 AND article.status = 1 AND article.moderation_status = 'visible'
		         WHERE event.collection_id = c.id AND event.id > subscription.last_read_event_id
		       ), 0) END AS unread_count,
		       subscription.created_at AS subscribed_at`+joins+`
		ORDER BY subscription.updated_at DESC, subscription.id DESC LIMIT ? OFFSET ?`, userID, limit, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("collection_subscription.list", err)
	}
	result := make([]port.CollectionSubscription, 0, len(rows))
	for _, row := range rows {
		result = append(result, port.CollectionSubscription{
			CollectionID: row.CollectionID, Slug: row.Slug, Title: row.Title, Description: row.Description, Cover: row.Cover,
			ArticleCount: row.ArticleCount,
			Owner:        port.PublicAuthor{Id: row.OwnerID, Handle: row.OwnerHandle, Nickname: row.OwnerName, Avatar: row.OwnerAvatar},
			Muted:        row.Muted, UnreadCount: row.UnreadCount, SubscribedAt: row.SubscribedAt,
		})
	}
	return result, count, nil
}

func (r *MyCollectionSubscriptionRepo) ListFeed(ctx context.Context, userID, current, size int) ([]port.CollectionFeedItem, int, error) {
	if userID <= 0 {
		return nil, 0, apperrors.Invalid("collection_feed.list", "invalid user")
	}
	session, err := repoSession(r.engine, ctx, "collection_feed.list")
	if err != nil {
		return nil, 0, err
	}
	const joins = `
		FROM t_collection_subscription subscription
		JOIN t_collection_update_event event ON event.collection_id = subscription.collection_id AND event.id > subscription.start_event_id
		JOIN t_collection c ON c.id = event.collection_id AND c.is_delete = 0
			AND c.visibility IN ('public', 'unlisted') AND c.moderation_status = 'visible'
		JOIN t_user_info owner ON owner.id = c.user_id AND owner.is_disable = 0
		JOIN t_article article ON article.id = event.article_id AND article.is_delete = 0 AND article.status = 1 AND article.moderation_status = 'visible'
		WHERE subscription.user_id = ?`
	var count int
	if _, err := session.SQL("SELECT count(1)"+joins, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("collection_feed.count", err)
	}
	if count == 0 {
		return []port.CollectionFeedItem{}, 0, nil
	}
	limit, offset := pgsql.Page(current, size)
	var rows []struct {
		EventID         int64     `xorm:"event_id"`
		CollectionID    int       `xorm:"collection_id"`
		Slug            string    `xorm:"slug"`
		CollectionTitle string    `xorm:"collection_title"`
		ArticleID       int       `xorm:"article_id"`
		ArticleTitle    string    `xorm:"article_title"`
		Excerpt         string    `xorm:"excerpt"`
		Cover           string    `xorm:"cover"`
		OwnerID         int       `xorm:"owner_id"`
		OwnerHandle     string    `xorm:"owner_handle"`
		OwnerName       string    `xorm:"owner_name"`
		OwnerAvatar     string    `xorm:"owner_avatar"`
		PublishedAt     time.Time `xorm:"published_at"`
	}
	if err := session.SQL(`
		SELECT event.id AS event_id, c.id AS collection_id, c.slug, c.title AS collection_title,
		       article.id AS article_id, article.article_title, SUBSTR(article.article_content, 1, 240) AS excerpt,
		       COALESCE(article.article_cover, '') AS cover,
		       owner.id AS owner_id, owner.handle AS owner_handle, owner.nickname AS owner_name, owner.avatar AS owner_avatar,
		       event.created_at AS published_at`+joins+`
		ORDER BY event.created_at DESC, event.id DESC LIMIT ? OFFSET ?`, userID, limit, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("collection_feed.list", err)
	}
	result := make([]port.CollectionFeedItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, port.CollectionFeedItem{
			EventId: row.EventID, CollectionID: row.CollectionID, Slug: row.Slug, CollectionTitle: row.CollectionTitle,
			ArticleID: row.ArticleID, ArticleTitle: row.ArticleTitle, Excerpt: row.Excerpt, Cover: row.Cover,
			Owner:       port.PublicAuthor{Id: row.OwnerID, Handle: row.OwnerHandle, Nickname: row.OwnerName, Avatar: row.OwnerAvatar},
			PublishedAt: row.PublishedAt,
		})
	}
	return result, count, nil
}
