package repository

import (
	"context"
	"encoding/json"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"github.com/google/uuid"
	"sort"
	"strconv"
	"strings"

	"xorm.io/xorm"
)

const (
	actionPin     = "pin"
	actionUnpin   = "unpin"
	actionDelete  = "delete"
	actionRestore = "restore"

	maxModerationBatchSize = 200
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
	query := `SELECT c.id, c.user_id, u.nickname, u.avatar, u.website, c.comment_content, c.create_time, c.is_top, c.is_delete,
		COALESCE((SELECT count(1) FROM t_comment_reaction reaction WHERE reaction.comment_id = c.id AND reaction.reaction = 'like'), 0) AS like_count,
		EXISTS (SELECT 1 FROM t_comment_reaction reaction WHERE reaction.comment_id = c.id
			AND reaction.user_info_id = ? AND reaction.reaction = 'like') AS liked
		FROM t_comment c JOIN t_user_info u ON c.user_id = u.id
		WHERE c.type = ? AND c.is_review = 1 AND (c.is_delete = 0 OR (c.type = 6 AND c.is_delete = 1 AND (c.user_id = ? OR EXISTS (SELECT 1 FROM t_comment child WHERE child.parent_id = c.id AND child.is_delete = 1 AND child.user_id = ?)))) AND c.parent_id = 0`
	args := []interface{}{filter.ViewerID, filter.Type, filter.ViewerID, filter.ViewerID}
	countQuery := "SELECT count(0) FROM t_comment c WHERE c.type = ? AND c.is_review = 1 AND (c.is_delete = 0 OR (c.type = 6 AND c.is_delete = 1 AND (c.user_id = ? OR EXISTS (SELECT 1 FROM t_comment child WHERE child.parent_id = c.id AND child.is_delete = 1 AND child.user_id = ?)))) AND c.parent_id = 0"
	countArgs := []interface{}{filter.Type, filter.ViewerID, filter.ViewerID}
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
		IsTop    int `xorm:"is_top"`
	}
	found, err := session.SQL(`
		SELECT id, parent_id, is_top
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
	// The public list orders roots by is_top DESC, id DESC, so the page of the
	// focused comment must be derived from the same ordering. Counting only
	// newer ids would land on the wrong page once a root comment is pinned.
	rootIsTop := focus.IsTop
	if focus.ParentId > 0 {
		var parentTop int
		if _, err := session.SQL(`SELECT is_top FROM t_comment WHERE id = ?`, rootID).Get(&parentTop); err != nil {
			return 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.focus_rank", err)
		}
		rootIsTop = parentTop
	}
	var newer int
	if _, err := session.SQL(`
		SELECT count(1)
		FROM t_comment
		WHERE type = ? AND topic_id = ? AND parent_id = 0 AND is_delete = 0 AND is_review = 1
			AND (is_top > ? OR (is_top = ? AND id > ?))`,
		commentType, topicID, rootIsTop, rootIsTop, rootID).Get(&newer); err != nil {
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
			c.reply_user_id, r.nickname AS reply_nickname, r.website AS reply_website, c.comment_content, c.create_time, c.is_delete,
			COALESCE((SELECT count(1) FROM t_comment_reaction reaction WHERE reaction.comment_id = c.id AND reaction.reaction = 'like'), 0) AS like_count,
			EXISTS (SELECT 1 FROM t_comment_reaction reaction WHERE reaction.comment_id = c.id
				AND reaction.user_info_id = ? AND reaction.reaction = 'like') AS liked,
			row_number() OVER (PARTITION BY parent_id ORDER BY c.create_time ASC) row_num
		FROM t_comment c JOIN t_user_info u ON c.user_id = u.id JOIN t_user_info r ON c.reply_user_id = r.id
		WHERE c.is_review = 1 AND (c.is_delete = 0 OR (c.type = 6 AND c.is_delete = 1 AND c.user_id = ?)) AND parent_id IN (` + placeholders(len(commentIDs)) + `)
		ORDER BY c.create_time DESC) t`
	var replies []*port.Reply
	args := []interface{}{viewerID, viewerID}
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
	if filter.CollectionID > 0 {
		query += " AND c.type = 6 AND c.topic_id = ?"
		args = append(args, filter.CollectionID)
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
	query := "SELECT c.id, u.avatar, u.nickname, r.nickname AS reply_nickname, COALESCE(a.article_title, collection.title, '') AS article_title, c.comment_content, c.type, c.is_review, c.is_top, c.is_delete, CASE WHEN c.type = 6 THEN c.topic_id ELSE 0 END AS collection_id, (SELECT count(1) FROM t_comment_report report WHERE report.comment_id = c.id AND report.status = 'pending') AS report_count, c.create_time FROM t_comment c LEFT JOIN t_article a ON c.type = 1 AND c.topic_id = a.id LEFT JOIN t_collection collection ON c.type = 6 AND c.topic_id = collection.id LEFT JOIN t_user_info u ON c.user_id = u.id LEFT JOIN t_user_info r ON c.reply_user_id = r.id" + filters + " ORDER BY c.is_delete ASC, c.id DESC LIMIT ? OFFSET ?"
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

// moderationTarget loads a collection comment the caller may moderate.
// ownerID > 0 enforces reading-list ownership; ownerID == 0 is the
// administrator path and only requires the collection to exist.
func moderationTarget(session *xorm.Session, ownerID, collectionID, commentID int, allowDeleted bool) (entity.TComment, error) {
	if ownerID > 0 {
		var owned bool
		if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_collection
			WHERE id = ? AND user_id = ? AND is_delete = 0)`, collectionID, ownerID).Get(&owned); err != nil {
			return entity.TComment{}, apperrors.Wrap(apperrors.KindUnavailable, "comment.moderation.collection", err)
		}
		if !owned {
			return entity.TComment{}, apperrors.NotFound("comment.moderation.collection")
		}
	} else {
		var exists bool
		if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_collection WHERE id = ?)`, collectionID).Get(&exists); err != nil {
			return entity.TComment{}, apperrors.Wrap(apperrors.KindUnavailable, "comment.moderation.collection", err)
		}
		if !exists {
			return entity.TComment{}, apperrors.NotFound("comment.moderation.collection")
		}
	}
	var comment entity.TComment
	found, err := session.ID(commentID).Get(&comment)
	if err != nil {
		return entity.TComment{}, apperrors.Wrap(apperrors.KindUnavailable, "comment.moderation.lookup", err)
	}
	if !found {
		return entity.TComment{}, apperrors.NotFound("comment.moderation.comment")
	}
	if comment.Type != 6 || comment.TopicId != collectionID || comment.IsReview != 1 {
		return entity.TComment{}, apperrors.Invalid("comment.moderation.target", "comment is not moderatable")
	}
	if !allowDeleted && comment.IsDelete != 0 {
		return entity.TComment{}, apperrors.Invalid("comment.moderation.target", "comment is not moderatable")
	}
	return comment, nil
}

type commentModerationState struct {
	IsTop    int `json:"isTop"`
	IsDelete int `json:"isDelete"`
}

func moderationState(comment entity.TComment) commentModerationState {
	return commentModerationState{IsTop: comment.IsTop, IsDelete: comment.IsDelete}
}

func nullableModerationState(state *commentModerationState) interface{} {
	if state == nil {
		return nil
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		return nil
	}
	return string(encoded)
}

func nullableIDArray(ids []int) interface{} {
	if len(ids) == 0 {
		return nil
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.Itoa(id))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func nullableBatchID(batchID string) interface{} {
	if strings.TrimSpace(batchID) == "" {
		return nil
	}
	return batchID
}

func writeModerationLog(session *xorm.Session, commentID, collectionID, actorID int, actorRole, action string, before, after *commentModerationState, cascadeIDs []int, batchID, reason string) error {
	if _, err := session.Exec(`INSERT INTO t_comment_moderation_log
		(comment_id, collection_id, actor_id, actor_role, action, before_snapshot, after_snapshot, cascade_ids, batch_id, reason)
		VALUES (?, ?, ?, ?, ?, CAST(? AS jsonb), CAST(? AS jsonb), CAST(? AS bigint[]), CAST(? AS uuid), ?)`,
		commentID, collectionID, actorID, actorRole, action,
		nullableModerationState(before), nullableModerationState(after), nullableIDArray(cascadeIDs), nullableBatchID(batchID), reason); err != nil {
		return apperrors.Unavailable("comment.moderation.audit", err)
	}
	return nil
}

// WriteModerationAudit appends one governance event outside a moderation
// transaction. Report and appeal services use it after their own state change.
func (c *MyCommentRepo) WriteModerationAudit(ctx context.Context, entry port.ModerationAuditEntry) error {
	if entry.CommentID <= 0 || entry.CollectionID <= 0 {
		return apperrors.Invalid("comment.moderation.audit", "invalid target")
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		return writeModerationLog(session, entry.CommentID, entry.CollectionID, entry.ActorID, entry.ActorRole, entry.Action, nil, nil, nil, "", entry.Reason)
	})
}

// pinnedRootReset clears the pinned root of a reading list so the partial
// unique index stays satisfiable while another root is pinned.
func pinnedRootReset(session *xorm.Session, collectionID int) error {
	if _, err := session.Exec(`UPDATE t_comment SET is_top = 0, update_time = CURRENT_TIMESTAMP
		WHERE type = 6 AND topic_id = ? AND parent_id = 0 AND is_top = 1`, collectionID); err != nil {
		return apperrors.Unavailable("comment.moderation.unpin_previous", err)
	}
	return nil
}

// hiddenReplyIDs lists the replies a root-comment deletion is about to cascade
// over, so a later restore revives only the replies hidden by that action.
func hiddenReplyIDs(session *xorm.Session, collectionID, rootID int) ([]int, error) {
	var ids []int
	if err := session.SQL(`SELECT id FROM t_comment
		WHERE type = 6 AND topic_id = ? AND parent_id = ? AND is_delete = 0 ORDER BY id`, collectionID, rootID).Find(&ids); err != nil {
		return nil, apperrors.Unavailable("comment.moderation.cascade", err)
	}
	return ids, nil
}

func lastCascadeIDs(session *xorm.Session, collectionID, commentID int) ([]int, error) {
	var raw string
	if _, err := session.SQL(`SELECT COALESCE(array_to_string(cascade_ids, ','), '') FROM t_comment_moderation_log
		WHERE comment_id = ? AND collection_id = ? AND action = 'delete'
		ORDER BY id DESC LIMIT 1`, commentID, collectionID).Get(&raw); err != nil {
		return nil, apperrors.Unavailable("comment.moderation.cascade_lookup", err)
	}
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	ids := make([]int, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		value, convErr := strconv.Atoi(strings.TrimSpace(part))
		if convErr != nil {
			continue
		}
		ids = append(ids, value)
	}
	return ids, nil
}

// applyCommentModeration performs exactly one moderation action on one comment
// inside an existing transaction and appends the matching audit row.
func applyCommentModeration(session *xorm.Session, actorID int, actorRole, action string, ownerID, collectionID, commentID int, batchID, reason string) error {
	switch action {
	case actionPin, actionUnpin:
		comment, err := moderationTarget(session, ownerID, collectionID, commentID, false)
		if err != nil {
			return err
		}
		if comment.ParentId != 0 {
			return apperrors.Invalid("comment.moderation.pin", "only root comments can be pinned")
		}
		pinned := action == actionPin
		if pinned {
			if err := pinnedRootReset(session, collectionID); err != nil {
				return err
			}
		}
		value := 0
		if pinned {
			value = 1
		}
		if _, err := session.Exec(`UPDATE t_comment SET is_top = ?, update_time = CURRENT_TIMESTAMP WHERE id = ?`, value, commentID); err != nil {
			return apperrors.Unavailable("comment.moderation.pin_update", err)
		}
		before := moderationState(comment)
		after := commentModerationState{IsTop: value, IsDelete: 0}
		return writeModerationLog(session, commentID, collectionID, actorID, actorRole, action, &before, &after, nil, batchID, reason)
	case actionDelete:
		comment, err := moderationTarget(session, ownerID, collectionID, commentID, false)
		if err != nil {
			return err
		}
		before := moderationState(comment)
		if comment.ParentId == 0 {
			cascadeIDs, err := hiddenReplyIDs(session, collectionID, commentID)
			if err != nil {
				return err
			}
			if _, err := session.Exec(`UPDATE t_comment SET is_delete = 1, is_top = 0, update_time = CURRENT_TIMESTAMP
				WHERE type = 6 AND topic_id = ? AND (id = ? OR parent_id = ?)`, collectionID, commentID, commentID); err != nil {
				return apperrors.Unavailable("comment.moderation.delete_thread", err)
			}
			after := commentModerationState{IsTop: 0, IsDelete: 1}
			return writeModerationLog(session, commentID, collectionID, actorID, actorRole, action, &before, &after, cascadeIDs, batchID, reason)
		}
		if _, err := session.Exec(`UPDATE t_comment SET is_delete = 1, update_time = CURRENT_TIMESTAMP WHERE id = ?`, commentID); err != nil {
			return apperrors.Unavailable("comment.moderation.delete_reply", err)
		}
		after := commentModerationState{IsTop: 0, IsDelete: 1}
		return writeModerationLog(session, commentID, collectionID, actorID, actorRole, action, &before, &after, nil, batchID, reason)
	case actionRestore:
		comment, err := moderationTarget(session, ownerID, collectionID, commentID, true)
		if err != nil {
			return err
		}
		if comment.IsDelete == 0 {
			return apperrors.Invalid("comment.moderation.restore", "comment is not deleted")
		}
		cascadeIDs, err := lastCascadeIDs(session, collectionID, commentID)
		if err != nil {
			return err
		}
		if _, err := session.Exec(`UPDATE t_comment SET is_delete = 0, update_time = CURRENT_TIMESTAMP WHERE id = ?`, commentID); err != nil {
			return apperrors.Unavailable("comment.moderation.restore", err)
		}
		if len(cascadeIDs) > 0 {
			restoreQuery := `UPDATE t_comment SET is_delete = 0, update_time = CURRENT_TIMESTAMP
				WHERE parent_id = ? AND id IN (` + placeholders(len(cascadeIDs)) + `)`
			restoreArgs := append([]interface{}{commentID}, intArgs(cascadeIDs)...)
			if _, err := session.Exec(append([]interface{}{restoreQuery}, restoreArgs...)...); err != nil {
				return apperrors.Unavailable("comment.moderation.restore_thread", err)
			}
		}
		before := moderationState(comment)
		after := commentModerationState{IsTop: comment.IsTop, IsDelete: 0}
		return writeModerationLog(session, commentID, collectionID, actorID, actorRole, action, &before, &after, cascadeIDs, batchID, reason)
	default:
		return apperrors.Invalid("comment.moderation.action", "unsupported action")
	}
}

func normalizeCommentIDs(commentIDs []int) []int {
	seen := make(map[int]bool, len(commentIDs))
	ids := make([]int, 0, len(commentIDs))
	for _, id := range commentIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func moderationFailureMessage(err error) string {
	switch apperrors.Op(err) {
	case "comment.moderation.collection":
		return "书单不存在或无权治理"
	case "comment.moderation.comment":
		return "评论不存在"
	case "comment.moderation.target":
		return "该评论不可治理（未审核或已删除）"
	case "comment.moderation.pin":
		return "只有根评论可以置顶"
	case "comment.moderation.restore":
		return "该评论未被删除"
	}
	if apperrors.IsKind(err, apperrors.KindNotFound) {
		return "评论不存在"
	}
	if apperrors.IsKind(err, apperrors.KindValidation) {
		return "该评论不可治理"
	}
	return "操作失败，请稍后重试"
}

// runCommentModerationBatch applies one action over a selection so the caller
// gets per-comment outcomes instead of failing the whole request.
func runCommentModerationBatch(c *MyCommentRepo, ctx context.Context, ownerID int, actorRole, action string, collectionID int, commentIDs []int) (port.ModerationBatchResult, error) {
	result := port.ModerationBatchResult{Succeeded: []int{}, Failed: []port.ModerationFailure{}}
	ids := normalizeCommentIDs(commentIDs)
	if ownerID <= 0 || collectionID <= 0 || len(ids) == 0 {
		return result, apperrors.Invalid("comment.moderation.batch", "invalid target")
	}
	if len(ids) > maxModerationBatchSize {
		return result, apperrors.Invalid("comment.moderation.batch", "selection too large")
	}
	batchID := uuid.NewString()
	if err := ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		for _, id := range ids {
			if err := applyCommentModeration(session, ownerID, actorRole, action, ownerID, collectionID, id, batchID, ""); err != nil {
				result.Failed = append(result.Failed, port.ModerationFailure{CommentId: id, Message: moderationFailureMessage(err)})
				continue
			}
			result.Succeeded = append(result.Succeeded, id)
		}
		return nil
	}); err != nil {
		return port.ModerationBatchResult{Succeeded: []int{}, Failed: []port.ModerationFailure{}}, err
	}
	return result, nil
}

func (c *MyCommentRepo) SetPinned(ctx context.Context, userID, collectionID, commentID int, pinned bool) error {
	if userID <= 0 || collectionID <= 0 || commentID <= 0 {
		return apperrors.Invalid("comment.moderation.pin", "invalid target")
	}
	action := actionUnpin
	if pinned {
		action = actionPin
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		return applyCommentModeration(session, userID, "owner", action, userID, collectionID, commentID, "", "")
	})
}

func (c *MyCommentRepo) SoftDeleteOwned(ctx context.Context, userID, collectionID, commentID int) error {
	if userID <= 0 || collectionID <= 0 || commentID <= 0 {
		return apperrors.Invalid("comment.moderation.delete", "invalid target")
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		return applyCommentModeration(session, userID, "owner", actionDelete, userID, collectionID, commentID, "", "")
	})
}

func (c *MyCommentRepo) BatchModerateOwned(ctx context.Context, userID, collectionID int, action string, commentIDs []int) (port.ModerationBatchResult, error) {
	if action != actionDelete && action != actionPin && action != actionUnpin {
		return port.ModerationBatchResult{Succeeded: []int{}, Failed: []port.ModerationFailure{}}, apperrors.Invalid("comment.moderation.batch", "unsupported action")
	}
	return runCommentModerationBatch(c, ctx, userID, "owner", action, collectionID, commentIDs)
}

func (c *MyCommentRepo) RestoreOwned(ctx context.Context, userID, collectionID int, commentIDs []int) (port.ModerationBatchResult, error) {
	return runCommentModerationBatch(c, ctx, userID, "owner", actionRestore, collectionID, commentIDs)
}

func (c *MyCommentRepo) RestoreAsAdmin(ctx context.Context, adminID, collectionID int, commentIDs []int) (port.ModerationBatchResult, error) {
	if adminID <= 0 {
		return port.ModerationBatchResult{Succeeded: []int{}, Failed: []port.ModerationFailure{}}, apperrors.Invalid("comment.moderation.restore", "invalid admin")
	}
	result := port.ModerationBatchResult{Succeeded: []int{}, Failed: []port.ModerationFailure{}}
	ids := normalizeCommentIDs(commentIDs)
	if collectionID <= 0 || len(ids) == 0 {
		return result, apperrors.Invalid("comment.moderation.restore", "invalid target")
	}
	if len(ids) > maxModerationBatchSize {
		return result, apperrors.Invalid("comment.moderation.restore", "selection too large")
	}
	if err := ormInit.WithEngineTx(c.engine, ctx, func(session *xorm.Session) error {
		for _, id := range ids {
			if err := applyCommentModeration(session, adminID, "admin", actionRestore, 0, collectionID, id, "", ""); err != nil {
				result.Failed = append(result.Failed, port.ModerationFailure{CommentId: id, Message: moderationFailureMessage(err)})
				continue
			}
			result.Succeeded = append(result.Succeeded, id)
		}
		return nil
	}); err != nil {
		return port.ModerationBatchResult{Succeeded: []int{}, Failed: []port.ModerationFailure{}}, err
	}
	return result, nil
}

func (c *MyCommentRepo) ListOwnedCollectionComments(ctx context.Context, userID, collectionID, current, size int, keywords string, includeDeleted bool) ([]*port.OwnedComment, int, error) {
	if userID <= 0 || collectionID <= 0 {
		return nil, 0, apperrors.Invalid("comment.moderation.list", "invalid target")
	}
	session, err := c.commentSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	var owned bool
	if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_collection WHERE id = ? AND user_id = ? AND is_delete = 0)`, collectionID, userID).Get(&owned); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.moderation.collection", err)
	}
	if !owned {
		return nil, 0, apperrors.NotFound("comment.moderation.collection")
	}
	limit, offset := pgsql.Page(current, size)
	where := ` WHERE c.type = 6 AND c.topic_id = ? AND c.is_review = 1`
	args := []interface{}{collectionID}
	if !includeDeleted {
		where += ` AND c.is_delete = 0`
	}
	if trimmed := strings.TrimSpace(keywords); trimmed != "" {
		where += ` AND c.comment_content LIKE ? ESCAPE '\'`
		args = append(args, pgsql.ContainsPattern(trimmed))
	}
	countQuery := `SELECT count(1) FROM t_comment c` + where
	var total int
	if _, err := session.SQL(countQuery, args...).Get(&total); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.moderation.count", err)
	}
	query := `SELECT c.id, c.user_id, u.nickname, u.avatar, c.comment_content, c.parent_id, c.is_top, c.is_delete, c.is_review,
			(SELECT count(1) FROM t_comment reply WHERE reply.parent_id = c.id AND reply.is_delete = 0 AND reply.is_review = 1) AS reply_count,
			(SELECT count(1) FROM t_comment_report report WHERE report.comment_id = c.id AND report.status = 'pending') AS report_count,
			c.create_time
		FROM t_comment c JOIN t_user_info u ON c.user_id = u.id` + where +
		` ORDER BY c.is_delete ASC, c.is_top DESC, c.id DESC LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	var comments []*port.OwnedComment
	if err := session.SQL(query, listArgs...).Find(&comments); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "comment.moderation.list", err)
	}
	return comments, total, nil
}

func (c *MyCommentRepo) CountPendingReportsByCommentIDs(ctx context.Context, commentIDs []int) (map[int]int, error) {
	counts := make(map[int]int)
	if len(commentIDs) == 0 {
		return counts, nil
	}
	session, err := c.commentSession(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]struct {
		CommentId int `xorm:"comment_id"`
		Total     int `xorm:"total"`
	}, 0, len(commentIDs))
	query := `SELECT comment_id, count(1) AS total FROM t_comment_report
		WHERE status = 'pending' AND comment_id IN (` + placeholders(len(commentIDs)) + `) GROUP BY comment_id`
	if err := session.SQL(query, intArgs(commentIDs)...).Find(&rows); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "comment.report.count", err)
	}
	for _, row := range rows {
		counts[row.CommentId] = row.Total
	}
	return counts, nil
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
