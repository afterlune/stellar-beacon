package repository

import (
	"context"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

func (r *MyPlatformRepo) GetAuthorByHandle(ctx context.Context, handle string, viewerID int) (port.AuthorCard, error) {
	session, err := repoSession(r.engine, ctx, "platform.author")
	if err != nil {
		return port.AuthorCard{}, err
	}
	var user entity.TUserInfo
	found, err := session.SQL(`SELECT * FROM t_user_info WHERE lower(handle) = lower(?) AND is_disable = 0`, strings.TrimSpace(handle)).Get(&user)
	if err != nil {
		return port.AuthorCard{}, apperrors.Unavailable("platform.author.get", err)
	}
	if !found {
		return port.AuthorCard{}, apperrors.NotFound("platform.author.get")
	}
	author := port.AuthorCard{PublicAuthor: toPublicAuthor(user)}
	if err := r.attachAuthorCounts(session, &author, viewerID); err != nil {
		return port.AuthorCard{}, err
	}
	return author, nil
}

func (r *MyPlatformRepo) attachAuthorCounts(session *xorm.Session, author *port.AuthorCard, viewerID int) error {
	if author == nil || author.Id <= 0 {
		return nil
	}
	if _, err := session.SQL(`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, author.Id).Get(&author.ArticleCount); err != nil {
		return apperrors.Unavailable("platform.author.article_count", err)
	}
	if _, err := session.SQL(`SELECT count(1) FROM t_talk WHERE user_id = ? AND status = 1 AND moderation_status = 'visible'`, author.Id).Get(&author.TalkCount); err != nil {
		return apperrors.Unavailable("platform.author.talk_count", err)
	}
	if _, err := session.SQL(`SELECT count(1) FROM t_series WHERE user_id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, author.Id).Get(&author.SeriesCount); err != nil {
		return apperrors.Unavailable("platform.author.series_count", err)
	}
	if _, err := session.SQL(`SELECT count(1) FROM t_collection c WHERE c.user_id = ? AND c.visibility = 'public' AND c.moderation_status = 'visible' AND c.is_delete = 0 AND EXISTS (SELECT 1 FROM t_collection_item i JOIN t_article a ON a.id = i.article_id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible' WHERE i.collection_id = c.id)`, author.Id).Get(&author.CollectionCount); err != nil {
		return apperrors.Unavailable("platform.author.collection_count", err)
	}
	if _, err := session.SQL(`
		SELECT count(1) FROM t_user_follow follow
		JOIN t_user_info follower ON follower.id = follow.follower_id AND follower.is_disable = 0
		WHERE follow.author_id = ?`, author.Id).Get(&author.FollowerCount); err != nil {
		return apperrors.Unavailable("platform.author.follower_count", err)
	}
	if viewerID > 0 && viewerID != author.Id {
		if _, err := session.SQL(`SELECT EXISTS (SELECT 1 FROM t_user_follow WHERE follower_id = ? AND author_id = ?)`, viewerID, author.Id).Get(&author.IsFollowing); err != nil {
			return apperrors.Unavailable("platform.author.following", err)
		}
	}
	return nil
}

func (r *MyPlatformRepo) ListAuthors(ctx context.Context, current, size, viewerID int, sort string) ([]*port.AuthorCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.authors")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`
		SELECT count(1) FROM t_user_info ui
		WHERE ui.is_disable = 0 AND (
			EXISTS (
				SELECT 1 FROM t_article a
				WHERE a.user_id = ui.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
			) OR EXISTS (
				SELECT 1 FROM t_talk t
				WHERE t.user_id = ui.id AND t.status = 1 AND t.moderation_status = 'visible'
			) OR EXISTS (
				SELECT 1 FROM t_series s
				WHERE s.user_id = ui.id AND s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'
			)
		)`).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.authors.count", err)
	}
	order := discoveryAuthorArticleCountSQL + ` DESC, ui.id ASC`
	switch sort {
	case port.AuthorSortFollowers:
		order = `(
			SELECT count(1) FROM t_user_follow follow
			JOIN t_user_info follower ON follower.id = follow.follower_id AND follower.is_disable = 0
			WHERE follow.author_id = ui.id
		) DESC, ` + discoveryAuthorArticleCountSQL + ` DESC, ui.id ASC`
	case port.AuthorSortActive:
		order = discoveryAuthorActivitySQL + ` DESC NULLS LAST, ui.id ASC`
	}
	var users []entity.TUserInfo
	if err := session.SQL(`
		SELECT ui.*
		FROM t_user_info ui
		WHERE ui.is_disable = 0 AND (
			EXISTS (
				SELECT 1 FROM t_article a
				WHERE a.user_id = ui.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
			) OR EXISTS (
				SELECT 1 FROM t_talk t
				WHERE t.user_id = ui.id AND t.status = 1 AND t.moderation_status = 'visible'
			) OR EXISTS (
				SELECT 1 FROM t_series s
				WHERE s.user_id = ui.id AND s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'
			)
		)
		ORDER BY `+order+` LIMIT ? OFFSET ?`, limit, offset).Find(&users); err != nil {
		return nil, 0, apperrors.Unavailable("platform.authors.list", err)
	}
	authors := make([]*port.AuthorCard, 0, len(users))
	for _, user := range users {
		author := &port.AuthorCard{PublicAuthor: toPublicAuthor(user)}
		if err := r.attachAuthorCounts(session, author, viewerID); err != nil {
			return nil, 0, err
		}
		authors = append(authors, author)
	}
	if sort == port.AuthorSortActive {
		if err := r.attachAuthorActivity(session, authors); err != nil {
			return nil, 0, err
		}
	}
	return authors, count, nil
}

// discoveryHotWindowDays is the trailing window every public ranking surface
// aggregates over. It is a constant on purpose: the API exposes no window
// parameter, so a client can never trigger an unbounded table aggregation.
const discoveryHotWindowDays = 7

// discoveryScoredCTE holds the windowed aggregates without the leading WITH
// keyword so another query can compose its own CTEs in front of them. Callers
// must bind three arguments: metric date, reaction time and comment time.
const discoveryScoredCTE = `
	window_metrics AS (
		SELECT m.article_id, SUM(m.unique_readers) AS readers
		FROM t_article_daily_metric m
		WHERE m.metric_date >= ?
		GROUP BY m.article_id
	),
	window_reactions AS (
		SELECT r.article_id,
		       count(1) FILTER (WHERE r.reaction = 'favorite') AS favorites,
		       count(1) FILTER (WHERE r.reaction = 'like') AS likes
		FROM t_article_reaction r
		WHERE r.create_time >= ?
		GROUP BY r.article_id
	),
	window_comments AS (
		SELECT c.topic_id AS article_id, count(1) AS comments
		FROM t_comment c
		WHERE c.type = 1 AND c.is_review = 1 AND c.is_delete = 0 AND c.create_time >= ?
		GROUP BY c.topic_id
	),
	scored AS (
		SELECT a.id,
		       5 * COALESCE(wr.favorites, 0) + 4 * COALESCE(wc.comments, 0)
		       + 3 * COALESCE(wr.likes, 0) + 2 * COALESCE(wm.readers, 0) AS hot_score
		FROM t_article a
		LEFT JOIN window_reactions wr ON wr.article_id = a.id
		LEFT JOIN window_comments wc ON wc.article_id = a.id
		LEFT JOIN window_metrics wm ON wm.article_id = a.id
		WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
	)
`

// discoveryHotScoreCTE is the standalone form used by the feed queries.
const discoveryHotScoreCTE = "\n\tWITH " + discoveryScoredCTE

// discoveryAuthorArticleCountSQL is the public-article count used to rank the
// author board. It resolves the surrounding query alias "ui".
const discoveryAuthorArticleCountSQL = `(
			SELECT count(1) FROM t_article a
			WHERE a.user_id = ui.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		)`

// discoveryAuthorActivitySQL is the "most recent public publish" timestamp
// behind the active-author ranking. Publish events whose content was later
// hidden, deleted or made private no longer count as activity.
const discoveryAuthorActivitySQL = `(
			SELECT max(event.published_at) FROM t_author_publish_event event
			WHERE event.author_id = ui.id AND (
				(event.content_type = 'article' AND EXISTS (
					SELECT 1 FROM t_article a
					WHERE a.id = event.content_id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
				)) OR (event.content_type = 'talk' AND EXISTS (
					SELECT 1 FROM t_talk t
					WHERE t.id = event.content_id AND t.status = 1 AND t.moderation_status = 'visible'
				))
			)
		)`

// discoveryWindow returns the start of the ranking window as a timestamp for
// row-level columns and as a date for the daily metric table.
func discoveryWindow(now time.Time) (time.Time, string) {
	start := now.AddDate(0, 0, -discoveryHotWindowDays)
	return start, start.Format("2006-01-02")
}

// attachAuthorActivity fills LastPublishedAt for a page of authors with one
// grouped query, so the active ranking never degrades into a per-author lookup.
func (r *MyPlatformRepo) attachAuthorActivity(session *xorm.Session, authors []*port.AuthorCard) error {
	ids := make([]int, 0, len(authors))
	for _, author := range authors {
		if author != nil && author.Id > 0 {
			ids = append(ids, author.Id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var rows []struct {
		AuthorID        int       `xorm:"author_id"`
		LastPublishedAt time.Time `xorm:"last_published_at"`
	}
	query := `
		SELECT event.author_id, max(event.published_at) AS last_published_at
		FROM t_author_publish_event event
		WHERE event.author_id IN (` + placeholders(len(ids)) + `) AND (
			(event.content_type = 'article' AND EXISTS (
				SELECT 1 FROM t_article a
				WHERE a.id = event.content_id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
			)) OR (event.content_type = 'talk' AND EXISTS (
				SELECT 1 FROM t_talk t
				WHERE t.id = event.content_id AND t.status = 1 AND t.moderation_status = 'visible'
			))
		)
		GROUP BY event.author_id`
	if err := session.SQL(query, intArgs(ids)...).Find(&rows); err != nil {
		return apperrors.Unavailable("platform.authors.activity", err)
	}
	activity := make(map[int]time.Time, len(rows))
	for _, row := range rows {
		activity[row.AuthorID] = row.LastPublishedAt
	}
	for _, author := range authors {
		if author == nil {
			continue
		}
		if at, ok := activity[author.Id]; ok {
			published := at
			author.LastPublishedAt = &published
		}
	}
	return nil
}
