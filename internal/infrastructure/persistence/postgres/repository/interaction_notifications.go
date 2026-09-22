package repository

import (
	"fmt"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"xorm.io/xorm"
)

func recordCommentNotification(session *xorm.Session, commentID int) error {
	if session == nil || commentID <= 0 {
		return nil
	}
	var comment struct {
		UserId      int `xorm:"user_id"`
		ReplyUserId int `xorm:"reply_user_id"`
		TopicId     int `xorm:"topic_id"`
		Type        int `xorm:"type"`
		IsDelete    int `xorm:"is_delete"`
		IsReview    int `xorm:"is_review"`
	}
	found, err := session.SQL(`SELECT user_id, reply_user_id, topic_id, type, is_delete, is_review FROM t_comment WHERE id = ?`, commentID).Get(&comment)
	if err != nil {
		return apperrors.Unavailable("notification.comment.lookup", err)
	}
	if !found || comment.IsDelete != 0 || comment.IsReview != 1 {
		return nil
	}

	recipientID := 0
	notificationType := port.NotificationTypeComment
	contentType := ""
	contentID := 0
	switch {
	case comment.ReplyUserId > 0:
		recipientID = comment.ReplyUserId
		notificationType = port.NotificationTypeReply
		contentType, contentID = commentContentTarget(comment.Type, comment.TopicId)
	case comment.Type == 1 && comment.TopicId > 0:
		contentType = port.FollowContentArticle
		contentID = comment.TopicId
		if _, err := session.SQL(`SELECT user_id FROM t_article WHERE id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, contentID).Get(&recipientID); err != nil {
			return apperrors.Unavailable("notification.comment.article", err)
		}
	case comment.Type == 5 && comment.TopicId > 0:
		contentType = port.FollowContentTalk
		contentID = comment.TopicId
		if _, err := session.SQL(`SELECT user_id FROM t_talk WHERE id = ? AND status = 1 AND moderation_status = 'visible'`, contentID).Get(&recipientID); err != nil {
			return apperrors.Unavailable("notification.comment.talk", err)
		}
	case comment.Type == 6 && comment.TopicId > 0:
		contentType = port.FollowContentCollection
		contentID = comment.TopicId
		if _, err := session.SQL(`SELECT c.user_id FROM t_collection c
			JOIN t_user_info owner ON owner.id = c.user_id AND owner.is_disable = 0
			WHERE c.id = ? AND c.is_delete = 0 AND c.moderation_status = 'visible'
			  AND c.visibility IN ('public', 'unlisted')`, contentID).Get(&recipientID); err != nil {
			return apperrors.Unavailable("notification.comment.collection", err)
		}
	}
	if recipientID <= 0 || recipientID == comment.UserId || contentType == "" || contentID <= 0 {
		return nil
	}
	return insertInteractionNotification(session, recipientID, comment.UserId, notificationType, contentType, contentID, commentID, fmt.Sprintf("comment:%d", commentID))
}

func recordReactionNotification(session *xorm.Session, articleID, actorID int, reaction string) error {
	if session == nil || articleID <= 0 || actorID <= 0 {
		return nil
	}
	if reaction != port.NotificationTypeLike && reaction != port.NotificationTypeFavorite {
		return nil
	}
	var article struct {
		AuthorId int `xorm:"author_id"`
	}
	found, err := session.SQL(`SELECT user_id AS author_id FROM t_article WHERE id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, articleID).Get(&article)
	if err != nil {
		return apperrors.Unavailable("notification.reaction.article", err)
	}
	if !found || article.AuthorId <= 0 || article.AuthorId == actorID {
		return nil
	}
	return insertInteractionNotification(
		session,
		article.AuthorId,
		actorID,
		reaction,
		port.FollowContentArticle,
		articleID,
		0,
		fmt.Sprintf("reaction:%s:%d:%d", reaction, articleID, actorID),
	)
}

func insertInteractionNotification(session *xorm.Session, recipientID, actorID int, notificationType, contentType string, contentID, commentID int, dedupeKey string) error {
	if _, err := session.Exec(`
		INSERT INTO t_user_notification (recipient_id, actor_id, type, content_type, content_id, comment_id, dedupe_key)
		SELECT ?, ?, ?, ?, ?, ?, ?
		FROM t_user_info recipient
		WHERE recipient.id = ? AND recipient.is_disable = 0 AND recipient.notify_interaction = 1
		ON CONFLICT (recipient_id, dedupe_key) DO NOTHING`,
		recipientID, actorID, notificationType, contentType, contentID, commentID, dedupeKey, recipientID,
	); err != nil {
		return apperrors.Unavailable("notification.create", err)
	}
	return nil
}

func commentContentTarget(commentType, topicID int) (string, int) {
	if topicID <= 0 {
		return "", 0
	}
	switch commentType {
	case 1:
		return port.FollowContentArticle, topicID
	case 5:
		return port.FollowContentTalk, topicID
	case 6:
		return port.FollowContentCollection, topicID
	default:
		return "", 0
	}
}
