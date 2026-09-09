package repository

import (
	"benetnasch/internal/domain/entity"
	apperrors "benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/infrastructure/persistence/postgres/orm"
	"benetnasch/internal/infrastructure/persistence/postgres/query"
	"context"
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
	query := "SELECT c.id, c.user_id, u.nickname, u.avatar, u.website, c.comment_content, c.create_time FROM t_comment c JOIN t_user_info u ON c.user_id = u.id WHERE c.type = ? AND c.is_review = 1 AND c.parent_id = 0"
	args := []interface{}{filter.Type}
	countQuery := "SELECT count(0) FROM t_comment c WHERE c.type = ? AND c.is_review = 1 AND c.parent_id = 0"
	countArgs := []interface{}{filter.Type}
	if filter.TopicID != nil {
		query += " AND c.topic_id = ?"
		args = append(args, *filter.TopicID)
		countQuery += " AND c.topic_id = ?"
		countArgs = append(countArgs, *filter.TopicID)
	}
	query += " ORDER BY c.id DESC LIMIT ? OFFSET ?"
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

func (c *MyCommentRepo) ListReplies(ctx context.Context, commentIDs []int) ([]*port.Reply, error) {
	if len(commentIDs) == 0 {
		return []*port.Reply{}, nil
	}
	session, err := c.commentSession(ctx)
	if err != nil {
		return nil, err
	}
	query := "SELECT * FROM (SELECT c.id, c.parent_id, c.user_id, u.nickname, u.avatar, u.website, c.reply_user_id, r.nickname AS reply_nickname, r.website AS reply_website, c.comment_content, c.create_time, row_number() OVER (PARTITION BY parent_id ORDER BY c.create_time ASC) row_num FROM t_comment c JOIN t_user_info u ON c.user_id = u.id JOIN t_user_info r ON c.reply_user_id = r.id WHERE c.is_review = 1 AND parent_id IN (" + placeholders(len(commentIDs)) + ") ORDER BY c.create_time DESC) t"
	var replies []*port.Reply
	if err := session.SQL(query, intArgs(commentIDs)...).Find(&replies); err != nil {
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
		query += " AND u.nickname LIKE ? ESCAPE '\\'"
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
	query := "SELECT c.id, u.avatar, u.nickname, r.nickname AS reply_nickname, a.article_title, c.comment_content, c.type, c.is_review, c.create_time FROM t_comment c LEFT JOIN t_article a ON c.topic_id = a.id LEFT JOIN t_user_info u ON c.user_id = u.id LEFT JOIN t_user_info r ON c.reply_user_id = r.id" + filters + " ORDER BY c.id DESC LIMIT ? OFFSET ?"
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

func (c *MyCommentRepo) Create(ctx context.Context, comment entity.TComment) error {
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.Insert(&comment); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "comment.create", err)
		}
		return nil
	})
}

func (c *MyCommentRepo) Review(ctx context.Context, ids []int, review int) error {
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		for _, id := range ids {
			comment := entity.TComment{Id: id, IsReview: review}
			if _, err := session.ID(id).MustCols("is_review").Update(&comment); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "comment.review", err)
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
