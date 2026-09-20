package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	pgsql "github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

var _ port.FollowRepository = (*MyFollowRepo)(nil)

type MyFollowRepo struct{ engine *xorm.Engine }

func NewFollowRepo(engine *xorm.Engine) *MyFollowRepo { return &MyFollowRepo{engine: engine} }

func (r *MyFollowRepo) Follow(ctx context.Context, followerID, authorID int) error {
	if followerID <= 0 || authorID <= 0 || followerID == authorID {
		return apperrors.Invalid("follow.create", "invalid follow target")
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var author entity.TUserInfo
		found, err := session.SQL("SELECT * FROM t_user_info WHERE id = ? AND is_disable = 0", authorID).Get(&author)
		if err != nil {
			return apperrors.Unavailable("follow.create.author", err)
		}
		if !found {
			return apperrors.NotFound("follow.create.author")
		}
		var cursor int64
		if _, err := session.SQL("SELECT COALESCE(MAX(id), 0) FROM t_author_publish_event").Get(&cursor); err != nil {
			return apperrors.Unavailable("follow.create.cursor", err)
		}
		if _, err := session.Exec(`
			INSERT INTO t_user_follow (follower_id, author_id, start_event_id, last_read_event_id)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (follower_id, author_id) DO NOTHING`, followerID, authorID, cursor, cursor); err != nil {
			return apperrors.Unavailable("follow.create", err)
		}
		return nil
	})
}

func (r *MyFollowRepo) Unfollow(ctx context.Context, followerID, authorID int) error {
	if followerID <= 0 || authorID <= 0 || followerID == authorID {
		return apperrors.Invalid("follow.delete", "invalid follow target")
	}
	session, err := repoSession(r.engine, ctx, "follow.delete")
	if err != nil {
		return err
	}
	if _, err := session.Exec("DELETE FROM t_user_follow WHERE follower_id = ? AND author_id = ?", followerID, authorID); err != nil {
		return apperrors.Unavailable("follow.delete", err)
	}
	return nil
}

func (r *MyFollowRepo) ListFollowing(ctx context.Context, userID, current, size int) ([]port.FollowUser, int, error) {
	session, err := repoSession(r.engine, ctx, "follow.list_following")
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL(`
		SELECT count(1) FROM t_user_follow f
		JOIN t_user_info user_info ON user_info.id = f.author_id AND user_info.is_disable = 0
		WHERE f.follower_id = ?`, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("follow.list_following.count", err)
	}
	page, offset := pgsql.Page(current, size)
	var rows []struct {
		Id         int       `xorm:"id"`
		Handle     string    `xorm:"handle"`
		Nickname   string    `xorm:"nickname"`
		Avatar     string    `xorm:"avatar"`
		Intro      string    `xorm:"intro"`
		FollowedAt time.Time `xorm:"followed_at"`
	}
	if err := session.SQL(`
		SELECT user_info.id, user_info.handle, user_info.nickname, user_info.avatar, COALESCE(user_info.intro, '') AS intro,
		       f.created_at AS followed_at
		FROM t_user_follow f
		JOIN t_user_info user_info ON user_info.id = f.author_id AND user_info.is_disable = 0
		WHERE f.follower_id = ?
		ORDER BY f.updated_at DESC, f.id DESC LIMIT ? OFFSET ?`, userID, page, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("follow.list_following", err)
	}
	result := make([]port.FollowUser, 0, len(rows))
	for _, row := range rows {
		result = append(result, port.FollowUser{Id: row.Id, Handle: row.Handle, Nickname: row.Nickname, Avatar: row.Avatar, Intro: row.Intro, FollowedAt: row.FollowedAt})
	}
	return result, count, nil
}

func (r *MyFollowRepo) ListFollowers(ctx context.Context, userID, current, size int) ([]port.FollowUser, int, error) {
	session, err := repoSession(r.engine, ctx, "follow.list_followers")
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL(`
		SELECT count(1) FROM t_user_follow f
		JOIN t_user_info user_info ON user_info.id = f.follower_id AND user_info.is_disable = 0
		WHERE f.author_id = ?`, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("follow.list_followers.count", err)
	}
	page, offset := pgsql.Page(current, size)
	var rows []struct {
		Id         int       `xorm:"id"`
		Handle     string    `xorm:"handle"`
		Nickname   string    `xorm:"nickname"`
		Avatar     string    `xorm:"avatar"`
		Intro      string    `xorm:"intro"`
		FollowedAt time.Time `xorm:"followed_at"`
	}
	if err := session.SQL(`
		SELECT user_info.id, user_info.handle, user_info.nickname, user_info.avatar, COALESCE(user_info.intro, '') AS intro,
		       f.created_at AS followed_at
		FROM t_user_follow f
		JOIN t_user_info user_info ON user_info.id = f.follower_id AND user_info.is_disable = 0
		WHERE f.author_id = ?
		ORDER BY f.created_at DESC, f.id DESC LIMIT ? OFFSET ?`, userID, page, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("follow.list_followers", err)
	}
	result := make([]port.FollowUser, 0, len(rows))
	for _, row := range rows {
		result = append(result, port.FollowUser{Id: row.Id, Handle: row.Handle, Nickname: row.Nickname, Avatar: row.Avatar, Intro: row.Intro, FollowedAt: row.FollowedAt})
	}
	return result, count, nil
}

func (r *MyFollowRepo) ListFollowFeed(ctx context.Context, userID int, contentType string, current, size int) ([]port.FollowFeedItem, int, error) {
	session, err := repoSession(r.engine, ctx, "follow.feed")
	if err != nil {
		return nil, 0, err
	}
	if contentType != "" && contentType != port.FollowContentArticle && contentType != port.FollowContentTalk {
		return nil, 0, apperrors.Invalid("follow.feed", "invalid content type")
	}
	const fromWhere = `
		FROM t_author_publish_event event
		JOIN t_user_follow follow ON follow.author_id = event.author_id AND follow.follower_id = ? AND event.id > follow.start_event_id
		JOIN t_user_info author ON author.id = event.author_id AND author.is_disable = 0
		LEFT JOIN t_article article ON event.content_type = 'article' AND article.id = event.content_id
			AND article.is_delete = 0 AND article.status = 1 AND article.moderation_status = 'visible'
		LEFT JOIN t_talk talk ON event.content_type = 'talk' AND talk.id = event.content_id
			AND talk.status = 1 AND talk.moderation_status = 'visible'
		WHERE ((event.content_type = 'article' AND article.id IS NOT NULL) OR (event.content_type = 'talk' AND talk.id IS NOT NULL))
		  AND (? = '' OR event.content_type = ?)`
	var count int
	if _, err := session.SQL("SELECT count(1)"+fromWhere, userID, contentType, contentType).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("follow.feed.count", err)
	}
	page, offset := pgsql.Page(current, size)
	var rows []struct {
		EventId      int64     `xorm:"event_id"`
		ContentType  string    `xorm:"content_type"`
		ContentId    int       `xorm:"content_id"`
		AuthorId     int       `xorm:"author_id"`
		AuthorHandle string    `xorm:"author_handle"`
		AuthorName   string    `xorm:"author_name"`
		AuthorAvatar string    `xorm:"author_avatar"`
		Title        string    `xorm:"title"`
		Excerpt      string    `xorm:"excerpt"`
		Cover        string    `xorm:"cover"`
		Images       string    `xorm:"images"`
		PublishedAt  time.Time `xorm:"published_at"`
	}
	if err := session.SQL(`
		SELECT event.id AS event_id, event.content_type, event.content_id, event.author_id,
		       author.handle AS author_handle, author.nickname AS author_name, author.avatar AS author_avatar,
		       COALESCE(article.article_title, '') AS title,
		       CASE WHEN event.content_type = 'article' THEN COALESCE(SUBSTR(article.article_content, 1, 240), '') ELSE COALESCE(SUBSTR(talk.content, 1, 240), '') END AS excerpt,
		       COALESCE(article.article_cover, '') AS cover,
		       COALESCE(talk.images, '') AS images,
		       event.published_at`+fromWhere+`
		ORDER BY event.published_at DESC, event.id DESC LIMIT ? OFFSET ?`, userID, contentType, contentType, page, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("follow.feed", err)
	}
	result := make([]port.FollowFeedItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, port.FollowFeedItem{
			EventId: row.EventId, ContentType: row.ContentType, ContentId: row.ContentId,
			Author: port.PublicAuthor{Id: row.AuthorId, Handle: row.AuthorHandle, Nickname: row.AuthorName, Avatar: row.AuthorAvatar},
			Title:  row.Title, Excerpt: row.Excerpt, Cover: row.Cover, Images: decodeTalkImages(row.Images), PublishedAt: row.PublishedAt,
		})
	}
	return result, count, nil
}

func (r *MyFollowRepo) ListNotifications(ctx context.Context, userID int, group string, current, size int) (port.NotificationPage, error) {
	if userID <= 0 {
		return port.NotificationPage{}, apperrors.Invalid("notification.list", "invalid user")
	}
	if group == "" {
		group = port.NotificationGroupAll
	}
	if !port.ValidNotificationGroup(group) {
		return port.NotificationPage{}, apperrors.Invalid("notification.list", "invalid group")
	}
	session, err := repoSession(r.engine, ctx, "notification.list")
	if err != nil {
		return port.NotificationPage{}, err
	}
	filterGroup := group
	if group == port.NotificationGroupAll {
		filterGroup = ""
	}
	args := notificationFeedArgs(userID, filterGroup)
	var result port.NotificationPage
	if _, err := session.SQL("SELECT count(1) FROM ("+notificationFeedSQL+") notification_feed", args...).Get(&result.Count); err != nil {
		return port.NotificationPage{}, apperrors.Unavailable("notification.count", err)
	}
	if _, err := session.SQL("SELECT count(1) FROM ("+notificationFeedSQL+") notification_feed WHERE is_read = false", args...).Get(&result.UnreadCount); err != nil {
		return port.NotificationPage{}, apperrors.Unavailable("notification.unread", err)
	}
	if _, err := session.SQL("SELECT count(1) FROM ("+notificationFeedSQL+") notification_feed WHERE is_read = false", notificationFeedArgs(userID, "")...).Get(&result.TotalUnreadCount); err != nil {
		return port.NotificationPage{}, apperrors.Unavailable("notification.unread", err)
	}
	if _, err := session.SQL(`
		SELECT
			COALESCE((
				SELECT MAX(event.id)
				FROM t_author_publish_event event
				JOIN t_user_follow follow ON follow.author_id = event.author_id
					AND follow.follower_id = ? AND event.id > follow.start_event_id
			), 0) AS publish_event_id,
			COALESCE((SELECT MAX(id) FROM t_user_notification WHERE recipient_id = ?), 0) AS interaction_id`,
		userID, userID).Get(&result.ReadCursor); err != nil {
		return port.NotificationPage{}, apperrors.Unavailable("notification.cursor", err)
	}
	page, offset := pgsql.Page(current, size)
	var rows []struct {
		Key         string    `xorm:"notification_key"`
		Type        string    `xorm:"notification_type"`
		Group       string    `xorm:"notification_group"`
		ActorId     int       `xorm:"actor_id"`
		ActorHandle string    `xorm:"actor_handle"`
		ActorName   string    `xorm:"actor_name"`
		ActorAvatar string    `xorm:"actor_avatar"`
		ContentType string    `xorm:"content_type"`
		ContentId   int       `xorm:"content_id"`
		CommentId   int       `xorm:"comment_id"`
		Title       string    `xorm:"title"`
		Excerpt     string    `xorm:"excerpt"`
		Cover       string    `xorm:"cover"`
		Images      string    `xorm:"images"`
		CreatedAt   time.Time `xorm:"created_at"`
		IsRead      bool      `xorm:"is_read"`
	}
	pageArgs := append(append([]interface{}{}, args...), page, offset)
	if err := session.SQL(notificationFeedSQL+` ORDER BY created_at DESC, sort_id DESC, notification_key DESC LIMIT ? OFFSET ?`, pageArgs...).Find(&rows); err != nil {
		return port.NotificationPage{}, apperrors.Unavailable("notification.list", err)
	}
	result.Records = make([]port.NotificationItem, 0, len(rows))
	for _, row := range rows {
		result.Records = append(result.Records, port.NotificationItem{
			Key: row.Key, Type: row.Type, Group: row.Group,
			Actor:       port.PublicAuthor{Id: row.ActorId, Handle: row.ActorHandle, Nickname: row.ActorName, Avatar: row.ActorAvatar},
			ContentType: row.ContentType, ContentId: row.ContentId, CommentId: row.CommentId,
			Title: row.Title, Excerpt: row.Excerpt, Cover: row.Cover, Images: decodeTalkImages(row.Images),
			CreatedAt: row.CreatedAt, Read: row.IsRead,
		})
	}
	return result, nil
}

func (r *MyFollowRepo) UnreadNotificationCount(ctx context.Context, userID int) (int, error) {
	page, err := r.ListNotifications(ctx, userID, port.NotificationGroupAll, 1, 1)
	if err != nil {
		return 0, err
	}
	return page.UnreadCount, nil
}

func (r *MyFollowRepo) MarkNotificationsRead(ctx context.Context, userID int, cursor port.NotificationCursor) error {
	if userID <= 0 {
		return apperrors.Invalid("notification.read", "invalid user")
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var maxPublish, maxInteraction int64
		if _, err := session.SQL(`
			SELECT COALESCE(MAX(event.id), 0)
			FROM t_author_publish_event event
			JOIN t_user_follow follow ON follow.author_id = event.author_id
				AND follow.follower_id = ? AND event.id > follow.start_event_id`, userID).Get(&maxPublish); err != nil {
			return apperrors.Unavailable("notification.read.publish_cursor", err)
		}
		if _, err := session.SQL(`SELECT COALESCE(MAX(id), 0) FROM t_user_notification WHERE recipient_id = ?`, userID).Get(&maxInteraction); err != nil {
			return apperrors.Unavailable("notification.read.interaction_cursor", err)
		}
		if cursor.PublishEventId <= 0 || cursor.PublishEventId > maxPublish {
			cursor.PublishEventId = maxPublish
		}
		if cursor.InteractionId <= 0 || cursor.InteractionId > maxInteraction {
			cursor.InteractionId = maxInteraction
		}
		if _, err := session.Exec(`
			UPDATE t_user_follow
			SET last_read_event_id = GREATEST(last_read_event_id, ?), updated_at = CURRENT_TIMESTAMP
			WHERE follower_id = ?`, cursor.PublishEventId, userID); err != nil {
			return apperrors.Unavailable("notification.read.publish", err)
		}
		if _, err := session.Exec(`
			UPDATE t_user_notification
			SET read_at = CURRENT_TIMESTAMP
			WHERE recipient_id = ? AND id <= ? AND read_at IS NULL`, userID, cursor.InteractionId); err != nil {
			return apperrors.Unavailable("notification.read.interaction", err)
		}
		return nil
	})
}

const notificationFeedSQL = `
	SELECT
		'publish:' || event.id::text AS notification_key,
		'publish' AS notification_type,
		'publish' AS notification_group,
		event.author_id,
		author.handle AS actor_handle,
		author.nickname AS actor_name,
		author.avatar AS actor_avatar,
		event.content_type,
		event.content_id,
		0::bigint AS comment_id,
		COALESCE(article.article_title, '') AS title,
		CASE WHEN event.content_type = 'article' THEN COALESCE(SUBSTR(article.article_content, 1, 240), '') ELSE COALESCE(SUBSTR(talk.content, 1, 240), '') END AS excerpt,
		COALESCE(article.article_cover, '') AS cover,
		COALESCE(talk.images, '') AS images,
		event.published_at AS created_at,
		(event.id <= follow.last_read_event_id) AS is_read,
		event.id AS sort_id
	FROM t_author_publish_event event
	JOIN t_user_follow follow ON follow.author_id = event.author_id AND follow.follower_id = ? AND event.id > follow.start_event_id
	JOIN t_user_info author ON author.id = event.author_id AND author.is_disable = 0
	LEFT JOIN t_article article ON event.content_type = 'article' AND article.id = event.content_id
		AND article.is_delete = 0 AND article.status = 1 AND article.moderation_status = 'visible'
	LEFT JOIN t_talk talk ON event.content_type = 'talk' AND talk.id = event.content_id
		AND talk.status = 1 AND talk.moderation_status = 'visible'
	WHERE ((event.content_type = 'article' AND article.id IS NOT NULL) OR (event.content_type = 'talk' AND talk.id IS NOT NULL))
	  AND (? = '' OR ? = 'publish')
	UNION ALL
	SELECT
		'interaction:' || notification.id::text AS notification_key,
		notification.type AS notification_type,
		CASE WHEN notification.type IN ('comment', 'reply') THEN 'comment' ELSE 'reaction' END AS notification_group,
		notification.actor_id,
		actor.handle AS actor_handle,
		actor.nickname AS actor_name,
		actor.avatar AS actor_avatar,
		notification.content_type,
		notification.content_id,
		notification.comment_id,
		CASE WHEN notification.content_type = 'article' THEN COALESCE(article.article_title, '') ELSE '' END AS title,
		CASE
			WHEN notification.type IN ('comment', 'reply') THEN COALESCE(SUBSTR(comment.comment_content, 1, 240), '')
			WHEN notification.content_type = 'article' THEN COALESCE(SUBSTR(article.article_content, 1, 240), '')
			ELSE COALESCE(SUBSTR(talk.content, 1, 240), '')
		END AS excerpt,
		COALESCE(article.article_cover, '') AS cover,
		COALESCE(talk.images, '') AS images,
		notification.created_at,
		(notification.read_at IS NOT NULL) AS is_read,
		notification.id AS sort_id
	FROM t_user_notification notification
	JOIN t_user_info actor ON actor.id = notification.actor_id AND actor.is_disable = 0
	LEFT JOIN t_comment comment ON notification.comment_id > 0 AND comment.id = notification.comment_id
	LEFT JOIN t_article article ON notification.content_type = 'article' AND article.id = notification.content_id
		AND article.is_delete = 0 AND article.status = 1 AND article.moderation_status = 'visible'
	LEFT JOIN t_talk talk ON notification.content_type = 'talk' AND talk.id = notification.content_id
		AND talk.status = 1 AND talk.moderation_status = 'visible'
	LEFT JOIN t_article_reaction reaction ON notification.type IN ('like', 'favorite')
		AND reaction.article_id = notification.content_id
		AND reaction.user_info_id = notification.actor_id
		AND reaction.reaction = notification.type
	WHERE notification.recipient_id = ?
	  AND (
		(notification.type IN ('comment', 'reply') AND comment.id IS NOT NULL AND comment.is_delete = 0 AND comment.is_review = 1
			AND ((notification.content_type = 'article' AND article.id IS NOT NULL) OR (notification.content_type = 'talk' AND talk.id IS NOT NULL)))
		OR
		(notification.type IN ('like', 'favorite') AND reaction.id IS NOT NULL AND notification.content_type = 'article' AND article.id IS NOT NULL)
	  )
	  AND (? = '' OR (? = 'comment' AND notification.type IN ('comment', 'reply')) OR (? = 'reaction' AND notification.type IN ('like', 'favorite')))
`

func notificationFeedArgs(userID int, group string) []interface{} {
	return []interface{}{userID, group, group, userID, group, group, group}
}
func decodeTalkImages(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var images []string
	if err := json.Unmarshal([]byte(value), &images); err != nil {
		return nil
	}
	return images
}
