package repository

import (
	"context"
	"fmt"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"xorm.io/xorm"
)

var _ port.CommentReactionRepository = (*MyCommentReactionRepo)(nil)

type MyCommentReactionRepo struct{ engine *xorm.Engine }

func NewCommentReactionRepo(engine *xorm.Engine) *MyCommentReactionRepo {
	return &MyCommentReactionRepo{engine: engine}
}

func commentReactionTarget(session *xorm.Session, commentID int) (int, string, int, error) {
	var comment struct {
		UserID  int `xorm:"user_id"`
		Type    int `xorm:"type"`
		TopicID int `xorm:"topic_id"`
	}
	found, err := session.SQL(`SELECT user_id, type, topic_id FROM t_comment
		WHERE id = ? AND is_delete = 0 AND is_review = 1`, commentID).Get(&comment)
	if err != nil {
		return 0, "", 0, apperrors.Unavailable("comment_reaction.target", err)
	}
	if !found {
		return 0, "", 0, apperrors.Invalid("comment_reaction.target", "comment does not exist")
	}
	contentType := ""
	contentID := comment.TopicID
	switch comment.Type {
	case 1:
		contentType = port.FollowContentArticle
		var exists bool
		if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_article WHERE id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible')`, contentID).Get(&exists); err != nil {
			return 0, "", 0, apperrors.Unavailable("comment_reaction.article", err)
		}
		if !exists {
			return 0, "", 0, apperrors.Invalid("comment_reaction.target", "article is unavailable")
		}
	case 5:
		contentType = port.FollowContentTalk
		var exists bool
		if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_talk WHERE id = ? AND status = 1 AND moderation_status = 'visible')`, contentID).Get(&exists); err != nil {
			return 0, "", 0, apperrors.Unavailable("comment_reaction.talk", err)
		}
		if !exists {
			return 0, "", 0, apperrors.Invalid("comment_reaction.target", "talk is unavailable")
		}
	case 6:
		contentType = port.FollowContentCollection
		var exists bool
		if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_collection c
			JOIN t_user_info owner ON owner.id = c.user_id AND owner.is_disable = 0
			WHERE c.id = ? AND c.is_delete = 0 AND c.moderation_status = 'visible'
			  AND c.visibility IN ('public', 'unlisted'))`, contentID).Get(&exists); err != nil {
			return 0, "", 0, apperrors.Unavailable("comment_reaction.collection", err)
		}
		if !exists {
			return 0, "", 0, apperrors.Invalid("comment_reaction.target", "collection is unavailable")
		}
	}
	return comment.UserID, contentType, contentID, nil
}

func (r *MyCommentReactionRepo) Set(ctx context.Context, commentID, userInfoID int, active bool) (bool, int, error) {
	if commentID <= 0 || userInfoID <= 0 {
		return false, 0, apperrors.Invalid("comment_reaction.set", "invalid reaction")
	}
	likeCount := 0
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		authorID, contentType, contentID, err := commentReactionTarget(session, commentID)
		if err != nil {
			return err
		}
		if active {
			result, err := session.Exec(`INSERT INTO t_comment_reaction (comment_id, user_info_id, reaction)
				VALUES (?, ?, 'like') ON CONFLICT (comment_id, user_info_id) DO NOTHING`, commentID, userInfoID)
			if err != nil {
				return apperrors.Unavailable("comment_reaction.insert", err)
			}
			inserted, err := result.RowsAffected()
			if err != nil {
				return apperrors.Unavailable("comment_reaction.insert.rows", err)
			}
			if inserted > 0 && authorID != userInfoID && contentType != "" && contentID > 0 {
				if err := insertInteractionNotification(session, authorID, userInfoID, port.NotificationTypeLike,
					contentType, contentID, commentID,
					fmt.Sprintf("comment_like:%d:%d", commentID, userInfoID)); err != nil {
					return err
				}
			}
		} else if _, err := session.Exec(`DELETE FROM t_comment_reaction WHERE comment_id = ? AND user_info_id = ?`, commentID, userInfoID); err != nil {
			return apperrors.Unavailable("comment_reaction.delete", err)
		}
		if _, err := session.SQL(`SELECT count(1) FROM t_comment_reaction WHERE comment_id = ? AND reaction = 'like'`, commentID).Get(&likeCount); err != nil {
			return apperrors.Unavailable("comment_reaction.count", err)
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}
	return active, likeCount, nil
}
