package repository

import (
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"xorm.io/xorm"
)

func recordAuthorPublishEvent(session *xorm.Session, contentType string, contentID, authorID int, publishedAt time.Time) error {
	if session == nil || contentID <= 0 || authorID <= 0 {
		return nil
	}
	if contentType != port.FollowContentArticle && contentType != port.FollowContentTalk {
		return nil
	}
	if publishedAt.IsZero() {
		publishedAt = time.Now()
	}
	if _, err := session.Exec(`
		INSERT INTO t_author_publish_event (author_id, content_type, content_id, published_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (content_type, content_id) DO NOTHING`, authorID, contentType, contentID, publishedAt); err != nil {
		return apperrors.Unavailable("follow.event.record", err)
	}
	return nil
}

func recordArticlePublishEvent(session *xorm.Session, articleID int, publishedAt time.Time) error {
	if session == nil || articleID <= 0 {
		return nil
	}
	var article struct {
		AuthorID         int    `xorm:"author_id"`
		Status           int    `xorm:"status"`
		IsDelete         int    `xorm:"is_delete"`
		ModerationStatus string `xorm:"moderation_status"`
	}
	if _, err := session.SQL(`SELECT user_id AS author_id, status, is_delete, moderation_status FROM t_article WHERE id = ?`, articleID).Get(&article); err != nil {
		return apperrors.Unavailable("follow.event.article", err)
	}
	if article.AuthorID <= 0 || article.Status != 1 || article.IsDelete != 0 || article.ModerationStatus != "visible" {
		return nil
	}
	return recordAuthorPublishEvent(session, port.FollowContentArticle, articleID, article.AuthorID, publishedAt)
}

func recordTalkPublishEvent(session *xorm.Session, talkID int, publishedAt time.Time) error {
	if session == nil || talkID <= 0 {
		return nil
	}
	var talk struct {
		AuthorID         int    `xorm:"author_id"`
		Status           int    `xorm:"status"`
		ModerationStatus string `xorm:"moderation_status"`
	}
	if _, err := session.SQL(`SELECT user_id AS author_id, status, moderation_status FROM t_talk WHERE id = ?`, talkID).Get(&talk); err != nil {
		return apperrors.Unavailable("follow.event.talk", err)
	}
	if talk.AuthorID <= 0 || talk.Status != 1 || talk.ModerationStatus != "visible" {
		return nil
	}
	return recordAuthorPublishEvent(session, port.FollowContentTalk, talkID, talk.AuthorID, publishedAt)
}
