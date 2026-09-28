package repository

import (
	"context"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

func (r *MyPlatformRepo) ListFeedArticles(ctx context.Context, current, size int, featuredOnly bool) ([]*port.ArticleCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.feed.articles")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	where := `a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'`
	if featuredOnly {
		where += ` AND a.is_featured = 1`
	}
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_article a WHERE ` + where).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.articles.count", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a
		LEFT JOIN t_category c ON a.category_id = c.id
		WHERE `+where+`
		ORDER BY a.is_top DESC, a.is_featured DESC, a.create_time DESC, a.id DESC
		LIMIT ? OFFSET ?`, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.articles.list", err)
	}
	if err := r.attachArticleAuthorsAndTags(ctx, session, articles); err != nil {
		return nil, 0, err
	}
	return articles, count, nil
}

func (r *MyPlatformRepo) ListFeedArticlesHot(ctx context.Context, current, size int) ([]*port.ArticleCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.feed.articles.hot")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	windowStart, windowDate := discoveryWindow(time.Now())
	windowArgs := []interface{}{windowDate, windowStart, windowStart}
	var count int
	if _, err := session.SQL(discoveryHotScoreCTE+`
		SELECT count(1) FROM scored WHERE hot_score > 0`, windowArgs...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.articles.hot.count", err)
	}
	if count == 0 {
		return []*port.ArticleCard{}, 0, nil
	}
	pageArgs := append(append([]interface{}{}, windowArgs...), limit, offset)
	var articles []*port.ArticleCard
	if err := session.SQL(discoveryHotScoreCTE+`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a
		JOIN scored s ON s.id = a.id
		LEFT JOIN t_category c ON a.category_id = c.id
		WHERE s.hot_score > 0
		ORDER BY s.hot_score DESC, a.create_time DESC, a.id DESC
		LIMIT ? OFFSET ?`, pageArgs...).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.articles.hot.list", err)
	}
	if err := r.attachArticleAuthorsAndTags(ctx, session, articles); err != nil {
		return nil, 0, err
	}
	return articles, count, nil
}
func (r *MyPlatformRepo) attachArticleAuthorsAndTags(ctx context.Context, session *xorm.Session, articles []*port.ArticleCard) error {
	if len(articles) == 0 {
		return nil
	}
	userIDs := make([]int, 0, len(articles))
	seen := map[int]struct{}{}
	for _, article := range articles {
		if article.UserId > 0 {
			if _, ok := seen[article.UserId]; !ok {
				seen[article.UserId] = struct{}{}
				userIDs = append(userIDs, article.UserId)
			}
		}
	}
	var users []entity.TUserInfo
	if len(userIDs) > 0 {
		if err := session.In("id", userIDs).Find(&users); err != nil {
			return apperrors.Unavailable("platform.feed.authors", err)
		}
	}
	byID := make(map[int]entity.TUserInfo, len(users))
	for _, user := range users {
		byID[user.Id] = user
	}
	for _, article := range articles {
		article.Author = toPublicAuthor(byID[article.UserId])
		var tags []string
		if err := session.SQL(pgsql.ArticleTags, article.Id).Find(&tags); err != nil {
			return apperrors.Unavailable("platform.feed.tags", err)
		}
		article.Tags = tags
	}
	return nil
}

func (r *MyPlatformRepo) ListFeedTalks(ctx context.Context, current, size int) ([]*port.Talk, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.feed.talks")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_talk WHERE status = 1 AND moderation_status = 'visible'`).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.talks.count", err)
	}
	var talks []*port.Talk
	if err := session.SQL(`
		SELECT t.id, t.user_id, t.content, t.images, t.is_top, t.status, t.moderation_status,
		       t.create_time, u.handle, u.nickname, u.avatar
		FROM t_talk t JOIN t_user_info u ON t.user_id = u.id
		WHERE t.status = 1 AND t.moderation_status = 'visible'
		ORDER BY t.is_top DESC, t.create_time DESC, t.id DESC LIMIT ? OFFSET ?`, limit, offset).Find(&talks); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.talks.list", err)
	}
	return talks, count, nil
}

func (r *MyPlatformRepo) ListAuthorArticles(ctx context.Context, userID, current, size int) ([]*port.ArticleCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.author.articles")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.articles.count", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a LEFT JOIN t_category c ON a.category_id = c.id
		WHERE a.user_id = ? AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		ORDER BY a.is_top DESC, a.create_time DESC, a.id DESC LIMIT ? OFFSET ?`, userID, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.articles.list", err)
	}
	if err := r.attachArticleAuthorsAndTags(ctx, session, articles); err != nil {
		return nil, 0, err
	}
	return articles, count, nil
}

func (r *MyPlatformRepo) ListAuthorArticlesHot(ctx context.Context, userID, current, size int) ([]*port.ArticleCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.author.articles.hot")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	windowStart, windowDate := discoveryWindow(time.Now())
	windowArgs := []interface{}{windowDate, windowStart, windowStart}
	var count int
	if _, err := session.SQL(discoveryHotScoreCTE+`
		SELECT count(1) FROM scored s
		JOIN t_article a ON a.id = s.id
		WHERE s.hot_score > 0 AND a.user_id = ?`, append(append([]interface{}{}, windowArgs...), userID)...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.articles.hot.count", err)
	}
	if count == 0 {
		return []*port.ArticleCard{}, 0, nil
	}
	pageArgs := append(append(append([]interface{}{}, windowArgs...), userID), limit, offset)
	var articles []*port.ArticleCard
	if err := session.SQL(discoveryHotScoreCTE+`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a
		JOIN scored s ON s.id = a.id
		LEFT JOIN t_category c ON a.category_id = c.id
		WHERE s.hot_score > 0 AND a.user_id = ?
		ORDER BY s.hot_score DESC, a.create_time DESC, a.id DESC
		LIMIT ? OFFSET ?`, pageArgs...).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.articles.hot.list", err)
	}
	if err := r.attachArticleAuthorsAndTags(ctx, session, articles); err != nil {
		return nil, 0, err
	}
	return articles, count, nil
}
func (r *MyPlatformRepo) ListAuthorTalks(ctx context.Context, userID, current, size int) ([]*port.Talk, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.author.talks")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_talk WHERE user_id = ? AND status = 1 AND moderation_status = 'visible'`, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.talks.count", err)
	}
	var talks []*port.Talk
	if err := session.SQL(`
		SELECT t.id, t.user_id, t.content, t.images, t.is_top, t.status, t.moderation_status,
		       t.create_time, u.handle, u.nickname, u.avatar
		FROM t_talk t JOIN t_user_info u ON t.user_id = u.id
		WHERE t.user_id = ? AND t.status = 1 AND t.moderation_status = 'visible'
		ORDER BY t.is_top DESC, t.create_time DESC, t.id DESC LIMIT ? OFFSET ?`, userID, limit, offset).Find(&talks); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.talks.list", err)
	}
	return talks, count, nil
}

func (r *MyPlatformRepo) ListAuthorSeries(ctx context.Context, userID, current, size int) ([]*port.Series, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.author.series")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_series WHERE user_id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.series.count", err)
	}
	var series []*port.Series
	if err := session.SQL(`
		SELECT s.id, s.user_id, s.series_name, s.series_desc, s.cover, s.status, s.moderation_status, s.update_time,
		       count(a.id) AS article_count
		FROM t_series s
		LEFT JOIN t_article a ON a.series_id = s.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		WHERE s.user_id = ? AND s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'
		GROUP BY s.id ORDER BY s.update_time DESC, s.id DESC LIMIT ? OFFSET ?`, userID, limit, offset).Find(&series); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.series.list", err)
	}
	return series, count, nil
}

func (r *MyPlatformRepo) ListTopicArticles(ctx context.Context, topic, slug string, current, size int) ([]*port.ArticleCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.topic.articles")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	join := ""
	where := ""
	switch topic {
	case "category":
		join = ` JOIN t_category c ON a.category_id = c.id`
		where = ` AND lower(btrim(c.category_name)) = lower(btrim(?))`
	case "tag":
		join = ` JOIN t_article_tag at ON at.article_id = a.id JOIN t_tag t ON t.id = at.tag_id`
		where = ` AND lower(btrim(t.tag_name)) = lower(btrim(?))`
	default:
		return nil, 0, apperrors.Invalid("platform.topic", "unsupported topic")
	}
	var count int
	if _, err := session.SQL(`SELECT count(DISTINCT a.id) FROM t_article a`+join+` WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'`+where, slug).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.topic.count", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(`
		SELECT DISTINCT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c2.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a`+join+`
		LEFT JOIN t_category c2 ON a.category_id = c2.id
		WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'`+where+`
		ORDER BY a.create_time DESC, a.id DESC LIMIT ? OFFSET ?`, slug, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.topic.list", err)
	}
	if err := r.attachArticleAuthorsAndTags(ctx, session, articles); err != nil {
		return nil, 0, err
	}
	return articles, count, nil
}

// ListTopicOverview ranks the public taxonomy surfaces by the same windowed
// hot score the feed uses, so the plaza and the trending feed agree on what
// "recently interesting" means. Every group is aggregated once and truncated
// by the given size; empty groups are dropped.
func (r *MyPlatformRepo) ListTopicOverview(ctx context.Context, size int) (port.TopicOverview, error) {
	overview := port.TopicOverview{
		Categories: []port.TopicOverviewItem{},
		Tags:       []port.TopicOverviewItem{},
		Series:     []port.TopicOverviewItem{},
	}
	session, err := repoSession(r.engine, ctx, "platform.topics.overview")
	if err != nil {
		return overview, err
	}
	windowStart, windowDate := discoveryWindow(time.Now())
	windowArgs := []interface{}{windowDate, windowStart, windowStart}
	limitArgs := func() []interface{} {
		return append(append([]interface{}{}, windowArgs...), size)
	}

	if err := session.SQL(discoveryHotScoreCTE+`
		, category_groups AS (
			SELECT lower(btrim(c.category_name)) AS group_key, min(c.id) AS id, min(c.category_name) AS name
			FROM t_category c
			GROUP BY 1
		),
		category_articles AS (
			SELECT g.group_key, a.id AS article_id
			FROM category_groups g
			JOIN t_category c ON lower(btrim(c.category_name)) = g.group_key
			JOIN t_article a ON a.category_id = c.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
			GROUP BY 1, 2
		)
		SELECT g.id, g.name, count(ca.article_id)::int AS article_count,
		       COALESCE(SUM(s.hot_score), 0)::int AS hot_score
		FROM category_groups g
		JOIN category_articles ca ON ca.group_key = g.group_key
		LEFT JOIN scored s ON s.id = ca.article_id
		GROUP BY g.group_key, g.id, g.name
		ORDER BY hot_score DESC, article_count DESC, g.name ASC
		LIMIT ?`, limitArgs()...).Find(&overview.Categories); err != nil {
		return overview, apperrors.Unavailable("platform.topics.categories", err)
	}

	if err := session.SQL(discoveryHotScoreCTE+`
		, tag_groups AS (
			SELECT lower(btrim(t.tag_name)) AS group_key, min(t.id) AS id, min(t.tag_name) AS name
			FROM t_tag t
			GROUP BY 1
		),
		tag_articles AS (
			SELECT g.group_key, a.id AS article_id
			FROM tag_groups g
			JOIN t_tag t ON lower(btrim(t.tag_name)) = g.group_key
			JOIN t_article_tag at ON at.tag_id = t.id
			JOIN t_article a ON a.id = at.article_id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
			GROUP BY 1, 2
		)
		SELECT g.id, g.name, count(ta.article_id)::int AS article_count,
		       COALESCE(SUM(s.hot_score), 0)::int AS hot_score
		FROM tag_groups g
		JOIN tag_articles ta ON ta.group_key = g.group_key
		LEFT JOIN scored s ON s.id = ta.article_id
		GROUP BY g.group_key, g.id, g.name
		ORDER BY hot_score DESC, article_count DESC, g.name ASC
		LIMIT ?`, limitArgs()...).Find(&overview.Tags); err != nil {
		return overview, apperrors.Unavailable("platform.topics.tags", err)
	}

	if err := session.SQL(discoveryHotScoreCTE+`
		, series_articles AS (
			SELECT s.id AS series_id, a.id AS article_id
			FROM t_series s
			JOIN t_article a ON a.series_id = s.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
			WHERE s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'
			GROUP BY 1, 2
		)
		SELECT s.id, s.series_name AS name, s.series_desc AS description, s.cover,
		       count(sa.article_id)::int AS article_count,
		       COALESCE(SUM(scored.hot_score), 0)::int AS hot_score
		FROM t_series s
		JOIN series_articles sa ON sa.series_id = s.id
		LEFT JOIN scored ON scored.id = sa.article_id
		WHERE s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'
		GROUP BY s.id, s.series_name, s.series_desc, s.cover
		ORDER BY hot_score DESC, article_count DESC, s.series_name ASC
		LIMIT ?`, limitArgs()...).Find(&overview.Series); err != nil {
		return overview, apperrors.Unavailable("platform.topics.series", err)
	}

	return overview, nil
}
