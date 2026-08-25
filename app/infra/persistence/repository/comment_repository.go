package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
	"strings"
)

type CommentRepo interface {
	ListComments(current, size int, vo *model.CommentVO) []*model.CommentDTO
	ListReplies(commentIds []int) []*model.ReplyDTO
	ListTopSixComments() []*model.CommentDTO
	CountComments(vo *model.ConditionVO) (count int64)
	ListCommentsAdmin(current, size int, vo *model.ConditionVO) []*model.CommentAdminDTO
	ListCommentCountByTypeAndTopicIds(ty int, topicIds []int) []*model.CommentCountDTO
	ListCommentCountByTypeAndTopicId(ty, topicId int) (commentCountDTO model.CommentCountDTO)
}

type MyCommentRepo struct{}

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

func (c *MyCommentRepo) ListComments(current, size int, vo *model.CommentVO) []*model.CommentDTO {
	limit, offset := pgsql.Page(current, size)
	query := "SELECT c.id, c.user_id, u.nickname, u.avatar, u.website, c.comment_content, c.create_time FROM t_comment c JOIN t_user_info u ON c.user_id = u.id WHERE c.type = ? AND c.is_review = 1 AND c.parent_id = 0"
	args := []interface{}{vo.Type}
	if vo.TopicId != "" {
		query += " AND c.topic_id = ?"
		args = append(args, vo.TopicId)
	}
	query += " ORDER BY c.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var comments []*model.CommentDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&comments); err != nil {
		zlog.Error("list comments: " + err.Error())
	}
	return comments
}

func (c *MyCommentRepo) ListReplies(commentIds []int) []*model.ReplyDTO {
	if len(commentIds) == 0 {
		return []*model.ReplyDTO{}
	}
	query := "SELECT * FROM (SELECT c.id, c.parent_id, c.user_id, u.nickname, u.avatar, u.website, c.reply_user_id, r.nickname AS reply_nickname, r.website AS reply_website, c.comment_content, c.create_time, row_number() OVER (PARTITION BY parent_id ORDER BY c.create_time ASC) row_num FROM t_comment c JOIN t_user_info u ON c.user_id = u.id JOIN t_user_info r ON c.reply_user_id = r.id WHERE c.is_review = 1 AND parent_id IN (" + placeholders(len(commentIds)) + ") ORDER BY c.create_time DESC) t"
	var replies []*model.ReplyDTO
	if err := ormInit.GetEngine().SQL(query, intArgs(commentIds)...).Find(&replies); err != nil {
		zlog.Error("list comment replies: " + err.Error())
	}
	return replies
}

func (c *MyCommentRepo) ListTopSixComments() []*model.CommentDTO {
	var comments []*model.CommentDTO
	if err := ormInit.GetEngine().SQL(pgsql.ListTopSixComments).Find(&comments); err != nil {
		zlog.Error("list top comments: " + err.Error())
	}
	return comments
}

func commentFilters(vo *model.ConditionVO) (string, []interface{}) {
	query := " WHERE 1 = 1"
	args := make([]interface{}, 0, 3)
	if vo.Type != 0 {
		query += " AND c.type = ?"
		args = append(args, vo.Type)
	}
	if vo.IsReview != 0 {
		query += " AND c.is_review = ?"
		args = append(args, vo.IsReview)
	}
	if vo.Keywords != "" {
		query += " AND u.nickname LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(vo.Keywords))
	}
	return query, args
}

func (c *MyCommentRepo) CountComments(vo *model.ConditionVO) (count int64) {
	filters, args := commentFilters(vo)
	query := "SELECT count(1) FROM t_comment c LEFT JOIN t_user_info u ON c.user_id = u.id" + filters
	if _, err := ormInit.GetEngine().SQL(query, args...).Get(&count); err != nil {
		zlog.Error("count comments: " + err.Error())
	}
	return count
}

func (c *MyCommentRepo) ListCommentsAdmin(current, size int, vo *model.ConditionVO) []*model.CommentAdminDTO {
	limit, offset := pgsql.Page(current, size)
	filters, args := commentFilters(vo)
	query := "SELECT c.id, u.avatar, u.nickname, r.nickname AS reply_nickname, a.article_title, c.comment_content, c.type, c.is_review, c.create_time FROM t_comment c LEFT JOIN t_article a ON c.topic_id = a.id LEFT JOIN t_user_info u ON c.user_id = u.id LEFT JOIN t_user_info r ON c.reply_user_id = r.id" + filters + " ORDER BY c.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var comments []*model.CommentAdminDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&comments); err != nil {
		zlog.Error("list admin comments: " + err.Error())
	}
	return comments
}

func (c *MyCommentRepo) ListCommentCountByTypeAndTopicIds(ty int, topicIds []int) []*model.CommentCountDTO {
	if len(topicIds) == 0 {
		return []*model.CommentCountDTO{}
	}
	query := "SELECT topic_id AS id, COUNT(1) AS comment_count FROM t_comment WHERE type = ? AND topic_id IN (" + placeholders(len(topicIds)) + ") GROUP BY topic_id"
	args := []interface{}{ty}
	args = append(args, intArgs(topicIds)...)
	var counts []*model.CommentCountDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&counts); err != nil {
		zlog.Error("count comments by topics: " + err.Error())
	}
	return counts
}

func (c *MyCommentRepo) ListCommentCountByTypeAndTopicId(ty, topicId int) (commentCountDTO model.CommentCountDTO) {
	query := "SELECT topic_id AS id, COUNT(1) AS comment_count FROM t_comment WHERE type = ? AND topic_id = ? GROUP BY topic_id"
	if _, err := ormInit.GetEngine().SQL(query, ty, topicId).Get(&commentCountDTO); err != nil {
		zlog.Error("count comments by topic: " + err.Error())
	}
	return commentCountDTO
}
