package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

var _ port.PlatformRepository = (*MyPlatformRepo)(nil)

type MyPlatformRepo struct{ engine *xorm.Engine }

func NewPlatformRepo(engine *xorm.Engine) *MyPlatformRepo { return &MyPlatformRepo{engine: engine} }

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
func (r *MyPlatformRepo) StudioDashboard(ctx context.Context, userID int) (port.StudioDashboard, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.dashboard")
	if err != nil {
		return port.StudioDashboard{}, err
	}
	var dashboard port.StudioDashboard
	queries := []struct {
		sql    string
		target *int
	}{
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0`, &dashboard.ArticleCount},
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 3`, &dashboard.DraftCount},
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 2`, &dashboard.PrivateCount},
		{`SELECT count(1) FROM t_talk WHERE user_id = ?`, &dashboard.TalkCount},
		{`SELECT count(1) FROM t_series WHERE user_id = ? AND is_delete = 0`, &dashboard.SeriesCount},
		{`SELECT count(1) FROM t_article_reaction WHERE user_info_id = ? AND reaction = 'favorite'`, &dashboard.FavoriteCount},
		{`SELECT count(1) FROM t_user_follow follow JOIN t_user_info follower ON follower.id = follow.follower_id AND follower.is_disable = 0 WHERE follow.author_id = ?`, &dashboard.FollowerCount},
		{`SELECT count(1) FROM t_user_follow follow JOIN t_user_info author ON author.id = follow.author_id AND author.is_disable = 0 WHERE follow.follower_id = ?`, &dashboard.FollowingCount},
	}
	for _, item := range queries {
		if _, err := session.SQL(item.sql, userID).Get(item.target); err != nil {
			return port.StudioDashboard{}, apperrors.Unavailable("platform.studio.dashboard", err)
		}
	}
	activation, err := loadStudioActivation(session, userID)
	if err != nil {
		return port.StudioDashboard{}, err
	}
	dashboard.Activation = activation
	return dashboard, nil
}

type studioActivationRow struct {
	UserId              int        `xorm:"user_id"`
	Collapsed           int        `xorm:"collapsed"`
	StartedAt           *time.Time `xorm:"started_at"`
	IdentityCompletedAt *time.Time `xorm:"identity_completed_at"`
	ContentCompletedAt  *time.Time `xorm:"content_completed_at"`
	ProfileVisitedAt    *time.Time `xorm:"profile_visited_at"`
	CompletedAt         *time.Time `xorm:"completed_at"`
}

func emptyStudioActivation() port.StudioActivation {
	return port.StudioActivation{}
}

func activationTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func studioActivationFromRow(row studioActivationRow) port.StudioActivation {
	return port.StudioActivation{
		StartedAt:           activationTime(row.StartedAt),
		Collapsed:           row.Collapsed != 0,
		IdentityCompletedAt: activationTime(row.IdentityCompletedAt),
		ContentCompletedAt:  activationTime(row.ContentCompletedAt),
		ProfileVisitedAt:    activationTime(row.ProfileVisitedAt),
		CompletedAt:         activationTime(row.CompletedAt),
	}
}

func activationTimeValue(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return *value
}

func loadStudioActivation(session *xorm.Session, userID int) (port.StudioActivation, error) {
	var row studioActivationRow
	found, err := session.SQL(`
		SELECT user_id, collapsed, started_at, identity_completed_at,
		       content_completed_at, profile_visited_at, completed_at
		FROM t_studio_activation
		WHERE user_id = ?
	`, userID).Get(&row)
	if err != nil {
		return emptyStudioActivation(), apperrors.Unavailable("platform.studio.activation.get", err)
	}
	if !found {
		return emptyStudioActivation(), nil
	}
	return studioActivationFromRow(row), nil
}

func (r *MyPlatformRepo) SyncStudioActivation(ctx context.Context, userID int, update port.StudioActivationUpdate) (port.StudioActivation, error) {
	var result port.StudioActivation
	err := repoTx(r.engine, ctx, "platform.studio.activation.sync", func(session *xorm.Session) error {
		collapsed := 0
		if update.Collapsed {
			collapsed = 1
		}
		if _, err := session.Exec(`
			INSERT INTO t_studio_activation (user_id, collapsed)
			VALUES (?, ?)
			ON CONFLICT (user_id) DO NOTHING
		`, userID, collapsed); err != nil {
			return apperrors.Unavailable("platform.studio.activation.sync", err)
		}
		var row studioActivationRow
		found, err := session.SQL(`
			SELECT user_id, collapsed, started_at, identity_completed_at,
			       content_completed_at, profile_visited_at, completed_at
			FROM t_studio_activation
			WHERE user_id = ?
			FOR UPDATE
		`, userID).Get(&row)
		if err != nil {
			return apperrors.Unavailable("platform.studio.activation.sync", err)
		}
		if !found {
			return apperrors.Unavailable("platform.studio.activation.sync", nil)
		}
		now := time.Now().UTC()
		if update.Started && row.StartedAt == nil {
			row.StartedAt = &now
		}
		if update.IdentityComplete && row.IdentityCompletedAt == nil {
			row.IdentityCompletedAt = &now
		}
		if update.ContentComplete && row.ContentCompletedAt == nil {
			row.ContentCompletedAt = &now
		}
		if update.ProfileVisited && row.ProfileVisitedAt == nil {
			row.ProfileVisitedAt = &now
		}
		if (update.Completed || (update.IdentityComplete && update.ContentComplete && update.ProfileVisited)) && row.CompletedAt == nil {
			row.CompletedAt = &now
		}
		row.Collapsed = collapsed
		if _, err := session.Exec(`
			UPDATE t_studio_activation
			SET collapsed = ?, started_at = ?, identity_completed_at = ?,
			    content_completed_at = ?, profile_visited_at = ?, completed_at = ?,
			    update_time = CURRENT_TIMESTAMP
			WHERE user_id = ?
		`, row.Collapsed, activationTimeValue(row.StartedAt), activationTimeValue(row.IdentityCompletedAt),
			activationTimeValue(row.ContentCompletedAt), activationTimeValue(row.ProfileVisitedAt),
			activationTimeValue(row.CompletedAt), userID); err != nil {
			return apperrors.Unavailable("platform.studio.activation.sync", err)
		}
		result = studioActivationFromRow(row)
		return nil
	})
	return result, err
}

func (r *MyPlatformRepo) CreateDueStudioActivationReminders(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	session, err := repoSession(r.engine, ctx, "platform.studio.activation_reminders")
	if err != nil {
		return 0, err
	}
	result, err := session.Exec(`
		INSERT INTO t_user_notification (
			recipient_id, actor_id, type, content_type, content_id, comment_id,
			dedupe_key, title, excerpt, action_url
		)
		SELECT
			activation.user_id,
			NULL,
			'studio_activation',
			'studio',
			0,
			0,
			'studio-activation:' || CASE WHEN activation.started_at <= ? THEN '72h' ELSE '24h' END,
			CASE
				WHEN activation.identity_completed_at IS NULL THEN '完成公开身份'
				WHEN activation.content_completed_at IS NULL THEN '写下第一条内容'
				ELSE '预览你的公开主页'
			END,
			CASE
				WHEN activation.identity_completed_at IS NULL THEN '补齐 Handle、头像、昵称和简介，让公开主页可以访问。'
				WHEN activation.content_completed_at IS NULL THEN '写一篇文章或发布一条随想，完成你的第一次表达。'
				ELSE '检查主页在公共空间里的最终呈现，完成创作者激活。'
			END,
			CASE
				WHEN activation.identity_completed_at IS NULL THEN '/studio/profile'
				WHEN activation.content_completed_at IS NULL THEN '/studio/dashboard#activation'
				WHEN recipient.handle <> '' THEN '/u/' || recipient.handle
				ELSE '/studio/profile'
			END
		FROM t_studio_activation activation
		JOIN t_user_info recipient ON recipient.id = activation.user_id
			AND recipient.is_disable = 0 AND recipient.notify_studio_activation = 1
		WHERE activation.started_at IS NOT NULL
		  AND activation.completed_at IS NULL
		  AND activation.started_at <= ?
		  AND (activation.identity_completed_at IS NULL
		       OR activation.content_completed_at IS NULL
		       OR activation.profile_visited_at IS NULL)
		ORDER BY activation.started_at ASC
		LIMIT ?
		ON CONFLICT (recipient_id, dedupe_key) DO NOTHING
	`, now.Add(-72*time.Hour), now.Add(-24*time.Hour), limit)
	if err != nil {
		return 0, apperrors.Unavailable("platform.studio.activation_reminders", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Unavailable("platform.studio.activation_reminders", err)
	}
	return int(affected), nil
}

func (r *MyPlatformRepo) GetStudioProfile(ctx context.Context, userID int) (port.StudioProfile, error) {
	session, err := repoSession(r.engine, ctx, "platform.profile.get")
	if err != nil {
		return port.StudioProfile{}, err
	}
	var user entity.TUserInfo
	found, err := session.ID(userID).Get(&user)
	if err != nil {
		return port.StudioProfile{}, apperrors.Unavailable("platform.profile.get", err)
	}
	if !found {
		return port.StudioProfile{}, apperrors.NotFound("platform.profile.get")
	}
	return port.StudioProfile{
		Handle: user.Handle, Nickname: user.Nickname, Avatar: user.Avatar,
		Intro: user.Intro, Website: user.Website, About: user.About,
		Links: decodeProfileLinks(user.ProfileLinksJSON),
	}, nil
}

func (r *MyPlatformRepo) UpdateAuthorProfile(ctx context.Context, userID int, profile port.StudioProfile) error {
	profile.Handle = strings.ToLower(strings.TrimSpace(profile.Handle))
	profile.Nickname = strings.TrimSpace(profile.Nickname)
	if !validPublicHandle(profile.Handle) {
		return apperrors.Invalid("platform.profile.handle", "handle is invalid")
	}
	if profile.Nickname == "" {
		return apperrors.Invalid("platform.profile.nickname", "nickname is required")
	}
	linksJSON, err := json.Marshal(profile.Links)
	if err != nil {
		return apperrors.Invalid("platform.profile.links", "links could not be encoded")
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var existing entity.TUserInfo
		found, err := session.Where("lower(handle) = lower(?) AND id <> ?", profile.Handle, userID).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.profile.handle.unique", err)
		}
		if found {
			return apperrors.Conflict("platform.profile.handle", "handle already exists")
		}
		affected, err := session.ID(userID).Cols("handle", "nickname", "intro", "website", "about", "profile_links_json").Update(&entity.TUserInfo{
			Handle: profile.Handle, Nickname: profile.Nickname, Intro: profile.Intro,
			Website: profile.Website, About: profile.About, ProfileLinksJSON: string(linksJSON),
		})
		if err != nil {
			return apperrors.Unavailable("platform.profile.update", err)
		}
		if affected == 0 {
			return apperrors.NotFound("platform.profile.update")
		}
		return nil
	})
}

func validPublicHandle(value string) bool {
	if len(value) < 3 || len(value) > 40 {
		return false
	}
	for index, r := range value {
		if index == 0 && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}
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
	if topic == "category" {
		join = ` JOIN t_category c ON a.category_id = c.id`
		where = ` AND lower(btrim(c.category_name)) = lower(btrim(?))`
	} else if topic == "tag" {
		join = ` JOIN t_article_tag at ON at.article_id = a.id JOIN t_tag t ON t.id = at.tag_id`
		where = ` AND lower(btrim(t.tag_name)) = lower(btrim(?))`
	} else {
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
func (r *MyPlatformRepo) ListOwnedArticles(ctx context.Context, userID int, filter port.StudioFilter) ([]*port.ArticleAdmin, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.articles")
	if err != nil {
		return nil, 0, err
	}
	where, args := ownedFilter(userID, filter, "a")
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_article a WHERE `+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.articles.count", err)
	}
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	pageArgs := append(append([]interface{}{}, args...), limit, offset)
	order := "a.update_time DESC NULLS LAST, a.id DESC"
	if filter.SeriesID > 0 {
		order = "a.series_order ASC, a.id ASC"
	} else if filter.Status == 4 {
		order = "a.scheduled_at ASC NULLS LAST, a.id ASC"
	}
	var articles []*port.ArticleAdmin
	if err := session.SQL(`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, a.series_id, a.series_order, a.scheduled_at,
		       a.is_top, a.is_featured, a.is_delete, a.status, a.moderation_status, a.moderation_reason,
		       a.type, a.create_time, c.category_name
		FROM t_article a LEFT JOIN t_category c ON a.category_id = c.id
		WHERE `+where+` ORDER BY `+order+` LIMIT ? OFFSET ?`, pageArgs...).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.articles.list", err)
	}
	return articles, count, nil
}

func (r *MyPlatformRepo) GetOwnedArticle(ctx context.Context, userID, articleID int) (port.ArticleAdminView, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.article")
	if err != nil {
		return port.ArticleAdminView{}, err
	}
	var article entity.TArticle
	found, err := session.Where("id = ? AND user_id = ?", articleID, userID).Get(&article)
	if err != nil {
		return port.ArticleAdminView{}, apperrors.Unavailable("platform.studio.article.get", err)
	}
	if !found {
		return port.ArticleAdminView{}, apperrors.NotFound("platform.studio.article.get")
	}
	categoryName := ""
	if article.CategoryId > 0 {
		var category entity.TCategory
		if _, err := session.Where("id = ? AND user_id = ?", article.CategoryId, userID).Get(&category); err == nil {
			categoryName = category.CategoryName
		}
	}
	var tagIDs []int
	if err := session.SQL(`SELECT tag_id FROM t_article_tag WHERE article_id = ?`, articleID).Find(&tagIDs); err != nil {
		return port.ArticleAdminView{}, apperrors.Unavailable("platform.studio.article.tags", err)
	}
	scheduledAt := ""
	if !article.ScheduledAt.IsZero() {
		scheduledAt = article.ScheduledAt.Format(time.RFC3339)
	}
	return port.ArticleAdminView{
		Id: article.Id, UserId: article.UserId, ArticleCover: article.ArticleCover, ArticleTitle: article.ArticleTitle,
		ArticleContent: article.ArticleContent, ArticleContentHTML: article.ArticleContentHTML, IsTop: article.IsTop,
		IsFeatured: article.IsFeatured, CategoryId: article.CategoryId, CategoryName: categoryName, TagNames: tagIDs, Status: article.Status,
		Type: article.Type, Password: article.Password, OriginalUrl: article.OriginalUrl, SeriesId: article.SeriesId,
		SeriesOrder: article.SeriesOrder, ScheduledAt: scheduledAt,
		ModerationStatus: article.ModerationStatus, ModerationReason: article.ModerationReason,
	}, nil
}

func (r *MyPlatformRepo) SaveOwnedArticle(ctx context.Context, userID int, article entity.TArticle, categoryID int, tagIDs []int) (entity.TArticle, error) {
	wasPublic := false
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if categoryID > 0 {
			ok, err := session.Where("id = ? AND user_id = ?", categoryID, userID).Exist(&entity.TCategory{})
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.category", err)
			}
			if !ok {
				return apperrors.Invalid("platform.studio.article.category", "category does not belong to author")
			}
			article.CategoryId = categoryID
		} else {
			article.CategoryId = 0
		}
		if article.SeriesId > 0 {
			ok, err := session.Where("id = ? AND user_id = ? AND is_delete = 0", article.SeriesId, userID).Exist(&entity.TSeries{})
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.series", err)
			}
			if !ok {
				return apperrors.Invalid("platform.studio.article.series", "series does not belong to author")
			}
		}
		uniqueTags := uniqueInts(tagIDs)
		if len(uniqueTags) > 0 {
			count, err := session.Where("user_id = ?", userID).In("id", uniqueTags).Count(&entity.TTag{})
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.tags", err)
			}
			if int(count) != len(uniqueTags) {
				return apperrors.Invalid("platform.studio.article.tags", "tag does not belong to author")
			}
		}
		article.UserId = userID
		if article.Status == 0 {
			article.Status = 3
		}
		if article.Id == 0 {
			if article.ModerationStatus == "" {
				article.ModerationStatus = "visible"
			}
			if _, err := session.Insert(&article); err != nil {
				return apperrors.Unavailable("platform.studio.article.create", err)
			}
		} else {
			var existing entity.TArticle
			found, err := session.Where("id = ? AND user_id = ?", article.Id, userID).Get(&existing)
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.owner", err)
			}
			if !found {
				return apperrors.NotFound("platform.studio.article.update")
			}
			wasPublic = existing.Status == 1 && existing.IsDelete == 0 && existing.ModerationStatus == "visible"
			if _, err := session.ID(article.Id).Where("user_id = ?", userID).Cols(
				"category_id", "article_cover", "article_title", "article_content", "article_content_html",
				"series_id", "series_order", "scheduled_at", "status", "type", "password", "original_url", "update_time",
			).Update(&article); err != nil {
				return apperrors.Unavailable("platform.studio.article.update", err)
			}
		}
		if _, err := session.Where("article_id = ?", article.Id).Delete(&entity.TArticleTag{}); err != nil {
			return apperrors.Unavailable("platform.studio.article.tags.clear", err)
		}
		for _, tagID := range uniqueTags {
			if _, err := session.Insert(&entity.TArticleTag{ArticleId: article.Id, TagId: tagID}); err != nil {
				return apperrors.Unavailable("platform.studio.article.tags.save", err)
			}
		}
		if !wasPublic {
			if err := recordArticlePublishEvent(session, article.Id, time.Now()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return entity.TArticle{}, err
	}
	return r.getArticleRecord(ctx, article.Id)
}

func (r *MyPlatformRepo) getArticleRecord(ctx context.Context, id int) (entity.TArticle, error) {
	session, err := repoSession(r.engine, ctx, "platform.article.record")
	if err != nil {
		return entity.TArticle{}, err
	}
	var article entity.TArticle
	found, err := session.ID(id).Get(&article)
	if err != nil {
		return entity.TArticle{}, apperrors.Unavailable("platform.article.record", err)
	}
	if !found {
		return entity.TArticle{}, apperrors.NotFound("platform.article.record")
	}
	return article, nil
}

func (r *MyPlatformRepo) DeleteOwnedArticles(ctx context.Context, userID int, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		for _, id := range uniqueInts(ids) {
			found, err := session.Where("id = ? AND user_id = ?", id, userID).Exist(&entity.TArticle{})
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.delete.owner", err)
			}
			if !found {
				return apperrors.NotFound("platform.studio.article.delete")
			}
			if _, err := session.Where("article_id = ?", id).Delete(&entity.TArticleTag{}); err != nil {
				return apperrors.Unavailable("platform.studio.article.delete.tags", err)
			}
			if _, err := session.Where("id = ?", id).Delete(&entity.TArticle{}); err != nil {
				return apperrors.Unavailable("platform.studio.article.delete", err)
			}
		}
		return nil
	})
}

func (r *MyPlatformRepo) ListOwnedTalks(ctx context.Context, userID int, filter port.StudioFilter) ([]*port.TalkAdmin, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.talks")
	if err != nil {
		return nil, 0, err
	}
	where, args := ownedFilter(userID, filter, "t")
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_talk t WHERE `+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.talks.count", err)
	}
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	pageArgs := append(append([]interface{}{}, args...), limit, offset)
	var talks []*port.TalkAdmin
	if err := session.SQL(`
		SELECT t.id, t.user_id, ui.nickname, ui.avatar, t.content, t.images, t.is_top, t.status,
		       t.moderation_status, t.moderation_reason, t.create_time
		FROM t_talk t JOIN t_user_info ui ON t.user_id = ui.id
		WHERE `+where+` ORDER BY t.id DESC LIMIT ? OFFSET ?`, pageArgs...).Find(&talks); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.talks.list", err)
	}
	return talks, count, nil
}

func (r *MyPlatformRepo) GetOwnedTalk(ctx context.Context, userID, talkID int) (entity.TTalk, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.talk")
	if err != nil {
		return entity.TTalk{}, err
	}
	var talk entity.TTalk
	found, err := session.Where("id = ? AND user_id = ?", talkID, userID).Get(&talk)
	if err != nil {
		return entity.TTalk{}, apperrors.Unavailable("platform.studio.talk.get", err)
	}
	if !found {
		return entity.TTalk{}, apperrors.NotFound("platform.studio.talk.get")
	}
	return talk, nil
}

func (r *MyPlatformRepo) SaveOwnedTalk(ctx context.Context, talk entity.TTalk) (entity.TTalk, error) {
	wasPublic := false
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if talk.Status == 0 {
			talk.Status = 3
		}
		if talk.Id == 0 {
			if talk.ModerationStatus == "" {
				talk.ModerationStatus = "visible"
			}
			if _, err := session.Insert(&talk); err != nil {
				return apperrors.Unavailable("platform.studio.talk.create", err)
			}
			return recordTalkPublishEvent(session, talk.Id, time.Now())
		}
		var existing entity.TTalk
		found, err := session.Where("id = ? AND user_id = ?", talk.Id, talk.UserId).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.studio.talk.owner", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.talk.update")
		}
		wasPublic = existing.Status == 1 && existing.ModerationStatus == "visible"
		if _, err := session.ID(talk.Id).Where("user_id = ?", talk.UserId).Cols(
			"content", "images", "is_top", "status", "update_time",
		).Update(&talk); err != nil {
			return apperrors.Unavailable("platform.studio.talk.update", err)
		}
		if !wasPublic {
			if err := recordTalkPublishEvent(session, talk.Id, time.Now()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return entity.TTalk{}, err
	}
	return talk, nil
}
func (r *MyPlatformRepo) DeleteOwnedTalks(ctx context.Context, userID int, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	for _, id := range uniqueInts(ids) {
		if _, err := r.GetOwnedTalk(ctx, userID, id); err != nil {
			return err
		}
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.Where("user_id = ?", userID).In("id", uniqueInts(ids)).Delete(&entity.TTalk{}); err != nil {
			return apperrors.Unavailable("platform.studio.talks.delete", err)
		}
		return nil
	})
}

func (r *MyPlatformRepo) ListOwnedSeries(ctx context.Context, userID int, filter port.StudioFilter) ([]*port.Series, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.series")
	if err != nil {
		return nil, 0, err
	}
	where, args := ownedFilter(userID, filter, "s")
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_series s WHERE `+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.series.count", err)
	}
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	pageArgs := append(append([]interface{}{}, args...), limit, offset)
	var series []*port.Series
	if err := session.SQL(`
		SELECT s.id, s.user_id, s.series_name, s.series_desc, s.cover, s.status, s.moderation_status, s.moderation_reason, s.update_time,
		       count(a.id) AS article_count
		FROM t_series s LEFT JOIN t_article a ON a.series_id = s.id AND a.is_delete = 0
		WHERE `+where+` GROUP BY s.id ORDER BY s.update_time DESC, s.id DESC LIMIT ? OFFSET ?`, pageArgs...).Find(&series); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.series.list", err)
	}
	return series, count, nil
}

func (r *MyPlatformRepo) GetOwnedSeries(ctx context.Context, userID, seriesID int) (entity.TSeries, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.series.get")
	if err != nil {
		return entity.TSeries{}, err
	}
	var series entity.TSeries
	found, err := session.Where("id = ? AND user_id = ? AND is_delete = 0", seriesID, userID).Get(&series)
	if err != nil {
		return entity.TSeries{}, apperrors.Unavailable("platform.studio.series.get", err)
	}
	if !found {
		return entity.TSeries{}, apperrors.NotFound("platform.studio.series.get")
	}
	return series, nil
}

func (r *MyPlatformRepo) SaveOwnedSeries(ctx context.Context, series entity.TSeries) (entity.TSeries, error) {
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if series.Id == 0 {
			if series.Status == 0 {
				series.Status = 1
			}
			if series.ModerationStatus == "" {
				series.ModerationStatus = "visible"
			}
			if _, err := session.Insert(&series); err != nil {
				return apperrors.Unavailable("platform.studio.series.create", err)
			}
			return nil
		}
		var existing entity.TSeries
		found, err := session.Where("id = ? AND user_id = ?", series.Id, series.UserId).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.studio.series.owner", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.series.update")
		}
		if _, err := session.ID(series.Id).Where("user_id = ?", series.UserId).Cols(
			"series_name", "series_desc", "cover", "status", "update_time",
		).Update(&series); err != nil {
			return apperrors.Unavailable("platform.studio.series.update", err)
		}
		return nil
	})
	if err != nil {
		return entity.TSeries{}, err
	}
	return series, nil
}

func (r *MyPlatformRepo) DeleteOwnedSeries(ctx context.Context, userID, seriesID int) error {
	if _, err := r.GetOwnedSeries(ctx, userID, seriesID); err != nil {
		return err
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.Exec(`UPDATE t_article SET series_id = NULL, series_order = 0 WHERE series_id = ? AND user_id = ?`, seriesID, userID); err != nil {
			return apperrors.Unavailable("platform.studio.series.detach", err)
		}
		if _, err := session.Where("id = ? AND user_id = ?", seriesID, userID).Cols("is_delete", "update_time").Update(&entity.TSeries{IsDelete: 1, UpdateTime: time.Now()}); err != nil {
			return apperrors.Unavailable("platform.studio.series.delete", err)
		}
		return nil
	})
}

func questionMarks(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func (r *MyPlatformRepo) ListOwnedCategories(ctx context.Context, userID int) ([]*port.Category, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.categories")
	if err != nil {
		return nil, err
	}
	var categories []*port.Category
	if err := session.SQL(`
		SELECT c.id, c.user_id, c.category_name, count(a.id) AS article_count
		FROM t_category c LEFT JOIN t_article a ON a.category_id = c.id AND a.is_delete = 0
		WHERE c.user_id = ? GROUP BY c.id ORDER BY c.category_name`, userID).Find(&categories); err != nil {
		return nil, apperrors.Unavailable("platform.studio.categories.list", err)
	}
	return categories, nil
}

func (r *MyPlatformRepo) SaveOwnedCategory(ctx context.Context, category entity.TCategory) (entity.TCategory, error) {
	if strings.TrimSpace(category.CategoryName) == "" {
		return entity.TCategory{}, apperrors.Invalid("platform.studio.category", "name is required")
	}
	category.CategoryName = strings.TrimSpace(category.CategoryName)
	var result entity.TCategory
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var existing entity.TCategory
		found, err := session.Where("user_id = ? AND lower(btrim(category_name)) = lower(btrim(?))", category.UserId, category.CategoryName).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.studio.category.unique", err)
		}
		if found && existing.Id != category.Id {
			return apperrors.Conflict("platform.studio.category", "category already exists")
		}
		if category.Id == 0 {
			if _, err := session.Insert(&category); err != nil {
				return apperrors.Unavailable("platform.studio.category.create", err)
			}
			result = category
			return nil
		}
		var current entity.TCategory
		found, err = session.Where("id = ? AND user_id = ?", category.Id, category.UserId).Get(&current)
		if err != nil {
			return apperrors.Unavailable("platform.studio.category.owner", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.category")
		}
		if _, err := session.ID(category.Id).Where("user_id = ?", category.UserId).Cols("category_name").Update(&category); err != nil {
			return apperrors.Unavailable("platform.studio.category.update", err)
		}
		result = category
		return nil
	})
	return result, err
}

func (r *MyPlatformRepo) DeleteOwnedCategory(ctx context.Context, userID, categoryID int) error {
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		count, err := session.Where("category_id = ? AND user_id = ? AND is_delete = 0", categoryID, userID).Count(&entity.TArticle{})
		if err != nil {
			return apperrors.Unavailable("platform.studio.category.in_use", err)
		}
		if count > 0 {
			return apperrors.Conflict("platform.studio.category.delete", "category is in use")
		}
		affected, err := session.Where("id = ? AND user_id = ?", categoryID, userID).Delete(&entity.TCategory{})
		if err != nil {
			return apperrors.Unavailable("platform.studio.category.delete", err)
		}
		if affected == 0 {
			return apperrors.NotFound("platform.studio.category.delete")
		}
		return nil
	})
}

func (r *MyPlatformRepo) ListOwnedTags(ctx context.Context, userID int) ([]*port.Tag, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.tags")
	if err != nil {
		return nil, err
	}
	var tags []*port.Tag
	if err := session.SQL(`
		SELECT t.id, t.user_id, t.tag_name, count(at.article_id) AS count
		FROM t_tag t
		LEFT JOIN t_article_tag at ON at.tag_id = t.id
		LEFT JOIN t_article a ON a.id = at.article_id AND a.is_delete = 0
		WHERE t.user_id = ? GROUP BY t.id ORDER BY t.tag_name`, userID).Find(&tags); err != nil {
		return nil, apperrors.Unavailable("platform.studio.tags.list", err)
	}
	return tags, nil
}

func (r *MyPlatformRepo) SaveOwnedTag(ctx context.Context, tag entity.TTag) (entity.TTag, error) {
	if strings.TrimSpace(tag.TagName) == "" {
		return entity.TTag{}, apperrors.Invalid("platform.studio.tag", "name is required")
	}
	tag.TagName = strings.TrimSpace(tag.TagName)
	var result entity.TTag
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var existing entity.TTag
		found, err := session.Where("user_id = ? AND lower(btrim(tag_name)) = lower(btrim(?))", tag.UserId, tag.TagName).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.studio.tag.unique", err)
		}
		if found && existing.Id != tag.Id {
			return apperrors.Conflict("platform.studio.tag", "tag already exists")
		}
		if tag.Id == 0 {
			if _, err := session.Insert(&tag); err != nil {
				return apperrors.Unavailable("platform.studio.tag.create", err)
			}
			result = tag
			return nil
		}
		var current entity.TTag
		found, err = session.Where("id = ? AND user_id = ?", tag.Id, tag.UserId).Get(&current)
		if err != nil {
			return apperrors.Unavailable("platform.studio.tag.owner", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.tag")
		}
		if _, err := session.ID(tag.Id).Where("user_id = ?", tag.UserId).Cols("tag_name").Update(&tag); err != nil {
			return apperrors.Unavailable("platform.studio.tag.update", err)
		}
		result = tag
		return nil
	})
	return result, err
}

func (r *MyPlatformRepo) DeleteOwnedTag(ctx context.Context, userID, tagID int) error {
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		count, err := session.Where("tag_id = ?", tagID).Count(&entity.TArticleTag{})
		if err != nil {
			return apperrors.Unavailable("platform.studio.tag.in_use", err)
		}
		if count > 0 {
			return apperrors.Conflict("platform.studio.tag.delete", "tag is in use")
		}
		affected, err := session.Where("id = ? AND user_id = ?", tagID, userID).Delete(&entity.TTag{})
		if err != nil {
			return apperrors.Unavailable("platform.studio.tag.delete", err)
		}
		if affected == 0 {
			return apperrors.NotFound("platform.studio.tag.delete")
		}
		return nil
	})
}

func (r *MyPlatformRepo) ModerateContent(ctx context.Context, contentType string, id, adminID int, hidden bool, reason string) error {
	table := ""
	switch contentType {
	case "article":
		table = "t_article"
	case "talk":
		table = "t_talk"
	case "series":
		table = "t_series"
	case "collection":
		table = "t_collection"
	default:
		return apperrors.Invalid("platform.moderation", "unsupported content type")
	}
	status := "visible"
	reason = strings.TrimSpace(reason)
	recommendationReset := ""
	if hidden {
		status = "hidden"
		switch table {
		case "t_article":
			recommendationReset = ", is_top = 0, is_featured = 0"
		case "t_talk":
			recommendationReset = ", is_top = 0"
		}
	} else {
		reason = ""
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		result, err := session.Exec("UPDATE "+table+" SET moderation_status = ?, moderation_reason = ?, moderated_by = ?, moderated_at = CURRENT_TIMESTAMP"+recommendationReset+" WHERE id = ?", status, reason, adminID, id)
		if err != nil {
			return apperrors.Unavailable("platform.moderation.update", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return apperrors.NotFound("platform.moderation.update")
		}
		if !hidden {
			switch contentType {
			case "article":
				return recordArticlePublishEvent(session, id, time.Now())
			case "talk":
				return recordTalkPublishEvent(session, id, time.Now())
			}
		}
		return nil
	})
}

func ownedFilter(userID int, filter port.StudioFilter, alias string) (string, []interface{}) {
	where := alias + ".user_id = ?"
	args := []interface{}{userID}
	if alias == "a" || alias == "s" {
		where += " AND " + alias + ".is_delete = 0"
	}
	if filter.Status > 0 {
		where += " AND " + alias + ".status = ?"
		args = append(args, filter.Status)
	}
	if alias == "a" && filter.SeriesID > 0 {
		where += " AND " + alias + ".series_id = ?"
		args = append(args, filter.SeriesID)
	}
	if strings.TrimSpace(filter.Keywords) != "" {
		column := alias + ".article_title"
		if alias == "t" {
			column = alias + ".content"
		} else if alias == "s" {
			column = alias + ".series_name"
		}
		where += " AND " + column + " LIKE ?"
		args = append(args, "%"+strings.TrimSpace(filter.Keywords)+"%")
	}
	return where, args
}

func uniqueInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func toPublicAuthor(user entity.TUserInfo) port.PublicAuthor {
	return port.PublicAuthor{
		Id:       user.Id,
		Handle:   user.Handle,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Intro:    user.Intro,
		Website:  user.Website,
		About:    user.About,
		Links:    decodeProfileLinks(user.ProfileLinksJSON),
	}
}

func decodeProfileLinks(raw string) []port.ProfileLink {
	var links []port.ProfileLink
	if raw == "" || json.Unmarshal([]byte(raw), &links) != nil {
		return []port.ProfileLink{}
	}
	return links
}
