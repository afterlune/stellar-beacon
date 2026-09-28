package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

var _ port.TopicSubscriptionRepository = (*MyTopicSubscriptionRepo)(nil)

type MyTopicSubscriptionRepo struct{ engine *xorm.Engine }

func NewTopicSubscriptionRepo(engine *xorm.Engine) *MyTopicSubscriptionRepo {
	return &MyTopicSubscriptionRepo{engine: engine}
}

// taxonomyMembershipCTE resolves every public article to the normalised topic keys
// it belongs to. Categories and tags are owned per author, so the reader-facing
// key is the trimmed lower-cased name, exactly like the discovery surfaces.
const taxonomyMembershipCTE = `
	topic_membership AS (
		SELECT 'category' AS topic_type, lower(btrim(c.category_name)) AS topic_key, a.id AS article_id
		FROM t_article a
		JOIN t_category c ON c.id = a.category_id
		WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		GROUP BY 1, 2, 3
		UNION
		SELECT 'tag', lower(btrim(t.tag_name)), a.id
		FROM t_article_tag at
		JOIN t_tag t ON t.id = at.tag_id
		JOIN t_article a ON a.id = at.article_id
		WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		GROUP BY 1, 2, 3
	)
`

// normalizeTopicKey is the canonical subscription key. Every public topic
// surface groups by the same expression, so a subscription keeps matching the
// moment any author publishes under that name.
func normalizeTopicKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validTopicKey(key string) bool {
	if key == "" || len([]rune(key)) > 64 {
		return false
	}
	// Taxonomy names never contain control characters; rejecting them keeps a
	// hand-crafted request from storing an unusable key.
	return !strings.ContainsAny(key, "\r\n\t")
}

// topicDisplayName resolves the visible name of a topic that currently has at
// least one public article. It returns NotFound when nothing matches.
func topicDisplayName(session *xorm.Session, topicType, topicKey string) (string, error) {
	query := `
		SELECT COALESCE(min(c.category_name), '') AS name FROM t_category c
		JOIN t_article a ON a.category_id = c.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		WHERE lower(btrim(c.category_name)) = ?`
	if topicType == port.TopicTypeTag {
		query = `
		SELECT COALESCE(min(t.tag_name), '') AS name FROM t_tag t
		JOIN t_article_tag at ON at.tag_id = t.id
		JOIN t_article a ON a.id = at.article_id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		WHERE lower(btrim(t.tag_name)) = ?`
	}
	// The aggregate always returns exactly one row, so an empty name is the
	// "topic has no public content" signal.
	var name string
	if _, err := session.SQL(query, topicKey).Get(&name); err != nil {
		return "", apperrors.Unavailable("topic_subscription.resolve", err)
	}
	if strings.TrimSpace(name) == "" {
		return "", apperrors.NotFound("topic_subscription.resolve")
	}
	return name, nil
}

func (r *MyTopicSubscriptionRepo) Subscribe(ctx context.Context, userID int, topicType, topicKey string) error {
	key := normalizeTopicKey(topicKey)
	if userID <= 0 || !port.ValidTopicType(topicType) || !validTopicKey(key) {
		return apperrors.Invalid("topic_subscription.create", "invalid topic subscription")
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		name, err := topicDisplayName(session, topicType, key)
		if err != nil {
			return err
		}
		var cursor int64
		if _, err := session.SQL("SELECT COALESCE(MAX(id), 0) FROM t_author_publish_event").Get(&cursor); err != nil {
			return apperrors.Unavailable("topic_subscription.cursor", err)
		}
		// Re-subscribing after an unsubscribe keeps the previous read state so a
		// reader is not re-notified about content they already saw.
		if _, err := session.Exec(`
			INSERT INTO t_topic_subscription (user_id, topic_type, topic_key, topic_name, muted, start_event_id, last_read_event_id)
			VALUES (?, ?, ?, ?, 0, ?, ?)
			ON CONFLICT (user_id, topic_type, topic_key)
			DO UPDATE SET topic_name = EXCLUDED.topic_name, updated_at = CURRENT_TIMESTAMP`,
			userID, topicType, key, name, cursor, cursor); err != nil {
			return apperrors.Unavailable("topic_subscription.create", err)
		}
		return nil
	})
}

func (r *MyTopicSubscriptionRepo) Unsubscribe(ctx context.Context, userID int, topicType, topicKey string) error {
	key := normalizeTopicKey(topicKey)
	if userID <= 0 || !port.ValidTopicType(topicType) || !validTopicKey(key) {
		return apperrors.Invalid("topic_subscription.delete", "invalid topic subscription")
	}
	session, err := repoSession(r.engine, ctx, "topic_subscription.delete")
	if err != nil {
		return err
	}
	if _, err := session.Exec("DELETE FROM t_topic_subscription WHERE user_id = ? AND topic_type = ? AND topic_key = ?", userID, topicType, key); err != nil {
		return apperrors.Unavailable("topic_subscription.delete", err)
	}
	return nil
}

func (r *MyTopicSubscriptionRepo) SetMuted(ctx context.Context, userID int, topicType, topicKey string, muted bool) error {
	key := normalizeTopicKey(topicKey)
	if userID <= 0 || !port.ValidTopicType(topicType) || !validTopicKey(key) {
		return apperrors.Invalid("topic_subscription.mute", "invalid topic subscription")
	}
	value := 0
	if muted {
		value = 1
	}
	session, err := repoSession(r.engine, ctx, "topic_subscription.mute")
	if err != nil {
		return err
	}
	result, err := session.Exec(
		"UPDATE t_topic_subscription SET muted = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ? AND topic_type = ? AND topic_key = ?",
		value, userID, topicType, key)
	if err != nil {
		return apperrors.Unavailable("topic_subscription.mute", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("topic_subscription.mute", err)
	}
	if affected == 0 {
		return apperrors.NotFound("topic_subscription.mute")
	}
	return nil
}

func (r *MyTopicSubscriptionRepo) ListSubscriptions(ctx context.Context, userID, current, size int) ([]port.TopicSubscription, int, error) {
	if userID <= 0 {
		return nil, 0, apperrors.Invalid("topic_subscription.list", "invalid user")
	}
	session, err := repoSession(r.engine, ctx, "topic_subscription.list")
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL("SELECT count(1) FROM t_topic_subscription WHERE user_id = ?", userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("topic_subscription.list.count", err)
	}
	if count == 0 {
		return []port.TopicSubscription{}, 0, nil
	}
	limit, offset := pgsql.Page(current, size)
	windowStart, windowDate := discoveryWindow(time.Now())
	var rows []struct {
		TopicType    string    `xorm:"topic_type"`
		TopicKey     string    `xorm:"topic_key"`
		TopicName    string    `xorm:"topic_name"`
		Muted        int       `xorm:"muted"`
		ArticleCount int       `xorm:"article_count"`
		HotScore     int       `xorm:"hot_score"`
		UnreadCount  int       `xorm:"unread_count"`
		SubscribedAt time.Time `xorm:"subscribed_at"`
	}
	if err := session.SQL("WITH "+taxonomyMembershipCTE+","+strings.TrimPrefix(discoveryScoredCTE, "\n\t")+`
		, topic_unread AS (
			SELECT subscription.id AS subscription_id, count(DISTINCT event.id)::int AS unread_count
			FROM t_topic_subscription subscription
			JOIN t_author_publish_event event ON event.content_type = 'article' AND event.id > subscription.last_read_event_id
			JOIN t_article article ON article.id = event.content_id
				AND article.is_delete = 0 AND article.status = 1 AND article.moderation_status = 'visible'
			JOIN topic_membership membership ON membership.article_id = article.id
				AND membership.topic_type = subscription.topic_type AND membership.topic_key = subscription.topic_key
			WHERE subscription.user_id = ? AND subscription.muted = 0
			GROUP BY subscription.id
		),
		topic_totals AS (
			SELECT subscription.id AS subscription_id,
			       count(membership.article_id)::int AS article_count,
			       COALESCE(SUM(scored.hot_score), 0)::int AS hot_score
			FROM t_topic_subscription subscription
			LEFT JOIN topic_membership membership
				ON membership.topic_type = subscription.topic_type AND membership.topic_key = subscription.topic_key
			LEFT JOIN scored ON scored.id = membership.article_id
			WHERE subscription.user_id = ?
			GROUP BY subscription.id
		)
		SELECT subscription.topic_type, subscription.topic_key, subscription.topic_name, subscription.muted,
		       COALESCE(totals.article_count, 0) AS article_count,
		       COALESCE(totals.hot_score, 0) AS hot_score,
		       COALESCE(unread.unread_count, 0) AS unread_count,
		       subscription.created_at AS subscribed_at
		FROM t_topic_subscription subscription
		LEFT JOIN topic_totals totals ON totals.subscription_id = subscription.id
		LEFT JOIN topic_unread unread ON unread.subscription_id = subscription.id
		WHERE subscription.user_id = ?
		ORDER BY subscription.updated_at DESC, subscription.id DESC
		LIMIT ? OFFSET ?`,
		windowDate, windowStart, windowStart, userID, userID, userID, limit, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("topic_subscription.list", err)
	}
	result := make([]port.TopicSubscription, 0, len(rows))
	for _, row := range rows {
		result = append(result, port.TopicSubscription{
			TopicType: row.TopicType, TopicKey: row.TopicKey, TopicName: row.TopicName,
			ArticleCount: row.ArticleCount, HotScore: row.HotScore, Muted: row.Muted == 1,
			UnreadCount: row.UnreadCount, SubscribedAt: row.SubscribedAt,
		})
	}
	return result, count, nil
}

func (r *MyTopicSubscriptionRepo) ListTopicFeed(ctx context.Context, userID, current, size int) ([]port.TopicFeedItem, int, error) {
	if userID <= 0 {
		return nil, 0, apperrors.Invalid("topic_feed.list", "invalid user")
	}
	session, err := repoSession(r.engine, ctx, "topic_feed.list")
	if err != nil {
		return nil, 0, err
	}
	const fromWhere = `
		FROM t_topic_subscription subscription
		JOIN topic_membership membership
			ON membership.topic_type = subscription.topic_type AND membership.topic_key = subscription.topic_key
		JOIN t_article article ON article.id = membership.article_id
		JOIN t_author_publish_event event
			ON event.content_type = 'article' AND event.content_id = article.id AND event.id > subscription.start_event_id
		JOIN t_user_info author ON author.id = event.author_id AND author.is_disable = 0
		WHERE subscription.user_id = ?`
	var count int
	if _, err := session.SQL("WITH "+taxonomyMembershipCTE+`SELECT count(DISTINCT event.id)`+fromWhere, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("topic_feed.count", err)
	}
	if count == 0 {
		return []port.TopicFeedItem{}, 0, nil
	}
	limit, offset := pgsql.Page(current, size)
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
		Topics       string    `xorm:"topics"`
		PublishedAt  time.Time `xorm:"published_at"`
	}
	if err := session.SQL("WITH "+taxonomyMembershipCTE+`,
		topic_feed AS (
			SELECT event.id AS event_id, event.content_type, event.content_id, event.author_id, event.published_at,
			       article.article_title, article.article_content, article.article_cover,
			       array_agg(DISTINCT subscription.topic_name) AS topic_names
			FROM t_topic_subscription subscription
			JOIN topic_membership membership
				ON membership.topic_type = subscription.topic_type AND membership.topic_key = subscription.topic_key
			JOIN t_article article ON article.id = membership.article_id
			JOIN t_author_publish_event event
				ON event.content_type = 'article' AND event.content_id = article.id AND event.id > subscription.start_event_id
			WHERE subscription.user_id = ?
			GROUP BY event.id, event.content_type, event.content_id, event.author_id, event.published_at,
			         article.article_title, article.article_content, article.article_cover
		)
		SELECT feed.event_id, feed.content_type, feed.content_id, feed.author_id,
		       COALESCE(feed.article_title, '') AS title,
		       COALESCE(SUBSTR(feed.article_content, 1, 240), '') AS excerpt,
		       COALESCE(feed.article_cover, '') AS cover,
		       to_json(feed.topic_names)::text AS topics,
		       feed.published_at,
		       author.handle AS author_handle, author.nickname AS author_name, author.avatar AS author_avatar
		FROM topic_feed feed
		JOIN t_user_info author ON author.id = feed.author_id AND author.is_disable = 0
		ORDER BY feed.published_at DESC, feed.event_id DESC
		LIMIT ? OFFSET ?`, userID, limit, offset).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("topic_feed.list", err)
	}
	result := make([]port.TopicFeedItem, 0, len(rows))
	for _, row := range rows {
		item := port.TopicFeedItem{
			FollowFeedItem: port.FollowFeedItem{
				EventId: row.EventId, ContentType: row.ContentType, ContentId: row.ContentId,
				Author:      port.PublicAuthor{Id: row.AuthorId, Handle: row.AuthorHandle, Nickname: row.AuthorName, Avatar: row.AuthorAvatar},
				Title:       row.Title,
				Excerpt:     row.Excerpt,
				Cover:       row.Cover,
				PublishedAt: row.PublishedAt,
			},
			Topics: []string{},
		}
		if strings.TrimSpace(row.Topics) != "" {
			var topics []string
			if err := json.Unmarshal([]byte(row.Topics), &topics); err == nil {
				item.Topics = topics
			}
		}
		result = append(result, item)
	}
	return result, count, nil
}
