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

func (r *MyFollowRepo) ListNotifications(ctx context.Context, userID, current, size int) (port.FollowNotificationResult, error) {
	session, err := repoSession(r.engine, ctx, "follow.notifications")
	if err != nil {
		return port.FollowNotificationResult{}, err
	}
	const fromWhere = `
		FROM t_author_publish_event event
		JOIN t_user_follow follow ON follow.author_id = event.author_id AND follow.follower_id = ? AND event.id > follow.start_event_id
		JOIN t_user_info author ON author.id = event.author_id AND author.is_disable = 0
		LEFT JOIN t_article article ON event.content_type = 'article' AND article.id = event.content_id
			AND article.is_delete = 0 AND article.status = 1 AND article.moderation_status = 'visible'
		LEFT JOIN t_talk talk ON event.content_type = 'talk' AND talk.id = event.content_id
			AND talk.status = 1 AND talk.moderation_status = 'visible'
		WHERE ((event.content_type = 'article' AND article.id IS NOT NULL) OR (event.content_type = 'talk' AND talk.id IS NOT NULL))`
	var result port.FollowNotificationResult
	if _, err := session.SQL("SELECT count(1)"+fromWhere, userID).Get(&result.Count); err != nil {
		return port.FollowNotificationResult{}, apperrors.Unavailable("follow.notifications.count", err)
	}
	if _, err := session.SQL("SELECT count(1)"+fromWhere+" AND event.id > follow.last_read_event_id", userID).Get(&result.UnreadCount); err != nil {
		return port.FollowNotificationResult{}, apperrors.Unavailable("follow.notifications.unread", err)
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
		IsRead       bool      `xorm:"is_read"`
	}
	if err := session.SQL(`
		SELECT event.id AS event_id, event.content_type, event.content_id, event.author_id,
		       author.handle AS author_handle, author.nickname AS author_name, author.avatar AS author_avatar,
		       COALESCE(article.article_title, '') AS title,
		       CASE WHEN event.content_type = 'article' THEN COALESCE(SUBSTR(article.article_content, 1, 240), '') ELSE COALESCE(SUBSTR(talk.content, 1, 240), '') END AS excerpt,
		       COALESCE(article.article_cover, '') AS cover,
		       COALESCE(talk.images, '') AS images,
		       event.published_at, (event.id <= follow.last_read_event_id) AS is_read`+fromWhere+`
		ORDER BY event.published_at DESC, event.id DESC LIMIT ? OFFSET ?`, userID, page, offset).Find(&rows); err != nil {
		return port.FollowNotificationResult{}, apperrors.Unavailable("follow.notifications", err)
	}
	result.Records = make([]port.FollowNotification, 0, len(rows))
	for _, row := range rows {
		result.Records = append(result.Records, port.FollowNotification{
			FollowFeedItem: port.FollowFeedItem{
				EventId: row.EventId, ContentType: row.ContentType, ContentId: row.ContentId,
				Author: port.PublicAuthor{Id: row.AuthorId, Handle: row.AuthorHandle, Nickname: row.AuthorName, Avatar: row.AuthorAvatar},
				Title:  row.Title, Excerpt: row.Excerpt, Cover: row.Cover, Images: decodeTalkImages(row.Images), PublishedAt: row.PublishedAt,
			},
			Read: row.IsRead,
		})
	}
	return result, nil
}

func (r *MyFollowRepo) UnreadNotificationCount(ctx context.Context, userID int) (int, error) {
	session, err := repoSession(r.engine, ctx, "follow.notifications.unread_count")
	if err != nil {
		return 0, err
	}
	const queryText = `
		SELECT count(1)
		FROM t_author_publish_event event
		JOIN t_user_follow follow ON follow.author_id = event.author_id AND follow.follower_id = ?
			AND event.id > follow.start_event_id AND event.id > follow.last_read_event_id
		JOIN t_user_info author ON author.id = event.author_id AND author.is_disable = 0
		LEFT JOIN t_article article ON event.content_type = 'article' AND article.id = event.content_id
			AND article.is_delete = 0 AND article.status = 1 AND article.moderation_status = 'visible'
		LEFT JOIN t_talk talk ON event.content_type = 'talk' AND talk.id = event.content_id
			AND talk.status = 1 AND talk.moderation_status = 'visible'
		WHERE ((event.content_type = 'article' AND article.id IS NOT NULL) OR (event.content_type = 'talk' AND talk.id IS NOT NULL))`
	var count int
	if _, err := session.SQL(queryText, userID).Get(&count); err != nil {
		return 0, apperrors.Unavailable("follow.notifications.unread_count", err)
	}
	return count, nil
}

func (r *MyFollowRepo) MarkNotificationsRead(ctx context.Context, userID int) error {
	if userID <= 0 {
		return apperrors.Invalid("follow.notifications.read", "invalid user")
	}
	session, err := repoSession(r.engine, ctx, "follow.notifications.read")
	if err != nil {
		return err
	}
	if _, err := session.Exec(`
		UPDATE t_user_follow
		SET last_read_event_id = (SELECT COALESCE(MAX(id), 0) FROM t_author_publish_event), updated_at = CURRENT_TIMESTAMP
		WHERE follower_id = ?`, userID); err != nil {
		return apperrors.Unavailable("follow.notifications.read", err)
	}
	return nil
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
