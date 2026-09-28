package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	pgsql "github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

var _ port.RecommendationRepository = (*MyRecommendationRepo)(nil)

type MyRecommendationRepo struct{ engine *xorm.Engine }

func NewRecommendationRepo(engine *xorm.Engine) *MyRecommendationRepo {
	return &MyRecommendationRepo{engine: engine}
}

const recommendationLookbackMonths = 6
const recommendationMaxBatches = 20

type recommendationCandidateRow struct {
	ArticleID         int       `xorm:"article_id"`
	UserID            int       `xorm:"user_id"`
	ArticleCover      string    `xorm:"article_cover"`
	ArticleTitle      string    `xorm:"article_title"`
	ArticleContent    string    `xorm:"article_content"`
	IsTop             int       `xorm:"is_top"`
	IsFeatured        int       `xorm:"is_featured"`
	CategoryName      string    `xorm:"category_name"`
	Status            int       `xorm:"status"`
	ModerationStatus  string    `xorm:"moderation_status"`
	CreateTime        time.Time `xorm:"create_time"`
	UpdateTime        time.Time `xorm:"update_time"`
	AuthorHandle      string    `xorm:"author_handle"`
	AuthorNickname    string    `xorm:"author_nickname"`
	AuthorAvatar      string    `xorm:"author_avatar"`
	RankWindow        int       `xorm:"rank_window"`
	Score             int       `xorm:"score"`
	SubscriptionTopic string    `xorm:"subscription_topic"`
	FavoriteMatch     bool      `xorm:"favorite_match"`
	LikeMatch         bool      `xorm:"like_match"`
	SeedMatch         bool      `xorm:"seed_match"`
	FollowedTopic     bool      `xorm:"followed_topic"`
	HotScore          int       `xorm:"hot_score"`
}

func recommendationLookback(now time.Time) time.Time {
	return now.AddDate(0, -recommendationLookbackMonths, 0)
}

func recommendationSeedCTE(seedIDs []int, args *[]interface{}) string {
	if len(seedIDs) == 0 {
		return ",\n\tseed_ids(article_id) AS (SELECT 0 WHERE FALSE)"
	}
	for _, id := range seedIDs {
		*args = append(*args, id)
	}
	return ",\n\tseed_ids(article_id) AS (VALUES " + strings.TrimSuffix(strings.Repeat("(?::int),", len(seedIDs)), ",") + ")"
}

func recommendationCursorArgs(cursor *port.RecommendationCursor) []interface{} {
	if cursor == nil {
		return []interface{}{0, 0, 0, 0, 0, 0, time.Time{}, 0, 0, time.Time{}, 0}
	}
	return []interface{}{
		1,
		cursor.Window, cursor.Window, cursor.Score,
		cursor.Window, cursor.Score, cursor.PublishedAt,
		cursor.Window, cursor.Score, cursor.PublishedAt, cursor.ArticleID,
	}
}

func recommendationCursorFromRow(row recommendationCandidateRow) *port.RecommendationCursor {
	return &port.RecommendationCursor{
		Version: 1, Window: row.RankWindow, Score: row.Score,
		PublishedAt: row.CreateTime, ArticleID: row.ArticleID,
	}
}

func (r *MyRecommendationRepo) ListRecommendations(ctx context.Context, request port.RecommendationRequest) (port.RecommendationPage, error) {
	result := port.RecommendationPage{Items: []port.RecommendationItem{}}
	if request.UserID <= 0 || request.Size <= 0 {
		return result, apperrors.Invalid("recommendation.list", "invalid request")
	}
	session, err := repoSession(r.engine, ctx, "recommendation.list")
	if err != nil {
		return result, err
	}
	if request.Snapshot.IsZero() {
		request.Snapshot = time.Now()
	}
	interestStart := recommendationLookback(request.Snapshot)
	var signals int
	if _, err := session.SQL(`SELECT CASE WHEN
			EXISTS (SELECT 1 FROM t_topic_subscription WHERE user_id = ?)
			OR EXISTS (SELECT 1 FROM t_article_reaction WHERE user_info_id = ? AND create_time >= ?)
			OR EXISTS (SELECT 1 FROM t_user_follow WHERE follower_id = ?)
			OR ? > 0 THEN 1 ELSE 0 END`,
		request.UserID, request.UserID, interestStart, request.UserID, len(request.SeedArticleIDs)).Get(&signals); err != nil {
		return result, apperrors.Unavailable("recommendation.signals", err)
	}
	result.Personalized = signals > 0

	cursor := request.Cursor
	for batch := 0; batch < recommendationMaxBatches && len(result.Items) < request.Size; batch++ {
		remaining := request.Size - len(result.Items)
		limit := remaining*4 + 1
		if limit < 25 {
			limit = 25
		}
		if limit > 200 {
			limit = 200
		}
		rows, queryErr := r.loadRecommendationBatch(ctx, session, request, cursor, limit+1)
		if queryErr != nil {
			return result, queryErr
		}
		if len(rows) == 0 {
			result.NextCursor = nil
			result.HasMore = false
			return result, nil
		}
		hasExtra := len(rows) > limit
		if hasExtra {
			rows = rows[:limit]
		}
		authorCounts := map[int]int{}
		for index, row := range rows {
			if authorCounts[row.UserID] >= 2 {
				continue
			}
			authorCounts[row.UserID]++
			item, buildErr := r.buildRecommendationItem(session, row)
			if buildErr != nil {
				return result, buildErr
			}
			result.Items = append(result.Items, item)
			cursor = recommendationCursorFromRow(row)
			cursor.Snapshot = request.Snapshot
			if len(result.Items) == request.Size {
				result.NextCursor = cursor
				result.HasMore = index < len(rows)-1 || hasExtra
				return result, nil
			}
		}
		last := rows[len(rows)-1]
		cursor = recommendationCursorFromRow(last)
		cursor.Snapshot = request.Snapshot
		result.NextCursor = cursor
		result.HasMore = hasExtra
		if !hasExtra {
			return result, nil
		}
	}
	return result, nil
}

const recommendationArticleTopicsCTE = `,
	article_topics AS (
		SELECT 'category' AS topic_type, lower(btrim(c.category_name)) AS topic_key,
		       min(c.category_name) AS topic_name, a.id AS article_id
		FROM t_article a
		JOIN t_category c ON c.id = a.category_id
		WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		GROUP BY 1, 2, 4
		UNION ALL
		SELECT 'tag', lower(btrim(t.tag_name)), min(t.tag_name), a.id
		FROM t_article_tag at
		JOIN t_tag t ON t.id = at.tag_id
		JOIN t_article a ON a.id = at.article_id
		WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		GROUP BY 1, 2, 4
	),
	recent_reactions AS (
		SELECT reaction.article_id, reaction.reaction
		FROM t_article_reaction reaction
		WHERE reaction.user_info_id = ? AND reaction.create_time >= ?
	),
	feedback AS (
		SELECT target_type, article_id, author_id, topic_type, topic_key
		FROM t_recommendation_feedback WHERE user_id = ?
	)`

const recommendationAffinityCTE = `,
	topic_affinity AS (
		SELECT topics.article_id, topics.topic_type, topics.topic_key, topics.topic_name, 40 AS weight, 'subscription'::text AS source
		FROM t_topic_subscription subscription
		JOIN article_topics topics ON topics.topic_type = subscription.topic_type AND topics.topic_key = subscription.topic_key
		WHERE subscription.user_id = ?
		UNION ALL
		SELECT topics.article_id, topics.topic_type, topics.topic_key, topics.topic_name,
		       CASE WHEN reaction.reaction = 'favorite' THEN 25 ELSE 20 END, reaction.reaction
		FROM recent_reactions reaction
		JOIN article_topics topics ON topics.article_id = reaction.article_id
		UNION ALL
		SELECT candidate_topics.article_id, candidate_topics.topic_type, candidate_topics.topic_key, candidate_topics.topic_name, 15, 'seed'
		FROM seed_ids seed
		JOIN article_topics seed_topics ON seed_topics.article_id = seed.article_id
		JOIN article_topics candidate_topics
			ON candidate_topics.topic_type = seed_topics.topic_type
			AND candidate_topics.topic_key = seed_topics.topic_key
		UNION ALL
		SELECT topics.article_id, topics.topic_type, topics.topic_key, topics.topic_name, 10, 'follow'
		FROM t_user_follow follow
		JOIN t_article followed_article ON followed_article.user_id = follow.author_id
			AND followed_article.is_delete = 0 AND followed_article.status = 1
			AND followed_article.moderation_status = 'visible' AND followed_article.create_time >= ?
		JOIN article_topics topics ON topics.article_id = followed_article.id
		WHERE follow.follower_id = ?
	),
	article_topic_scores AS (
		SELECT article_id,
		       LEAST(100, SUM(weight))::int AS score,
		       min(topic_name) FILTER (WHERE source = 'subscription') AS subscription_topic,
		       bool_or(source = 'favorite') AS favorite_match,
		       bool_or(source = 'like') AS like_match,
		       bool_or(source = 'seed') AS seed_match,
		       bool_or(source = 'follow') AS followed_topic
		FROM topic_affinity
		GROUP BY article_id
	),
	author_affinity AS (
		SELECT article.user_id AS author_id,
		       CASE WHEN reaction.reaction = 'favorite' THEN 20 ELSE 15 END AS weight,
		       reaction.reaction AS source
		FROM recent_reactions reaction
		JOIN t_article article ON article.id = reaction.article_id
		UNION ALL
		SELECT article.user_id, 8, 'seed'
		FROM seed_ids seed
		JOIN t_article article ON article.id = seed.article_id
	),
	article_author_scores AS (
		SELECT author_id, LEAST(30, SUM(weight))::int AS score,
		       bool_or(source = 'favorite') AS favorite_match,
		       bool_or(source = 'like') AS like_match,
		       bool_or(source = 'seed') AS seed_match
		FROM author_affinity
		GROUP BY author_id
	)`

const recommendationRankedCTE = `,
	ranked_candidates AS (
		SELECT article.id AS article_id, article.user_id, article.article_cover, article.article_title,
		       SUBSTR(article.article_content, 1, 500) AS article_content,
		       article.is_top, article.is_featured, category.category_name, article.status,
		       article.moderation_status, article.create_time, article.update_time,
		       author.handle AS author_handle, author.nickname AS author_nickname, author.avatar AS author_avatar,
		       CASE WHEN article.create_time >= ? THEN 0 ELSE 1 END AS rank_window,
		       GREATEST(0,
		         LEAST(100, COALESCE(topic_scores.score, 0))
		         + LEAST(30, COALESCE(author_scores.score, 0))
		         + LEAST(30, COALESCE(scored.hot_score, 0))
		         - 80 * (SELECT count(*) FROM feedback muted_topic
		                 JOIN article_topics muted_topics ON muted_topics.article_id = article.id
		                    AND muted_topics.topic_type = muted_topic.topic_type AND muted_topics.topic_key = muted_topic.topic_key
		                 WHERE muted_topic.target_type = 'topic')
		         - CASE WHEN EXISTS (SELECT 1 FROM feedback muted_author
		                             WHERE muted_author.target_type = 'author' AND muted_author.author_id = article.user_id)
		                THEN 80 ELSE 0 END
		       )::int AS score,
		       LEAST(30, COALESCE(scored.hot_score, 0))::int AS hot_score,
		       COALESCE(topic_scores.subscription_topic, '') AS subscription_topic,
		       COALESCE(topic_scores.favorite_match, false) OR COALESCE(author_scores.favorite_match, false) AS favorite_match,
		       COALESCE(topic_scores.like_match, false) OR COALESCE(author_scores.like_match, false) AS like_match,
		       COALESCE(topic_scores.seed_match, false) OR COALESCE(author_scores.seed_match, false) AS seed_match,
		       COALESCE(topic_scores.followed_topic, false) AS followed_topic
		FROM t_article article
		JOIN scored ON scored.id = article.id
		LEFT JOIN t_category category ON category.id = article.category_id
		JOIN t_user_info author ON author.id = article.user_id AND author.is_disable = 0
		LEFT JOIN article_topic_scores topic_scores ON topic_scores.article_id = article.id
		LEFT JOIN article_author_scores author_scores ON author_scores.author_id = article.user_id
		WHERE article.is_delete = 0 AND article.status = 1 AND article.moderation_status = 'visible'
		  AND article.create_time <= ?
		  AND article.user_id <> ?
		  AND NOT EXISTS (SELECT 1 FROM t_user_follow own_follow WHERE own_follow.follower_id = ? AND own_follow.author_id = article.user_id)
		  AND NOT EXISTS (SELECT 1 FROM t_article_reaction own_reaction WHERE own_reaction.user_info_id = ? AND own_reaction.article_id = article.id)
		  AND NOT EXISTS (SELECT 1 FROM seed_ids own_seed WHERE own_seed.article_id = article.id)
		  AND NOT EXISTS (SELECT 1 FROM feedback hidden_article WHERE hidden_article.target_type = 'article' AND hidden_article.article_id = article.id)
	)`

const recommendationCursorSQL = `
	SELECT * FROM ranked_candidates candidate
	WHERE (? = 0
		OR candidate.rank_window > ?
		OR (candidate.rank_window = ? AND candidate.score < ?)
		OR (candidate.rank_window = ? AND candidate.score = ? AND candidate.create_time < ?)
		OR (candidate.rank_window = ? AND candidate.score = ? AND candidate.create_time = ? AND candidate.article_id < ?))
	ORDER BY candidate.rank_window ASC, candidate.score DESC, candidate.create_time DESC, candidate.article_id DESC
	LIMIT ?`

func (r *MyRecommendationRepo) loadRecommendationBatch(ctx context.Context, session *xorm.Session, request port.RecommendationRequest, cursor *port.RecommendationCursor, limit int) ([]recommendationCandidateRow, error) {
	windowStart, windowDate := discoveryWindow(request.Snapshot)
	interestStart := recommendationLookback(request.Snapshot)
	args := []interface{}{windowDate, windowStart, windowStart, request.UserID, interestStart, request.UserID}
	seedCTE := recommendationSeedCTE(request.SeedArticleIDs, &args)
	args = append(args, request.UserID, interestStart, request.UserID, windowStart, request.Snapshot, request.UserID, request.UserID, request.UserID)
	args = append(args, recommendationCursorArgs(cursor)...)
	args = append(args, limit)
	queryText := "WITH " + discoveryScoredCTE + recommendationArticleTopicsCTE + seedCTE + recommendationAffinityCTE + recommendationRankedCTE + recommendationCursorSQL
	var rows []recommendationCandidateRow
	if err := session.SQL(queryText, args...).Find(&rows); err != nil {
		slog.ErrorContext(ctx, "recommendation query failed", "error", err)
		return nil, apperrors.Unavailable("recommendation.list", err)
	}
	return rows, nil
}

func (r *MyRecommendationRepo) buildRecommendationItem(session *xorm.Session, row recommendationCandidateRow) (port.RecommendationItem, error) {
	card := port.ArticleCard{
		Id: row.ArticleID, UserId: row.UserID, ArticleCover: row.ArticleCover,
		ArticleTitle: row.ArticleTitle, ArticleContent: row.ArticleContent,
		IsTop: row.IsTop, IsFeatured: row.IsFeatured, CategoryName: row.CategoryName,
		Status: row.Status, ModerationStatus: row.ModerationStatus,
		CreateTime: row.CreateTime, UpdateTime: row.UpdateTime,
		Author: port.PublicAuthor{Id: row.UserID, Handle: row.AuthorHandle, Nickname: row.AuthorNickname, Avatar: row.AuthorAvatar},
	}
	reason := port.RecommendationReason{Type: port.RecommendationReasonLatest, Label: "新近发布"}
	if row.SubscriptionTopic != "" {
		reason = port.RecommendationReason{Type: port.RecommendationReasonSubscribedTopic, Label: fmt.Sprintf("你订阅了「%s」", row.SubscriptionTopic)}
	} else if row.FavoriteMatch {
		reason = port.RecommendationReason{Type: port.RecommendationReasonFavorite, Label: "你收藏过相关内容"}
	} else if row.LikeMatch {
		reason = port.RecommendationReason{Type: port.RecommendationReasonLike, Label: "你赞过相关内容"}
	} else if row.SeedMatch {
		reason = port.RecommendationReason{Type: port.RecommendationReasonReading, Label: "与你最近阅读相关"}
	} else if row.FollowedTopic {
		reason = port.RecommendationReason{Type: port.RecommendationReasonFollowedTopic, Label: "你关注的作者也在写这个主题"}
	} else if row.HotScore > 0 {
		reason = port.RecommendationReason{Type: port.RecommendationReasonTrending, Label: "近期热门"}
	}
	var tags []string
	if err := session.SQL(pgsql.ArticleTags, row.ArticleID).Find(&tags); err != nil {
		return port.RecommendationItem{}, apperrors.Unavailable("recommendation.tags", err)
	}
	card.Tags = tags
	return port.RecommendationItem{ArticleCard: card, Reason: reason}, nil
}

type recommendationFeedbackRow struct {
	ID         int       `xorm:"id"`
	TargetType string    `xorm:"target_type"`
	TargetKey  string    `xorm:"target_key"`
	ArticleID  int       `xorm:"article_id"`
	AuthorID   int       `xorm:"author_id"`
	TopicType  string    `xorm:"topic_type"`
	TopicKey   string    `xorm:"topic_key"`
	Label      string    `xorm:"target_label"`
	CreatedAt  time.Time `xorm:"create_time"`
}

func (r recommendationFeedbackRow) toPort() port.RecommendationFeedback {
	return port.RecommendationFeedback{
		ID: r.ID, TargetType: r.TargetType, TargetKey: r.TargetKey,
		ArticleID: r.ArticleID, AuthorID: r.AuthorID, TopicType: r.TopicType,
		TopicKey: r.TopicKey, Label: r.Label, CreatedAt: r.CreatedAt,
	}
}

func (r *MyRecommendationRepo) ListFeedback(ctx context.Context, userID, current, size int) ([]port.RecommendationFeedback, int, error) {
	if userID <= 0 {
		return nil, 0, apperrors.Invalid("recommendation_feedback.list", "invalid user")
	}
	session, err := repoSession(r.engine, ctx, "recommendation_feedback.list")
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL("SELECT count(1) FROM t_recommendation_feedback WHERE user_id = ?", userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("recommendation_feedback.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	var rows []recommendationFeedbackRow
	if err := session.SQL(`SELECT id, target_type, target_key, COALESCE(article_id, 0) AS article_id,
			COALESCE(author_id, 0) AS author_id, COALESCE(topic_type, '') AS topic_type,
			COALESCE(topic_key, '') AS topic_key, target_label, create_time
		FROM t_recommendation_feedback WHERE user_id = ?
		ORDER BY create_time DESC, id DESC LIMIT ? OFFSET ?`, userID, limit, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("recommendation_feedback.list", err)
	}
	items := make([]port.RecommendationFeedback, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toPort())
	}
	return items, count, nil
}

func (r *MyRecommendationRepo) UpsertFeedback(ctx context.Context, userID int, input port.RecommendationFeedbackInput) (port.RecommendationFeedback, error) {
	var result port.RecommendationFeedback
	if userID <= 0 || !port.ValidRecommendationTarget(input.TargetType) {
		return result, apperrors.Invalid("recommendation_feedback.save", "invalid feedback target")
	}
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var (
			targetKey string
			label     string
			articleID *int
			authorID  *int
			topicType string
			topicKey  string
		)
		switch input.TargetType {
		case port.RecommendationTargetArticle:
			if input.ArticleID <= 0 {
				return apperrors.Invalid("recommendation_feedback.article", "invalid article")
			}
			var row struct {
				Title  string `xorm:"article_title"`
				UserID int    `xorm:"user_id"`
			}
			found, queryErr := session.SQL(`SELECT article_title, user_id FROM t_article
				WHERE id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, input.ArticleID).Get(&row)
			if queryErr != nil {
				return apperrors.Unavailable("recommendation_feedback.article", queryErr)
			}
			if !found || row.UserID == userID {
				return apperrors.NotFound("recommendation_feedback.article")
			}
			targetKey, label = strconv.Itoa(input.ArticleID), strings.TrimSpace(row.Title)
			id := input.ArticleID
			articleID = &id
		case port.RecommendationTargetAuthor:
			if input.AuthorID <= 0 || input.AuthorID == userID {
				return apperrors.Invalid("recommendation_feedback.author", "invalid author")
			}
			var row struct {
				Nickname string `xorm:"nickname"`
				Handle   string `xorm:"handle"`
			}
			found, queryErr := session.SQL("SELECT COALESCE(nickname, '') AS nickname, COALESCE(handle, '') AS handle FROM t_user_info WHERE id = ? AND is_disable = 0", input.AuthorID).Get(&row)
			if queryErr != nil {
				return apperrors.Unavailable("recommendation_feedback.author", queryErr)
			}
			if !found {
				return apperrors.NotFound("recommendation_feedback.author")
			}
			targetKey = strconv.Itoa(input.AuthorID)
			label = strings.TrimSpace(row.Nickname)
			if label == "" {
				label = row.Handle
			}
			id := input.AuthorID
			authorID = &id
		case port.RecommendationTargetTopic:
			topicKey = normalizeTopicKey(input.TopicKey)
			if !port.ValidTopicType(input.TopicType) || topicKey == "" {
				return apperrors.Invalid("recommendation_feedback.topic", "invalid topic")
			}
			name, queryErr := topicDisplayName(session, input.TopicType, topicKey)
			if queryErr != nil {
				return queryErr
			}
			targetKey, label, topicType = input.TopicType+":"+topicKey, name, input.TopicType
		}
		var row recommendationFeedbackRow
		if _, queryErr := session.SQL(`INSERT INTO t_recommendation_feedback
			(user_id, target_type, target_key, article_id, author_id, topic_type, topic_key, target_label)
			VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?)
			ON CONFLICT (user_id, target_type, target_key)
			DO UPDATE SET target_label = EXCLUDED.target_label, create_time = CURRENT_TIMESTAMP
			RETURNING id, target_type, target_key, COALESCE(article_id, 0) AS article_id,
				COALESCE(author_id, 0) AS author_id, COALESCE(topic_type, '') AS topic_type,
				COALESCE(topic_key, '') AS topic_key, target_label, create_time`,
			userID, input.TargetType, targetKey, articleID, authorID, topicType, topicKey, label).Get(&row); queryErr != nil {
			return apperrors.Unavailable("recommendation_feedback.save", queryErr)
		}
		result = row.toPort()
		return nil
	})
	return result, err
}

func (r *MyRecommendationRepo) DeleteFeedback(ctx context.Context, userID, feedbackID int) error {
	if userID <= 0 || feedbackID <= 0 {
		return apperrors.Invalid("recommendation_feedback.delete", "invalid feedback")
	}
	session, err := repoSession(r.engine, ctx, "recommendation_feedback.delete")
	if err != nil {
		return err
	}
	if _, err := session.Exec("DELETE FROM t_recommendation_feedback WHERE id = ? AND user_id = ?", feedbackID, userID); err != nil {
		return apperrors.Unavailable("recommendation_feedback.delete", err)
	}
	return nil
}
