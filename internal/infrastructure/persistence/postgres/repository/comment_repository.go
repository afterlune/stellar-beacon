package repository

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"strings"

	"xorm.io/xorm"
)

var _ port.CommentRepository = (*MyCommentRepo)(nil)

type MyCommentRepo struct {
	engine *xorm.Engine
}

func NewCommentRepo(engine *xorm.Engine) *MyCommentRepo {
	return &MyCommentRepo{engine: engine}
}

func (c *MyCommentRepo) commentSession(ctx context.Context) (*xorm.Session, error) {
	return repoSession(c.engine, ctx, "comment")
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func intArgs(values []int) []interface{} {
	args := make([]interface{}, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	return args
}

func (c *MyCommentRepo) ListComments(ctx context.Context, filter port.CommentFilter) ([]*port.Comment, int, error) {
	session, err := c.commentSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	query := `SELECT c.id, c.user_id, u.nickname, u.avatar, u.website, c.comment_content, c.create_time, c.is_top,
		COALESCE((SELECT count(1) FROM t_comment_reaction reaction WHERE reaction.comment_id = c.id AND reaction.reaction = 'like'), 0) AS like_count,
		EXISTS (SELECT 1 FROM t_comment_reaction reaction WHERE reaction.comment_id = c.id
			AND reaction.user_info_id = ? AND reaction.reaction = 'like') AS liked
		FROM t_comment c JOIN t_user_info u ON c.user_id = u.id
		WHERE c.type = ? AND c.is_review = 1 AND c.is_delete = 0 AND c.parent_id = 0`
	args := []interface{}{filter.ViewerID, filter.Type}
	countQuery := "SELECT count(0) FROM t_comment c WHERE c.type = ? AND c.is_review = 1 AND c.is_delete = 0 AND c.parent_id = 0"
	countArgs := []interface{}{filter.Type}
	if filter.TopicID != nil {
		query += " AND c.topic_id = ?"
		args = append(args, *filter.TopicID)
		countQuery += " AND c.topic_id = ?"
		countArgs = append(countArgs, *filter.TopicID)
	}
	query += " ORDER BY c.is_top DESC, c.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var count int
	if _, err := session.SQL(countQuery, countArgs...).Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.count", err)
	}
	var comments []*port.Comment
	if err := session.SQL(query, args...).Find(&comments); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.list", err)
	}
	return comments, count, nil
}

func (c *MyCommentRepo) ResolveCommentPage(ctx context.Context, commentType, topicID, commentID, size int) (int, error) {
	if commentType <= 0 || topicID <= 0 || commentID <= 0 {
		return 1, nil
	}
	if size <= 0 {
		size = 7
	}
	session, err := c.commentSession(ctx)
	if err != nil {
		return 0, err
	}
	var focus struct {
		Id       int `xorm:"id"`
		ParentId int `xorm:"parent_id"`
	}
	found, err := session.SQL(`
		SELECT id, parent_id
		FROM t_comment
		WHERE id = ? AND type = ? AND topic_id = ? AND is_delete = 0 AND is_review = 1`,
		commentID, commentType, topicID).Get(&focus)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.focus", err)
	}
	if !found {
		return 1, nil
	}
	rootID := focus.Id
	if focus.ParentId > 0 {
		rootID = focus.ParentId
	}
	var newer int
	if _, err := session.SQL(`
		SELECT count(1)
		FROM t_comment
		WHERE type = ? AND topic_id = ? AND parent_id = 0 AND is_delete = 0 AND is_review = 1 AND id > ?`,
		commentType, topicID, rootID).Get(&newer); err != nil {
		return 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.focus_rank", err)
	}
	return newer/size + 1, nil
}
func (c *MyCommentRepo) ListReplies(ctx context.Context, commentIDs []int, viewerID int) ([]*port.Reply, error) {
	if len(commentIDs) == 0 {
		return []*port.Reply{}, nil
	}
	session, err := c.commentSession(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM (SELECT c.id, c.parent_id, c.user_id, u.nickname, u.avatar, u.website,
			c.reply_user_id, r.nickname AS reply_nickname, r.website AS reply_website, c.comment_content, c.create_time,
			COALESCE((SELECT count(1) FROM t_comment_reaction reaction WHERE reaction.comment_id = c.id AND reaction.reaction = 'like'), 0) AS like_count,
			EXISTS (SELECT 1 FROM t_comment_reaction reaction WHERE reaction.comment_id = c.id
				AND reaction.user_info_id = ? AND reaction.reaction = 'like') AS liked,
			row_number() OVER (PARTITION BY parent_id ORDER BY c.create_time ASC) row_num
		FROM t_comment c JOIN t_user_info u ON c.user_id = u.id JOIN t_user_info r ON c.reply_user_id = r.id
		WHERE c.is_review = 1 AND c.is_delete = 0 AND parent_id IN (` + placeholders(len(commentIDs)) + `)
		ORDER BY c.create_time DESC) t`
	var replies []*port.Reply
	args := []interface{}{viewerID}
	args = append(args, intArgs(commentIDs)...)
	if err := session.SQL(query, args...).Find(&replies); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "comment.replies", err)
	}
	return replies, nil
}

func (c *MyCommentRepo) ListTopSixComments(ctx context.Context) ([]*port.Comment, error) {
	session, err := c.commentSession(ctx)
	if err != nil {
		return nil, err
	}
	var comments []*port.Comment
	if err := session.SQL(pgsql.ListTopSixComments).Find(&comments); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "comment.top_six", err)
	}
	return comments, nil
}

func commentFilters(filter port.CommentFilter) (string, []interface{}) {
	query := " WHERE 1 = 1"
	args := make([]interface{}, 0, 3)
	if filter.Type != 0 {
		query += " AND c.type = ?"
		args = append(args, filter.Type)
	}
	if filter.IsReview != 0 {
		query += " AND c.is_review = ?"
		args = append(args, filter.IsReview)
	}
	if filter.Keywords != "" {
		query += " AND c.comment_content LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(filter.Keywords))
	}
	return query, args
}

func (c *MyCommentRepo) CountComments(ctx context.Context, filter port.CommentFilter) (int64, error) {
	session, err := c.commentSession(ctx)
	if err != nil {
		return 0, err
	}
	filters, args := commentFilters(filter)
	var count int64
	if _, err := session.SQL("SELECT count(1) FROM t_comment c LEFT JOIN t_user_info u ON c.user_id = u.id"+filters, args...).Get(&count); err != nil {
		return 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.count_admin", err)
	}
	return count, nil
}

func (c *MyCommentRepo) ListCommentsAdmin(ctx context.Context, filter port.CommentFilter) ([]*port.CommentAdmin, error) {
	session, err := c.commentSession(ctx)
	if err != nil {
		return nil, err
	}
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	filters, args := commentFilters(filter)
	query := "SELECT c.id, u.avatar, u.nickname, r.nickname AS reply_nickname, COALESCE(a.article_title, collection.title, '') AS article_title, c.comment_content, c.type, c.is_review, c.create_time FROM t_comment c LEFT JOIN t_article a ON c.type = 1 AND c.topic_id = a.id LEFT JOIN t_collection collection ON c.type = 6 AND c.topic_id = collection.id LEFT JOIN t_user_info u ON c.user_id = u.id LEFT JOIN t_user_info r ON c.reply_user_id = r.id" + filters + " ORDER BY c.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var comments []*port.CommentAdmin
	if err := session.SQL(query, args...).Find(&comments); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "comment.list_admin", err)
	}
	return comments, nil
}

func (c *MyCommentRepo) ListCommentCountsByTypeAndTopicIDs(ctx context.Context, commentType int, topicIDs []int) ([]*port.CommentCount, error) {
	if len(topicIDs) == 0 {
		return []*port.CommentCount{}, nil
	}
	session, err := c.commentSession(ctx)
	if err != nil {
		return nil, err
	}
	query := "SELECT topic_id AS id, COUNT(1) AS comment_count FROM t_comment WHERE type = ? AND topic_id IN (" + placeholders(len(topicIDs)) + ") GROUP BY topic_id"
	args := []interface{}{commentType}
	args = append(args, intArgs(topicIDs)...)
	var counts []*port.CommentCount
	if err := session.SQL(query, args...).Find(&counts); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "comment.count_topics", err)
	}
	return counts, nil
}

func (c *MyCommentRepo) ListCommentCountByTypeAndTopicID(ctx context.Context, commentType, topicID int) (port.CommentCount, error) {
	session, err := c.commentSession(ctx)
	if err != nil {
		return port.CommentCount{}, err
	}
	query := "SELECT topic_id AS id, COUNT(1) AS comment_count FROM t_comment WHERE type = ? AND topic_id = ? GROUP BY topic_id"
	var count port.CommentCount
	found, err := session.SQL(query, commentType, topicID).Get(&count)
	if err != nil {
		return port.CommentCount{}, apperrors.Wrap(apperrors.KindUnavailable, "comment.count_topic", err)
	}
	if !found {
		return port.CommentCount{}, nil
	}
	return count, nil
}

func (c *MyCommentRepo) ValidateTarget(ctx context.Context, commentType, topicID int) error {
	session, err := c.commentSession(ctx)
	if err != nil {
		return err
	}
	query := ""
	switch commentType {
	case 1:
		query = "SELECT id FROM t_article WHERE id = ?"
	case 5:
		query = "SELECT id FROM t_talk WHERE id = ?"
	case 6:
		query = `SELECT c.id FROM t_collection c
			JOIN t_user_info owner ON owner.id = c.user_id AND owner.is_disable = 0
			WHERE c.id = ? AND c.is_delete = 0 AND c.moderation_status = 'visible'
			  AND c.visibility IN ('public', 'unlisted')`
	default:
		return nil
	}
	var id int
	found, err := session.SQL(query, topicID).Get(&id)
	if err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "comment.validate_target", err)
	}
	if !found {
		return apperrors.Invalid("comment.validate_target", "target does not exist")
	}
	return nil
}

func (c *MyCommentRepo) ValidateReply(ctx context.Context, commentType, parentID, replyUserID int) error {
	session, err := c.commentSession(ctx)
	if err != nil {
		return err
	}
	var parent entity.TComment
	found, err := session.SQL("SELECT id, parent_id, type FROM t_comment WHERE id = ?", parentID).Get(&parent)
	if err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "comment.validate_parent", err)
	}
	if !found || parent.ParentId != 0 || parent.Type != commentType {
		return apperrors.Invalid("comment.validate_parent", "invalid parent comment")
	}
	var user entity.TUserInfo
	found, err = session.SQL("SELECT id FROM t_user_info WHERE id = ?", replyUserID).Get(&user)
	if err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "comment.validate_reply_user", err)
	}
	if !found {
		return apperrors.Invalid("comment.validate_reply_user", "reply user does not exist")
	}
	return nil
}

func (c *MyCommentRepo) GetByID(ctx context.Context, commentID int) (entity.TComment, error) {
	session, err := c.commentSession(ctx)
	if err != nil {
		return entity.TComment{}, err
	}
	var comment entity.TComment
	found, err := session.ID(commentID).Get(&comment)
	if err != nil {
		return entity.TComment{}, apperrors.Wrap(apperrors.KindUnavailable, "comment.get", err)
	}
	if !found {
		return entity.TComment{}, apperrors.NotFound("comment.get")
	}
	return comment, nil
}

func ownedCollectionComment(session *xorm.Session, userID, collectionID, commentID int) (entity.TComment, error) {
	var owned bool
	if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_collection
		WHERE id = ? AND user_id = ? AND is_delete = 0)`, collectionID, userID).Get(&owned); err != nil {
		return entity.TComment{}, apperrors.Wrap(apperrors.KindUnavailable, "comment.moderation.collection", err)
	}
	if !owned {
		return entity.TComment{}, apperrors.NotFound("comment.moderation.collection")
	}
	var comment entity.TComment
	found, err := session.ID(commentID).Get(&comment)
	if err != nil {
		return entity.TComment{}, apperrors.Wrap(apperrors.KindUnavailable, "comment.moderation.lookup", err)
	}
	if !found {
		return entity.TComment{}, apperrors.NotFound("comment.moderation.comment")
	}
	if comment.Type != 6 || comment.TopicId != collectionID || comment.IsReview != 1 || comment.IsDelete != 0 {
		return entity.TComment{}, apperrors.Invalid("comment.moderation.target", "comment is not moderatable")
	}
	return comment, nil
}

func (c *MyCommentRepo) SetPinned(ctx context.Context, userID, collectionID, commentID int, pinned bool) error {
	if userID <= 0 || collectionID <= 0 || commentID <= 0 {
		return apperrors.Invalid("comment.moderation.pin", "invalid target")
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		comment, err := ownedCollectionComment(session, userID, collectionID, commentID)
		if err != nil {
			return err
		}
		if comment.ParentId != 0 {
			return apperrors.Invalid("comment.moderation.pin", "only root comments can be pinned")
		}
		if pinned {
			if _, err := session.Exec(`UPDATE t_comment SET is_top = 0, update_time = CURRENT_TIMESTAMP
				WHERE type = 6 AND topic_id = ? AND parent_id = 0 AND is_top = 1`, collectionID); err != nil {
				return apperrors.Unavailable("comment.moderation.unpin_previous", err)
			}
		}
		value := 0
		if pinned {
			value = 1
		}
		if _, err := session.Exec(`UPDATE t_comment SET is_top = ?, update_time = CURRENT_TIMESTAMP WHERE id = ?`, value, commentID); err != nil {
			return apperrors.Unavailable("comment.moderation.pin_update", err)
		}
		return nil
	})
}

func (c *MyCommentRepo) SoftDeleteOwned(ctx context.Context, userID, collectionID, commentID int) error {
	if userID <= 0 || collectionID <= 0 || commentID <= 0 {
		return apperrors.Invalid("comment.moderation.delete", "invalid target")
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		comment, err := ownedCollectionComment(session, userID, collectionID, commentID)
		if err != nil {
			return err
		}
		if comment.ParentId == 0 {
			if _, err := session.Exec(`UPDATE t_comment SET is_delete = 1, is_top = 0, update_time = CURRENT_TIMESTAMP
				WHERE type = 6 AND topic_id = ? AND (id = ? OR parent_id = ?)`, collectionID, commentID, commentID); err != nil {
				return apperrors.Unavailable("comment.moderation.delete_thread", err)
			}
			return nil
		}
		if _, err := session.Exec(`UPDATE t_comment SET is_delete = 1, update_time = CURRENT_TIMESTAMP WHERE id = ?`, commentID); err != nil {
			return apperrors.Unavailable("comment.moderation.delete_reply", err)
		}
		return nil
	})
}

// Create inserts the comment and returns its generated id so callers can use it
// without a follow-up query.
func (c *MyCommentRepo) Create(ctx context.Context, comment entity.TComment) (int, error) {
	err := ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.Insert(&comment); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "comment.create", err)
		}
		if comment.IsReview == 1 {
			if err := recordCommentNotification(session, comment.Id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return comment.Id, nil
}

func (c *MyCommentRepo) MarkNotificationDispatched(ctx context.Context, commentID int) error {
	if commentID <= 0 {
		return nil
	}
	session, err := c.commentSession(ctx)
	if err != nil {
		return err
	}
	if _, err := session.Exec(`UPDATE t_comment SET notification_dispatched_at = CURRENT_TIMESTAMP WHERE id = ? AND notification_dispatched_at IS NULL`, commentID); err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "comment.notification_dispatched", err)
	}
	return nil
}
func (c *MyCommentRepo) Review(ctx context.Context, ids []int, review int) error {
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		for _, id := range ids {
			comment := entity.TComment{Id: id, IsReview: review}
			if _, err := session.ID(id).MustCols("is_review").Update(&comment); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "comment.review", err)
			}
			if review == 1 {
				if err := recordCommentNotification(session, id); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (c *MyCommentRepo) Delete(ctx context.Context, ids []int) error {
	session, err := c.commentSession(ctx)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if _, err := session.In("id", ids).Delete(&entity.TComment{}); err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "comment.delete", err)
	}
	return nil
}
